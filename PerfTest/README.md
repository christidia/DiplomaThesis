# Running Load Testing Experiments using the RabbitMQ Performance Test

The experiments can be ran using the script provided in this directory.

## Prerequisites
* Kubernetes CLI (kubectl)
* Java Runtime Environment (JRE)
* jq (Command-line JSON processor)
* The Knative and Rabbitmq system set up and running

## Running a Test

### Set Up the Script
Update the script with the correct path to your home directory and the location of the perf-test.jar file.

```bash
# perf-test-script.sh

# Set this variable to point to the path where the repo is saved
HOME="/home/yourusername"

# Ensure the path to the PerfTest jar is correct
PERF_TEST_JAR="$HOME/DiplomaThesis/PerfTest/perf-test.jar"

```

### Make the script executable

```bash
chmod +x perf-test-script.sh
```

### Run the script

```bash
./perf-test-script.sh
```

## Script Breakdown
* **start_port_forwarding:** Starts port forwarding from the RabbitMQ server to the local machine.
* **cleanup:** Ensures port forwarding is stopped when the script exits.
* **RabbitMQ Host and Port:** Defines the RabbitMQ host and port for the connection.
* **Fetch RabbitMQ Secret:** Retrieves and decodes RabbitMQ credentials from Kubernetes secrets.
* **Test Parameters:** Sets various parameters for the performance test, such as message rate, producer count, body content type, and exchange name.
* **Run PerfTest:** Executes the RabbitMQ performance test using the provided parameters.
