package events

import (
	"context"
	"log"
	"rate-controller/config"
	"rate-controller/controller"
	"rate-controller/metrics"
	"strconv"
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	rateController *controller.RateController
	rdbClient      *redis.Client
	requestBuffer  = make(chan []byte, 100) // Buffer for storing incoming requests
	wg             sync.WaitGroup
	bufferCond     = sync.NewCond(&sync.Mutex{})
)

// Subscribe to the Redis channel for admission rate updates for the specific service.
func SubscribeToAdmissionRate(rdb *redis.Client) {
	serviceName := config.ServiceName
	pubSub := rdb.Subscribe(context.Background(), "admission_rate:"+serviceName)

	// Defer closing pubSub to ensure graceful shutdown
	defer pubSub.Close()

	// Listen for admission rate updates
	ch := pubSub.Channel()
	for msg := range ch {
		admissionRateStr := msg.Payload
		admissionRate, err := strconv.ParseFloat(admissionRateStr, 64)
		if err != nil {
			log.Printf("⚠️ Error parsing admission rate: %v", err)
			continue
		}

		log.Printf("🚀 Received new admission rate for %s: %f", serviceName, admissionRate)

		// Update the rate controller's limiter with the new admission rate
		rateController.UpdateAdmissionRateFromRedis(admissionRate)

		// Update the Prometheus metric with the new admission rate
		metrics.UpdateMetric(admissionRate)
		log.Printf("✅ Updated admission rate for %s to %f", serviceName, admissionRate)
	}
}

// ProcessBufferedRequests listens for requests in the buffer and forwards them at the rate defined by the controller
func ProcessBufferedRequests() {
	for {
		bufferCond.L.Lock()
		for len(requestBuffer) == 0 {
			log.Printf("⏳ Waiting for requests in the buffer...")
			bufferCond.Wait() // Wait until a new request is added
		}
		requestBody := <-requestBuffer
		bufferCond.L.Unlock()

		log.Printf("🚚 Processing request from the buffer...")
		rateController.ForwardRequest(requestBody)
	}
}

// AddRequestToBuffer adds an incoming request to the buffer for processing
func AddRequestToBuffer(requestBody []byte) {
	bufferCond.L.Lock()
	log.Printf("📥 Adding request to buffer. Buffer size before adding: %d", len(requestBuffer))
	requestBuffer <- requestBody
	bufferCond.Signal() // Signal the processor that a new request is available
	bufferCond.L.Unlock()
}

// Initialize the rate controller
func InitRateController(alpha, beta float64) {
	log.Printf("🔧 Initializing RateController with alpha: %f, beta: %f", alpha, beta)
	rateController = controller.NewRateController(alpha, beta)
}

// StartReceiver subscribes to the Redis channel for the specific service's admission rate and starts processing requests.
func StartReceiver() {
	log.Printf("🔌 Connecting to Redis at %s", config.RedisURL)
	rdbClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass, // Redis password from env
		DB:       0,
	})

	// Test Redis connection
	pong, err := rdbClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}
	log.Printf("✅ Redis connection successful: %s", pong)

	// Subscribe to admission rate updates for the specific service
	go SubscribeToAdmissionRate(rdbClient)

	// Start processing buffered requests
	go ProcessBufferedRequests()
}
