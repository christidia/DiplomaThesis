package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"load-balancer/config"
	"load-balancer/events"
	"load-balancer/rabbitmq"
	"load-balancer/redis"
	"load-balancer/routing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ctx = context.Background()
)

func main() {
	// Load configurations
	config.LoadConfig()

	// Initialize Redis client
	rdb := redis.NewRedisClient()

	// Initialize services and other components
	redis.InitializeServices(rdb)
	go events.StartReceiver()
	go routing.StartAdmissionRateUpdater(rdb)

	// Find the queue name with the specified prefix
	queueName, err := rabbitmq.FindQueueWithPrefix("rabbitmq-setup.event-trigger.")
	if err != nil {
		log.Fatalf("❌ Error finding queue: %v", err)
	}
	if queueName == "" {
		log.Fatalf("❌ Queue with prefix 'rabbitmq-setup.event-trigger.' not found")
	}
	log.Printf("🔍 Found queue with name: %s", queueName)

	// Set up RabbitMQ connection and channel
	conn, ch, err := rabbitmq.SetupRabbitMQ()
	if err != nil {
		log.Fatalf("❌ Failed to setup RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Create a channel to signal termination
	done := make(chan bool)

	// Start the AMQP channel to continuously poll the queue
	go func() {
		rabbitmq.PollQueue(queueName, ch, done)
	}()

	// Start HTTP server for Prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("🚀 Metrics server is running on port 2112")
		if err := http.ListenAndServe(":2112", nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Could not listen on port 2112: %v\n", err)
		}
	}()

	// Handle graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	log.Println("📴 Received termination signal, shutting down gracefully...")

	// Signal the polling goroutine to stop
	done <- true
	close(done)
	log.Println("🛑 Application stopped")

	// Block indefinitely to keep the program running
	select {}
}
