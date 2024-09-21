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
	for requestBody := range requestBuffer {
		rateController.ForwardRequest(requestBody) // Forward each request at the rate limit
	}
}

// AddRequestToBuffer adds an incoming request to the buffer for processing
func AddRequestToBuffer(requestBody []byte) {
	requestBuffer <- requestBody
}

// Initialize the rate controller
func InitRateController(alpha, beta float64) {
	rateController = controller.NewRateController(alpha, beta)
}

// StartReceiver subscribes to the Redis channel for the specific service's admission rate and starts processing requests.
func StartReceiver() {
	rdbClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass, // No password set
		DB:       0,                // Use default DB
	})

	// Subscribe to admission rate updates for the specific service
	go SubscribeToAdmissionRate(rdbClient)

	// Start processing buffered requests
	go ProcessBufferedRequests()
}
