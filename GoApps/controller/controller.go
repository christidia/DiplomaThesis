package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/streadway/amqp"
)

// Define the Service struct with a Timestamp field
type Service struct {
	Name         string
	CurrWeight   int
	EmptyQWeight int
	Beta         float64
	Alpha        int
}

var (
	ctx               = context.Background()
	rabbitMQURL       string
	rabbitMQURLhttp   string
	rabbitMQUser      string
	rabbitMQPass      string
	checkInterval     time.Duration
	isPreviouslyEmpty = true
)

func init() {
	rabbitMQURLhttp = os.Getenv("RABBITMQ_URL")
	if rabbitMQURLhttp == "" {
		log.Fatal("RABBITMQ_URL environment variable is not set")
	}
	rabbitMQUser = os.Getenv("RABBITMQ_USERNAME")
	if rabbitMQUser == "" {
		log.Fatal("RABBITMQ_USERNAME environment variable is not set")
	}
	rabbitMQPass = os.Getenv("RABBITMQ_PASSWORD")
	if rabbitMQPass == "" {
		log.Fatal("RABBITMQ_PASSWORD environment variable is not set")
	}

	// Construct RabbitMQ URL with username and password
	rabbitMQURL = fmt.Sprintf("amqp://%s:%s@rabbitmq.rabbitmq-setup.svc.cluster.local:5672/",
		rabbitMQUser, rabbitMQPass)
	log.Printf("Constructed RabbitMQ URL: %s", rabbitMQURL)

	// Set checkInterval from environment variable or use default
	checkInterval = getCheckIntervalFromEnv("CHECK_INTERVAL", 5000) * time.Millisecond
}

func getCheckIntervalFromEnv(envVar string, defaultValue int) time.Duration {
	intervalStr := os.Getenv(envVar)
	if intervalStr == "" {
		return time.Duration(defaultValue)
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		log.Printf("Invalid value for %s: %s. Using default: %dms", envVar, intervalStr, defaultValue)
		return time.Duration(defaultValue)
	}
	return time.Duration(interval)
}

func findQueueWithPrefix(prefix string) (string, error) {
	request := gorequest.New()
	// Update the URL to include the management API endpoint
	apiURL := fmt.Sprintf("http://rabbitmq.rabbitmq-setup.svc.cluster.local:15672/api/queues")
	resp, body, errs := request.Get(apiURL).
		SetBasicAuth(rabbitMQUser, rabbitMQPass).
		End()

	if len(errs) > 0 {
		return "", fmt.Errorf("Failed to get queues: %v", errs)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	var queues []Queue
	err := json.Unmarshal([]byte(body), &queues)
	if err != nil {
		return "", fmt.Errorf("Failed to parse response: %v", err)
	}

	for _, queue := range queues {
		if strings.HasPrefix(queue.Name, prefix) {
			return queue.Name, nil
		}
	}

	return "", nil // Queue with the specified prefix not found
}

func pollQueue(queueName string, ch *amqp.Channel, done chan bool) {
	log.Printf("Starting to poll queue %s", queueName)
	for {
		select {
		case <-done:
			log.Println("Stopping pollQueue goroutine")
			return
		default:
			messageCount, err := checkQueue(queueName, ch)
			if err != nil {
				log.Printf("Error checking queue: %v\n", err)
			} else {
				if messageCount == 0 && !isPreviouslyEmpty {
					log.Printf("Queue %s is now empty\n", queueName)
					isPreviouslyEmpty = true
					go updateEmptyQWeightRoutine()
				} else if messageCount > 0 {
					isPreviouslyEmpty = false
					log.Printf("Queue %s has %d messages\n", queueName, messageCount)
				}
			}
			time.Sleep(checkInterval)
		}
	}
}

func checkQueue(queueName string, ch *amqp.Channel) (int, error) {
	queue, err := ch.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue: %v", err)
	}
	return queue.Messages, nil
}

func updateEmptyQWeightRoutine() {
	log.Println("Starting updateEmptyQWeightRoutine")
	// Implement the logic for updating EmptyQWeight here
}

func main() {
	// Find the queue name with the specified prefix
	queueName, err := findQueueWithPrefix("rabbitmq-setup.event-trigger.")
	if err != nil {
		log.Fatalf("Error finding queue: %v", err)
	}
	if queueName == "" {
		log.Fatalf("Queue with prefix 'rabbitmq-setup.event-trigger.' not found")
	}
	log.Printf("Found queue with name: %s", queueName)

	// Set up RabbitMQ connection and channel
	conn, ch, err := setupRabbitMQ()
	if err != nil {
		log.Fatalf("Failed to setup RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Create a channel to signal termination
	done := make(chan bool)

	// Start the AMQP channel to continuously poll the queue
	go func() {
		pollQueue(queueName, ch, done)
	}()

	// Handle graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	log.Println("Received termination signal, shutting down gracefully...")

	// Signal the polling goroutine to stop
	done <- true
	close(done)
}

// Function to set up a persistent RabbitMQ connection and channel
func setupRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	return conn, ch, nil
}

// Define the Queue struct
type Queue struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
}
