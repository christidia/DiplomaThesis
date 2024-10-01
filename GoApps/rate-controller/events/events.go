package events

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"rate-controller/config"
	"rate-controller/controller"
	"rate-controller/metrics"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-redis/redis/v8"
)

var (
	rateController *controller.RateController
	rdbClient      *redis.Client
	//httpClient     = &http.Client{}
)

// Subscribe to the Redis channel for admission rate updates for the specific service.
func SubscribeToAdmissionRate(rdb *redis.Client) {
	serviceName := config.ThisService
	pubSub := rdb.Subscribe(context.Background(), "admission_rate:"+serviceName)

	if pubSub == nil {
		log.Fatalf("❌ Failed to subscribe to admission_rate:%s channel", serviceName)
	}

	defer pubSub.Close() // Defer closing pubSub to ensure graceful shutdown

	log.Printf("🔊 Subscribed to admission_rate:%s channel", serviceName)

	// Listen for admission rate updates
	ch := pubSub.Channel()
	for msg := range ch {
		log.Printf("📡 Received message from Redis Pub/Sub channel for %s: %s", serviceName, msg.Payload)

		admissionRateStr := msg.Payload

		// Attempt to parse the admission rate
		admissionRate, err := strconv.ParseFloat(admissionRateStr, 64)
		if err != nil {
			log.Printf("⚠️ Error parsing admission rate: %v", err)
			continue
		}

		log.Printf("🚀 Received new admission rate for %s: %f", serviceName, admissionRate)

		// Update the rate controller
		rateController.UpdateAdmissionRateFromRedis(admissionRate)

		// Update the Prometheus metric with the new admission rate
		metrics.UpdateMetric(admissionRate)
		log.Printf("✅ Updated admission rate for %s to %f", serviceName, admissionRate)
	}
}

// HandleEvent processes incoming CloudEvents and forwards them to the consuming service with rate-limiting applied.
func HandleEvent(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	// Wait until the rate limiter allows us to process the event
	err := rateController.Limiter.Wait(ctx)
	if err != nil {
		log.Printf("❌ Error applying rate limit: %v", err)
		return cloudevents.NewHTTPResult(http.StatusTooManyRequests, "Rate limit exceeded")
	}

	// Forward the CloudEvent to the consuming service
	return forwardEventToService(ctx, event)
}

// forwardEventToService forwards the CloudEvent to the configured service URL.
func forwardEventToService(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	// Use CloudEvents client to handle the forwarding instead of manually creating the HTTP request
	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Printf("❌ Error creating CloudEvents client: %v", err)
		return cloudevents.NewHTTPResult(http.StatusInternalServerError, "Error creating CloudEvents client")
	}

	// Forward the event
	ctx = cloudevents.ContextWithTarget(ctx, config.ServiceURL)
	result := client.Send(ctx, event)

	if cloudevents.IsACK(result) {
		log.Printf("✅ Successfully forwarded CloudEvent to %s", config.ServiceURL)
		return cloudevents.ResultACK
	}

	log.Printf("❌ Failed to forward CloudEvent to %s", config.ServiceURL)
	return result
}

// Initialize the rate controller
func InitRateController() {
	rateController = controller.NewRateController()
}

// StartReceiver initializes the CloudEvents receiver and subscribes to admission rate updates.
func StartReceiver() {
	// Set up the Redis client to receive rate limit updates
	rdbClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass, // No password set
		DB:       0,                // Use default DB
	})

	// Subscribe to admission rate updates for the specific service
	go SubscribeToAdmissionRate(rdbClient)

	// Set up the CloudEvents client and start receiving events
	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create CloudEvents client: %v", err)
	}

	// Start receiving CloudEvents and handle them
	if err := client.StartReceiver(context.Background(), HandleEvent); err != nil {
		log.Fatalf("❌ Error starting receiver: %v", err)
	}
}
