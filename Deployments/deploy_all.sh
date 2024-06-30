#!/bin/bash

# Base deployment directory
BASE_DEPLOYMENT_DIR="$HOME/DiplomaThesis/Deployments"

# Function to check if a Kubernetes pod is ready
check_pod_ready() {
  local pod_name="$1"
  local namespace="$2"

  echo "Checking if pod $pod_name is ready in namespace $namespace..."
  while true; do
    # Check if the pod is found and in Running state
    POD_STATUS=$(kubectl get pods -n "$namespace" --field-selector=status.phase=Running | grep "$pod_name")
    if [[ $? -eq 0 ]]; then
      READY_STATUS=$(kubectl get pod "$pod_name" -n "$namespace" -o jsonpath='{.status.containerStatuses[*].ready}')
      if [[ "$READY_STATUS" == "true" ]]; then
        echo "Pod $pod_name is ready."
        break
      else
        echo "Waiting for pod $pod_name to be ready..."
      fi
    else
      echo "Waiting for pod $pod_name to be in Running state..."
    fi
    sleep 5
  done
}

# Deploy RabbitMQ cluster first
kubectl apply -f "$BASE_DEPLOYMENT_DIR/RabbitMQ/rabbitmq-cluster.yaml"
check_pod_ready "rabbitmq-server-0" "rabbitmq-setup"

# Deploy broker-config
kubectl apply -f "$BASE_DEPLOYMENT_DIR/RabbitMQ/broker-config.yaml"
echo "broker-config is created."

# Deploy rabbitmq-broker
kubectl apply -f "$BASE_DEPLOYMENT_DIR/RabbitMQ/rabbitmq-broker.yaml"
#check_pod_ready "queue-broker-ingress" "rabbitmq-setup"

# Deploy rabbitmq-event-trigger
kubectl apply -f "$BASE_DEPLOYMENT_DIR/RabbitMQ/rabbitmq-event-trigger.yaml"
#check_pod_ready "event-trigger-dispatcher" "rabbitmq-setup"

# Deploy rabbitmq-source
kubectl apply -f "$BASE_DEPLOYMENT_DIR/RabbitMQ/rabbitmq-source.yaml"
#check_pod_ready "rabbitmqsource-rabbitmq-source" "rabbitmq-setup"

: << END
# Deploy load balancer
kubectl apply -f "$BASE_DEPLOYMENT_DIR/loadbalancer.yaml"
check_pod_ready "loadbalancer" "rabbitmq-setup" # Adjust pod name as needed

# Deploy services
kubectl apply -f "$BASE_DEPLOYMENT_DIR/services.yaml"
check_pod_ready "service1-00001-deployment" "rabbitmq-setup"
check_pod_ready "service2-00001-deployment" "rabbitmq-setup"
check_pod_ready "service3-00001-deployment" "rabbitmq-setup"
END 

echo "All resources have been applied."

