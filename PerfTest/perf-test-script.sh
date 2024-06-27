#!/bin/bash

set -e

# Set this variable to point to the path where the repo is saved
HOME="/home/ubuntu"

# Function to start port forwarding and set up cleanup on exit
start_port_forwarding() {
  kubectl -n rabbitmq-setup port-forward rabbitmq-server-0 5672:5672 &

  PORT_FORWARD_PID=$!

  # Allow some time for port forwarding to be set up
  sleep 5
}

# Function to clean up port forwarding
cleanup() {
  if [ -n "$PORT_FORWARD_PID" ]; then
    echo "Cleaning up port forwarding..."
    kill $PORT_FORWARD_PID
  fi
}

# Trap script exit and interrupt signals to ensure cleanup is called
trap cleanup EXIT
trap cleanup INT

# Start port forwarding
start_port_forwarding

# Define RabbitMQ host and port
RABBITMQ_HOST="localhost"
RABBITMQ_PORT=5672

echo "RabbitMQ Host: $RABBITMQ_HOST"
echo "RabbitMQ Port: $RABBITMQ_PORT"

# Fetch the RabbitMQ secret from Kubernetes
RABBITMQ_SECRET=$(kubectl get secrets rabbitmq-default-user -n rabbitmq-setup -o json)
echo "RabbitMQ Secret JSON: $RABBITMQ_SECRET" # Debug statement to print the secret JSON

# Extract the default_user.conf content
DEFAULT_USER_CONF=$(echo $RABBITMQ_SECRET | jq -r '.data["default_user.conf"]' | base64 -d)
echo "Default User Config: $DEFAULT_USER_CONF" # Debug statement to print the decoded config

# Extract username and password from the key-value format
RABBITMQ_USERNAME=$(echo "$DEFAULT_USER_CONF" | grep 'default_user' | cut -d '=' -f2 | xargs)
RABBITMQ_PASSWORD=$(echo "$DEFAULT_USER_CONF" | grep 'default_pass' | cut -d '=' -f2 | xargs)

# Print the username and password for debugging
echo "RabbitMQ Username: $RABBITMQ_USERNAME"
echo "RabbitMQ Password: $RABBITMQ_PASSWORD"

# Define other variables for the test
RABBITMQ_VHOST="/"

# Test parameters
MESSAGE_RATE=10
PRODUCER_COUNT=2
BODY_DIR="$HOME/DiplomaThesis/GoApps/cloudevents/payloads"
BODY_CONTENT_TYPE="application/json"
EXCHANGE_NAME="eventing-rabbitmq-source"
MESSAGE_TYPE="headers"

# Construct BODY_PATH by joining all JSON files in the directory with commas
BODY_PATH=$(ls ${BODY_DIR}/*.json 2>/dev/null | paste -sd "," -)

# Example output of BODY_PATH for demonstration purposes
echo "Constructed BODY_PATH: $BODY_PATH"

# Ensure that the BODY_PATH is not empty
if [ -z "$BODY_PATH" ]; then
  echo "No JSON files found in the directory: $BODY_DIR"
  exit 1
fi

# Ensure the path to the PerfTest jar is correct
PERF_TEST_JAR="$HOME/DiplomaThesis/PerfTest/perf-test.jar"

# Check if the PerfTest jar file exists
if [ ! -f "$PERF_TEST_JAR" ]; then
  echo "PerfTest jar file not found at: $PERF_TEST_JAR"
  exit 1
fi

# Run PerfTest with the constructed BODY_PATH
java -jar $PERF_TEST_JAR \
  --uri amqp://$RABBITMQ_USERNAME:$RABBITMQ_PASSWORD@$RABBITMQ_HOST:$RABBITMQ_PORT \
  --body $BODY_PATH \
  --body-content-type $BODY_CONTENT_TYPE \
  --rate $MESSAGE_RATE \
  --producers $PRODUCER_COUNT \
  --exchange $EXCHANGE_NAME \
  --type $MESSAGE_TYPE

# Cleanup port forwarding
cleanup
