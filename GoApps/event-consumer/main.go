package main

import (
	"context"
	"log"
	"sync"

	"consumer/cloudevents"
	"consumer/config"
	"consumer/processor"
)

func main() {
	run(context.Background())
}

func run(ctx context.Context) {
	config.LoadConfig()

	// Create CloudEvents client
	c, err := cloudevents.NewClient(
		cloudevents.WithMiddleware(cloudevents.HealthzMiddleware),
		cloudevents.WithMiddleware(cloudevents.RequestLoggingMiddleware(config.RequestLoggingEnabled)),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	// Start the processor goroutines
	var wg sync.WaitGroup
	for i := 0; i < config.NumWorkers; i++ {
		wg.Add(1)
		go processor.StartProcessor(i, &wg)
	}

	// Start the receiver
	if err := c.StartReceiver(ctx, processor.Display); err != nil {
		log.Fatalf("❌ Error during receiver's runtime: %v", err)
	}

	// Wait for all workers to finish
	wg.Wait()
}
