package main

import (
	"os"
	"strconv"

	"rate-controller/config"
	"rate-controller/events"
	"rate-controller/metrics"
)

func main() {
	config.LoadConfig()

	alpha, err := strconv.ParseFloat(os.Getenv("ALPHA"), 64)
	if err != nil {
		alpha = 3
	}

	beta, err := strconv.ParseFloat(os.Getenv("BETA"), 64)
	if err != nil {
		beta = 0.5
	}

	// Initialize Rate Controller
	events.InitRateController(alpha, beta)

	// Start the metrics server
	go metrics.StartMetricsServer()

	// Start the event receiver and request processor
	events.StartReceiver()

	// Block main from exiting
	select {} // This will keep the process running
}
