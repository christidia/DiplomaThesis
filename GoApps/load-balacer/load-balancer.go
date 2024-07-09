package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-redis/redis/v8"
)

type CloudEventData struct {
	ImageData string `json:"imageData"`
	Message   string `json:"message,omitempty"`
}

type Service struct {
	Name         string
	CurrWeight   int
	EmptyQWeight int
	Beta         float64
	Alpha        int
}

var (
	admissionRatesMutex   sync.Mutex
	servicesMap           map[string]*Service
	emptyQueueEventSent   bool
	messageCounter        int
	messageRateThreshold  int = 1
	serviceNames          []string
	lastUpdateTime        time.Time
	redisURL              string
	redisPass             string
	ctx                   = context.Background()
	idleTimeoutDuration   = 5 * time.Second // Idle timeout duration variable
	admissionRateInterval = 5 * time.Second // Interval for updating admission rates
	serviceKeyPrefix      = "service:"
	tkKey                 = "tk"
	emptyQueue            = false
	prevQueueEmpty        = false // Flag to check if the queue was empty in the previous check
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

	// Additive Increase (AI) phase
	elapsedTime := time.Since(lastUpdateTime).Seconds()
	if elapsedTime >= 1.0 {
		for _, service := range servicesMap {
			// Increase the current weight by Beta * EmptyQWeight + Alpha * elapsedTime
			service.CurrWeight = int(float64(service.Beta)*float64(service.EmptyQWeight)) + service.Alpha*int(elapsedTime)
			err := rdb.HSet(ctx, serviceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
			}
		}
		lastUpdateTime = currentTime
	}

	// Log updated admission rates
	for _, service := range servicesMap {
		log.Printf("📈 Updated admission rate for %s: %d", service.Name, service.CurrWeight)
	}
}

// Send event to one of the services based on admission rates
func sendToEventDisplays(rdb *redis.Client, event cloudevents.Event) {
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

// Receive and process incoming CloudEvents
func receive(rdb *redis.Client, event cloudevents.Event, currentTime time.Time) {
	sendToEventDisplays(rdb, event)
	prevQueueEmpty = false // Reset flag as the queue is no longer empty
}

// Start the CloudEvent receiver and monitor the queue
func startReceiver(rdb *redis.Client) {
	rand.Seed(time.Now().UnixNano())
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	idleTimer := time.NewTimer(idleTimeoutDuration)

	go func() {
		for {
			select {
			case <-idleTimer.C:
				// Check if there are no messages and trigger empty queue event
				createEmptyQueueEvent(rdb, time.Now())
				idleTimer.Reset(idleTimeoutDuration)
			}
		}
	}()

	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		currentTime := time.Now()
		receive(rdb, event, currentTime)

		// Reset idle timer
		if !idleTimer.Stop() {
			<-idleTimer.C
		}
		idleTimer.Reset(idleTimeoutDuration)
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

func main() {
	rdb := newRedisClient()

	// Initialize 'tk' value in Redis
	err := initializeTkIfNotExists(rdb)
	if err != nil {
		log.Fatalf("❌ Error initializing value of 'tk' in Redis: %v", err)
	}
	fmt.Println("✅ Value of 'tk' initialized in Redis")

	initializeServices(rdb)
	go startReceiver(rdb)
	go startAdmissionRateUpdater(rdb)

	// Block indefinitely to keep the program running
	select {}
}
