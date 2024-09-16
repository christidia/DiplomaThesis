package main

import (
	"admission-controller/config"
	"admission-controller/events"
	"admission-controller/metrics"
	"log"
)

func main() {
	// Load configuration from environment variables
	config.LoadConfig()

	// Initialize Rate Controller
	events.InitRateController(config.Alpha, config.Alpha)

	// Start the metrics server
	go metrics.StartMetricsServer()

	// Start the event receiver (subscribes to admission rate updates)
	events.StartReceiver()

	log.Println("🚀 Admission Controller started successfully.")
}
