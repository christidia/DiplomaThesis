#!/bin/bash

# Default Configuration
TARGET_URL="http://rabbitmq-source.rabbitmq-setup.svc.cluster.local"  # Send to RabbitMQSource
TMP_REQ="/tmp/load-test.txt"

# Default values (can be overridden by command-line arguments)
RATE=${1:-50}             # Requests per second
DURATION=${2:-5m}         # Total test duration

echo "🚀 Starting Vegeta load test with:"
echo "   🔹 RATE=$RATE requests/sec"
echo "   🔹 DURATION=$DURATION"

run_load_test() {
    while true; do
        # Generate a new random Picsum image URL for each request
        IMAGE_URL="https://picsum.photos/200/300"

        # Prepare Vegeta payload (simple JSON with image_url)
        cat <<EOF > "$TMP_REQ"
POST $TARGET_URL
Content-Type: application/json

{
  "image_url": "$IMAGE_URL"
}
EOF

        # Run Vegeta attack with the prepared request
        cat "$TMP_REQ" | vegeta attack -rate="$RATE" -duration="$DURATION" | tee results.bin | vegeta report
    done
}

# Start load test
run_load_test
