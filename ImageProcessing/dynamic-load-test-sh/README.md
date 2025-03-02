# Vegeta Load Testing for Knative Eventing with RabbitMQ

This script performs load testing on a Knative-based image processing system using Vegeta, a powerful HTTP load testing tool. Instead of sending actual images, the script dynamically generates requests containing random image URLs from Lorem Picsum, ensuring realistic and diverse testing.

The requests are sent to a RabbitMQSource, which automatically converts them into CloudEvents before forwarding them to the RabbitMQ broker. From there, a custom load balancer picks them up and routes them to Knative consumer services running the image-processing function.

## How It Works
- Generates a new random image URL for each request (using https://picsum.photos/200/300).
- Formats the request as JSON ({ "image_url": "<URL>" }).
- Sends requests at a defined rate to the RabbitMQSource endpoint, which turns them into CloudEvents.
- The Knative eventing system routes the events through the RabbitMQ broker to the custom load balancer.
- The load balancer forwards the events to consuming services, which fetch, process, and analyze the images.

## Usage
### Running the Load Test
```
./vegeta-dynamic-load-test.sh [RATE] [DURATION]
```

Example:
```
./vegeta-dynamic-load-test.sh 100 10m
```
This sends 100 requests per second for 10 minutes.

**Default Values:**
- **RATE**: 50 requests per second.
- **DURATION**: 5m (5 minutes).
_If you don’t provide values, it defaults to 50 RPS for 5 minutes._

## Monitoring Performance
### View the Vegeta Summary Report
```
cat results.bin | vegeta report
```

### Generate a Latency Histogram
```
cat results.bin | vegeta report -type='hist[0,2ms,4ms,6ms,10ms]'
```

### Graph Results with a Plot
```
cat results.bin | vegeta plot > plot.html && xdg-open plot.html
```
