package main

import (
	"rate-controller/config"
	"rate-controller/events"
	"rate-controller/metrics"
)

func main() {
	config.LoadConfig()

	// Initialize Rate Controller
	events.InitRateController()

	// Start the metrics server
	go metrics.StartMetricsServer()

	// Start the event receiver and request processor
	events.StartReceiver()

	// Block main from exiting
	select {} // This will keep the process running
}
