#!/bin/bash

# Default Configuration
TARGET_URL="http://image-processing.rabbitmq-setup.svc.cluster.local/process"
BOUNDARY="BOUNDARY"
IMG_DIR="$HOME/serverless/DiplomaThesis/ImageProcessing/images"
TMP_REQ="/tmp/load-test.txt"

# Default values (can be overridden by command-line arguments)
NUM_IMAGES=${1:-10}  # Total images in rotation
REFRESH_INTERVAL=${2:-30}  # Refresh images every X seconds
REPLACE_COUNT=${3:-2}  # How many images to replace each refresh cycle
RATE=${4:-50}  # Requests per second
DURATION=${5:-5m}  # Total test duration

# Ensure image directory exists
mkdir -p $IMG_DIR

# Step 1: Download initial images
echo "📥 Downloading initial images..."
for i in $(seq 1 $NUM_IMAGES); do
    curl -s -L "https://picsum.photos/200/300" -o "$IMG_DIR/sample_$i.jpg"
done
echo "✅ Initial images ready!"

# Step 2: Background process to replace images gradually
refresh_images() {
    while true; do
        sleep $REFRESH_INTERVAL
        echo "♻️ Refreshing images..."
        for i in $(seq 1 $REPLACE_COUNT); do
            IMG_NUM=$((RANDOM % NUM_IMAGES + 1))  # Pick a random image to replace
            curl -s -L "https://picsum.photos/200/300" -o "$IMG_DIR/sample_$IMG_NUM.jpg"
            echo "🔄 Replaced sample_$IMG_NUM.jpg"
        done
    done
}

# Run the refresh function in the background
refresh_images &

# Step 3: Load Test Execution
echo "🚀 Starting Vegeta load test with:"
echo "   🔹 NUM_IMAGES=$NUM_IMAGES"
echo "   🔹 REFRESH_INTERVAL=$REFRESH_INTERVAL sec"
echo "   🔹 REPLACE_COUNT=$REPLACE_COUNT"
echo "   🔹 RATE=$RATE requests/sec"
echo "   🔹 DURATION=$DURATION"

run_load_test() {
    while true; do
        # Pick a random image from the set
        IMG_FILE=$(ls $IMG_DIR | shuf -n 1)
        FULL_PATH="$IMG_DIR/$IMG_FILE"

        # Prepare vegeta payload (multipart request with the picked image)
        cat <<EOF > $TMP_REQ
POST $TARGET_URL
Content-Type: multipart/form-data; boundary=----$BOUNDARY

------$BOUNDARY
Content-Disposition: form-data; name="file"; filename="$IMG_FILE"
Content-Type: image/jpeg

@$FULL_PATH
------$BOUNDARY--
EOF

        # Run Vegeta attack with the prepared request
        cat $TMP_REQ | vegeta attack -rate=$RATE -duration=$DURATION | tee results.bin | vegeta report
    done
}

# Start load test
run_load_test
