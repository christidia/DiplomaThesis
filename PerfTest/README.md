# Running Load Testing Experiments using the RabbitMQ Performance Test

This script, `perf_test.sh`, automates performance testing of RabbitMQ within a Knative environment. It sets up port forwarding, retrieves RabbitMQ credentials from a Kubernetes secret, and runs the PerfTest Java tool to simulate message traffic with customizable parameters.

## Features

- Configurable message rates and producer counts.
- Supports different test types (`baseline`, `scaling`, `spike`).
- Allows variable message rates for complex traffic patterns.
- Automatically retrieves RabbitMQ credentials from Kubernetes.
- Automates port forwarding for RabbitMQ in Kubernetes.

## Prerequisites

- RabbitMQ and Kubernetes setup with RabbitMQ deployed as a service.
- PerfTest tool (Java) located in the `PerfTest` directory.
- JSON payloads for testing stored in `GoApps/cloudevents/payloads`.

### Set Up the Script
Update the script with the correct path to your home directory and the location of the perf-test.jar file.

```bash
# perf_test.sh

# Set this variable to point to the path where the repo is saved
HOME="/home/yourusername"

# Ensure the path to the PerfTest jar is correct
PERF_TEST_JAR="$HOME/DiplomaThesis/PerfTest/perf-test.jar"

```

### Make the script executable

```bash
chmod +x perf-test-script.sh
```



## Usage

### 1. Fixed Message Rate Example
```bash
./perf_test.sh --rate <message_rate> --producers <count> --test-type <type>
```

**Example:**
```
./perf_test.sh --rate 5 --producers 2 --test-type baseline
```
This runs the test with a fixed message rate of 5 messages per second, using 2 producers.

### 2. Variable Message Rate Example
```
./perf_test.sh --producers <count> --test-type <type> --variable-rate <rate:duration> --variable-rate <rate:duration>
```

**Example:**
```
./perf_test.sh --producers 2 --test-type scaling --variable-rate 3:5 --variable-rate 0:30
```
This runs the test with variable message rates, sending 3 messages per second for 5 seconds, followed by a 30-second idle period.

## Command-line Parameters
--rate: Set the message rate per second (default: 6).<br>
--producers: Set the number of producers (default: 1).<br>
--test-type: Specify the test type (baseline, scaling, spike).<br>
--variable-rate: Specify variable rates using the format <rate:duration>.<br>

## How it Works
**Port Forwarding:** The script sets up port forwarding for RabbitMQ running in a Kubernetes environment.<br>
**RabbitMQ Credentials:** Credentials are fetched from a Kubernetes secret for secure access.<br>
**Payload Handling:** JSON payloads are retrieved from the specified directory and sent as message bodies during the test.<br>
**PerfTest Execution:** The script runs the PerfTest Java tool, sending messages to RabbitMQ based on the defined parameters.<br>

## Notes
Ensure that the perf-test.jar file is correctly placed in the PerfTest directory.
Ensure that the directory containing JSON payloads is correctly populated before running the test.
