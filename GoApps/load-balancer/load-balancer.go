package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-redis/redis/v8"
	"github.com/parnurzeal/gorequest"
	"github.com/streadway/amqp"
)

// Define CloudEventData struct to represent event data
type CloudEventData struct {
	ImageData string `json:"imageData"`
	Message   string `json:"message,omitempty"`
}

// Define Service struct to represent service details
type Service struct {
	Name         string
	CurrWeight   int
	EmptyQWeight int
	Beta         float64
	Alpha        int
}

// Define Queue struct to represent RabbitMQ queue details
type Queue struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
}

var (
	// Mutex to handle concurrent access to admission rates
	admissionRatesMutex sync.Mutex
	// Map to store service details
	servicesMap map[string]*Service
	// List of service names
	serviceNames []string
	// Timestamp of the last update
	lastUpdateTime time.Time
	// Redis connection parameters
	redisURL  string
	redisPass string
	// Context for Redis operations
	ctx = context.Background()
	// Interval for updating admission rates
	admissionRateInterval time.Duration
	// Redis key prefixes
	serviceKeyPrefix = "service:"
	tkKey            = "tk"
	// Flags to handle queue state
	emptyQueue     = false
	prevQueueEmpty = false // Flag to check if the queue was empty in the previous check

	// RabbitMQ connection parameters
	rabbitMQURL     string
	rabbitMQURLhttp string
	rabbitMQUser    string
	rabbitMQPass    string
	// Interval for checking queue state
	checkInterval time.Duration
)

// Initialize environment variables and configurations
func init() {
	redisURL = os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is not set")
	}
	redisPass = os.Getenv("REDIS_PASSWORD")
	if redisPass == "" {
		log.Fatal("❌ REDIS_PASSWORD environment variable is not set")
	}

	rabbitMQURLhttp = os.Getenv("RABBITMQ_URL")
	if rabbitMQURLhttp == "" {
		log.Fatal("❌ RABBITMQ_URL environment variable is not set")
	}
	rabbitMQUser = os.Getenv("RABBITMQ_USERNAME")
	if rabbitMQUser == "" {
		log.Fatal("❌ RABBITMQ_USERNAME environment variable is not set")
	}
	rabbitMQPass = os.Getenv("RABBITMQ_PASSWORD")
	if rabbitMQPass == "" {
		log.Fatal("❌ RABBITMQ_PASSWORD environment variable is not set")
	}

	// Construct RabbitMQ URL with username and password
	rabbitMQURL = fmt.Sprintf("amqp://%s:%s@rabbitmq.rabbitmq-setup.svc.cluster.local:5672/",
		rabbitMQUser, rabbitMQPass)
	log.Printf("🔗 Constructed RabbitMQ URL: %s", rabbitMQURL)

	// Set checkInterval from environment variable or use default
	checkInterval = getCheckIntervalFromEnv("CHECK_INTERVAL", 500) * time.Millisecond
	admissionRateInterval = checkInterval
	log.Printf("⏱️ Check interval and admission rate interval set to: %s", checkInterval)

	// Read the number of services from the environment variable
	numServicesStr := os.Getenv("NUM_SERVICES")
	if numServicesStr == "" {
		log.Fatal("❌ NUM_SERVICES environment variable is not set")
	}

	numServices, err := strconv.Atoi(numServicesStr)
	if err != nil {
		log.Fatalf("❌ Invalid NUM_SERVICES value: %v", err)
	}

	// Generate service names dynamically
	serviceNames = make([]string, numServices)
	for i := 0; i < numServices; i++ {
		serviceNames[i] = fmt.Sprintf("service%d", i+1)
	}
}

// Function to get check interval from environment variable
func getCheckIntervalFromEnv(envVar string, defaultValue int) time.Duration {
	intervalStr := os.Getenv(envVar)
	if intervalStr == "" {
		return time.Duration(defaultValue)
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		log.Printf("⚠️ Invalid value for %s: %s. Using default: %dms", envVar, intervalStr, defaultValue)
		return time.Duration(defaultValue)
	}
	return time.Duration(interval)
}

// Redis-related functions

// Function to create a new Redis client
func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisPass,
	})
}

// Function to check if a key exists in Redis
func keyExists(rdb *redis.Client, key string) (bool, error) {
	val, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

// Function to save a Service to Redis
func saveServiceToRedis(rdb *redis.Client, service *Service) error {
	key := serviceKeyPrefix + service.Name
	err := rdb.HSet(ctx, key, map[string]interface{}{
		"curr_weight":   service.CurrWeight,
		"emptyq_weight": service.EmptyQWeight,
		"beta":          service.Beta,
		"alpha":         service.Alpha,
	}).Err()
	return err
}

// Initialize services with specified values
func initializeServices(rdb *redis.Client) {
	servicesMap = make(map[string]*Service)
	for i, name := range serviceNames {
		service := &Service{
			Name:         name,
			CurrWeight:   10 * (i + 1),
			EmptyQWeight: 10 * (i + 1),
			Beta:         0.5,
			Alpha:        3 + i,
		}
		servicesMap[service.Name] = service
		if err := saveServiceToRedis(rdb, service); err != nil {
			log.Fatalf("❌ Error saving service %s to Redis: %v", service.Name, err)
		}
	}
	lastUpdateTime = time.Now()
	log.Println("✅ Services initialized")
}

// Initialize the value of 'tk' in Redis if it doesn't already exist
func initializeTkIfNotExists(rdb *redis.Client) error {
	exists, err := keyExists(rdb, tkKey)
	if err != nil {
		return err
	}
	if !exists {
		tk := time.Now().Add(-time.Minute).Unix() // Initialize 'tk' to the current timestamp minus an offset
		err := rdb.Set(ctx, tkKey, tk, 0).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

// Update admission rates based on AI/MD phases
func updateAdmissionRates(rdb *redis.Client, currentTime time.Time) {
	admissionRatesMutex.Lock()
	defer admissionRatesMutex.Unlock()

	// Check if the queue was empty in the last check
	if prevQueueEmpty {
		// Multiplicative Decrease (MD) phase
		for _, service := range servicesMap {
			service.CurrWeight = int(float64(service.Beta) * float64(service.EmptyQWeight))
			err := rdb.HSet(ctx, serviceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
			}
		}
		prevQueueEmpty = false // Reset the flag after applying MD
	} else {
		// Additive Increase (AI) phase
		elapsedTime := time.Since(lastUpdateTime).Seconds()
		if elapsedTime >= 0.5 { // Ensure updates are frequent
			for _, service := range servicesMap {
				// Increase the current weight by Alpha * elapsedTime
				service.CurrWeight += service.Alpha * int(elapsedTime)
				err := rdb.HSet(ctx, serviceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
				if err != nil {
					log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
				}
			}
			lastUpdateTime = currentTime
		}
	}

	// Log updated admission rates
	for _, service := range servicesMap {
		log.Printf("📈 Updated admission rate for %s: %d", service.Name, service.CurrWeight)
	}
}

// Send event to one of the services based on admission rates
func sendToEventDisplays(event cloudevents.Event) {
	admissionRatesMutex.Lock()
	defer admissionRatesMutex.Unlock()

	totalRate := 0
	for _, service := range servicesMap {
		totalRate += service.CurrWeight
	}

	randomValue := rand.Intn(totalRate)

	cumulativeRate := 0
	var destination *Service
	for _, service := range servicesMap {
		cumulativeRate += service.CurrWeight
		if randomValue < cumulativeRate {
			destination = service
			break
		}
	}

	if destination == nil {
		log.Println("❌ Error: Destination is empty. Admission rate selection failed.")
		return
	}

	destinationURL := fmt.Sprintf("http://%s.rabbitmq-setup.svc.cluster.local", destination.Name)
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = cloudevents.ContextWithTarget(ctx, destinationURL)
	log.Printf("☁️ Sending CloudEvent to %s", destination.Name)

	if result := c.Send(ctx, event); !cloudevents.IsACK(result) {
		log.Printf("❌ Failed to send: %v", result)
		return
	}

	log.Printf("✅ Successfully sent event to %s", destination.Name)
}

// Update 'tk' value in Redis and log the empty queue event
func updateTkInRedis(rdb *redis.Client, currentTime time.Time) {
	tk := currentTime.Unix()
	err := rdb.Set(ctx, tkKey, tk, 0).Err()
	if err != nil {
		log.Printf("❌ Error updating 'tk' value in Redis: %v", err)
		return
	}

	log.Printf("🗑️ Empty queue event at %s. Updated 'tk' to %d.", currentTime.Format(time.RFC3339), tk)
}

// Create and log an empty queue event
func createEmptyQueueEvent(rdb *redis.Client, currentTime time.Time) {
	if !prevQueueEmpty {
		updateTkInRedis(rdb, currentTime)
		emptyQueue = true

		admissionRatesMutex.Lock()
		defer admissionRatesMutex.Unlock()

		for _, service := range servicesMap {
			service.EmptyQWeight = service.CurrWeight
			err := rdb.HSet(ctx, serviceKeyPrefix+service.Name, "emptyq_weight", service.EmptyQWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating emptyq_weight for service %s in Redis: %v", service.Name, err)
			}
		}

		log.Printf("⚖️ Updated EmptyQWeight for all services: %v\n", servicesMap)
		prevQueueEmpty = true
	} else {
		log.Printf("ℹ️ Queue already empty, no new empty queue event triggered.")
	}
}

// CloudEvent-related functions

// Receive and process incoming CloudEvents
func receive(event cloudevents.Event) {
	sendToEventDisplays(event)
	prevQueueEmpty = false // Reset flag as the queue is no longer empty
}

// Start the CloudEvent receiver and monitor the queue
func startReceiver() {
	rand.Seed(time.Now().UnixNano())
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		receive(event)
	})
	if err != nil {
		log.Fatalf("❌ Failed to start receiver: %v", err)
	}
}

// Function to periodically update the admission rates
func startAdmissionRateUpdater(rdb *redis.Client) {
	ticker := time.NewTicker(admissionRateInterval)
	defer ticker.Stop()

	for {
		select {
		case currentTime := <-ticker.C:
			updateAdmissionRates(rdb, currentTime)
		}
	}
}

// RabbitMQ-related functions

// Function to set up a persistent RabbitMQ connection and channel
func setupRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	log.Println("🔌 Setting up RabbitMQ connection")
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("❌ failed to open a channel: %v", err)
	}

	log.Println("✅ RabbitMQ connection and channel set up successfully")
	return conn, ch, nil
}

// Function to find the queue with the specified prefix
func findQueueWithPrefix(prefix string) (string, error) {
	request := gorequest.New()
	apiURL := `http://rabbitmq.rabbitmq-setup.svc.cluster.local:15672/api/queues`
	resp, body, errs := request.Get(apiURL).
		SetBasicAuth(rabbitMQUser, rabbitMQPass).
		End()

	if len(errs) > 0 {
		return "", fmt.Errorf("❌ Failed to get queues: %v", errs)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("❌ Unexpected status code: %d", resp.StatusCode)
	}

	var queues []Queue
	err := json.Unmarshal([]byte(body), &queues)
	if err != nil {
		return "", fmt.Errorf("❌ Failed to parse response: %v", err)
	}

	for _, queue := range queues {
		if strings.HasPrefix(queue.Name, prefix) {
			return queue.Name, nil
		}
	}

	return "", nil // Queue with the specified prefix not found
}

// Function to poll the specified queue for messages
func pollQueue(queueName string, ch *amqp.Channel, done chan bool) {
	log.Printf("📡 Starting to poll queue %s", queueName)
	for {
		select {
		case <-done:
			log.Println("🛑 Stopping pollQueue goroutine")
			return
		default:
			messageCount, err := checkQueue(queueName, ch)
			if err != nil {
				log.Printf("❌ Error checking queue: %v\n", err)
			} else {
				log.Printf("📋 Queue %s has %d messages\n", queueName, messageCount)
				if messageCount == 0 && !prevQueueEmpty {
					log.Printf("📭 Queue %s is now empty\n", queueName)
					prevQueueEmpty = true
					updateEmptyQWeightRoutine()
				} else if messageCount > 0 {
					prevQueueEmpty = false
				}
			}
			time.Sleep(checkInterval)
		}
	}
}

// Function to check the number of messages in a RabbitMQ queue
func checkQueue(queueName string, ch *amqp.Channel) (int, error) {
	log.Printf("🔍 Checking queue: %s", queueName)
	queue, err := ch.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("❌ failed to inspect queue: %v", err)
	}
	return queue.Messages, nil
}

// Function to update EmptyQWeight when queue is empty
func updateEmptyQWeightRoutine() {
	log.Println("🛠️ Starting updateEmptyQWeightRoutine")
	rdb := newRedisClient()
	currentTime := time.Now()
	createEmptyQueueEvent(rdb, currentTime)
}

// Main function to start the application
func main() {
	rdb := newRedisClient()

	// Initialize 'tk' value in Redis
	err := initializeTkIfNotExists(rdb)
	if err != nil {
		log.Fatalf("❌ Error initializing value of 'tk' in Redis: %v", err)
	}
	fmt.Println("✅ Value of 'tk' initialized in Redis")

	initializeServices(rdb)
	go startReceiver()
	go startAdmissionRateUpdater(rdb)

	// Find the queue name with the specified prefix
	queueName, err := findQueueWithPrefix("rabbitmq-setup.event-trigger.")
	if err != nil {
		log.Fatalf("❌ Error finding queue: %v", err)
	}
	if queueName == "" {
		log.Fatalf("❌ Queue with prefix 'rabbitmq-setup.event-trigger.' not found")
	}
	log.Printf("🔍 Found queue with name: %s", queueName)

	// Set up RabbitMQ connection and channel
	conn, ch, err := setupRabbitMQ()
	if err != nil {
		log.Fatalf("❌ Failed to setup RabbitMQ: %v", err)
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
	log.Println("📴 Received termination signal, shutting down gracefully...")

	// Signal the polling goroutine to stop
	done <- true
	close(done)
	log.Println("🛑 Application stopped")

	// Block indefinitely to keep the program running
	select {}
}
