package events

import (
	"context"
	"log"
	"strconv"

	"admission-controller/config"
	"admission-controller/controller"
	"admission-controller/metrics"

	"github.com/go-redis/redis/v8"
)

var (
	rateController *controller.RateController
	rdbClient      *redis.Client
)

// SubscribeToAdmissionRate subscribes to the Redis channel for admission rate updates
func SubscribeToAdmissionRate(rdb *redis.Client) {
	// Get the service name from the config (ENV variable)
	serviceName := config.ServiceName
	pubSub := rdb.Subscribe(context.Background(), "admission_rate:"+serviceName)

	// Defer closing pubSub to ensure graceful shutdown
	defer pubSub.Close()

	// Listen for admission rate updates
	ch := pubSub.Channel()
	for msg := range ch {
		// Update the admission rate whenever a new message is received
		admissionRateStr := msg.Payload
		admissionRate, err := strconv.ParseFloat(admissionRateStr, 64)
		if err != nil {
			log.Printf("⚠️ Error parsing admission rate: %v", err)
			continue
		}

		log.Printf("🚀 Received new admission rate for %s: %f", serviceName, admissionRate)

		// Apply the new rate to the rate controller
		rateController.UpdateAdmissionRateFromRedis(admissionRate)

		// Update the Prometheus metric with the new admission rate
		metrics.UpdateMetric(rateController.GetAdmissionRate())
		log.Printf("✅ Updated admission rate for %s to %f", serviceName, admissionRate)
	}
}

// Initialize the rate controller
func InitRateController(alpha, beta float64) {
	rateController = controller.NewRateController(alpha, beta)
}

// StartReceiver subscribes to the Redis channel for admission rate updates
func StartReceiver() {
	// Initialize Redis client
	rdbClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass,
		DB:       0, // use default DB
	})

	// Subscribe to admission rate changes for the specific service
	SubscribeToAdmissionRate(rdbClient)
}
