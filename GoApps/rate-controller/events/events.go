package events

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"rate-controller/config"
	"rate-controller/controller"
	"rate-controller/metrics"
	"strconv"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-redis/redis/v8"
)

var (
	rateController *controller.RateController
	rdbClient      *redis.Client
	httpClient     = &http.Client{}
)

// Subscribe to the Redis channel for admission rate updates for the specific service.
func SubscribeToAdmissionRate(rdb *redis.Client) {
	serviceName := config.ServiceName
	pubSub := rdb.Subscribe(context.Background(), "admission_rate:"+serviceName)

	defer pubSub.Close() // Defer closing pubSub to ensure graceful shutdown

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

		// Update the rate controller
		rateController.UpdateAdmissionRateFromRedis(admissionRate)

		// Update the Prometheus metric with the new admission rate
		metrics.UpdateMetric(admissionRate)
		log.Printf("✅ Updated admission rate for %s to %f", serviceName, admissionRate)
	}
}

// HandleEvent processes incoming CloudEvents and forwards them to the consuming service with rate-limiting applied.
func HandleEvent(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	// Apply the rate limit when receiving the event
	if err := rateController.Limiter.Wait(ctx); err != nil {
		log.Printf("❌ Error applying rate limit: %v", err)
		return cloudevents.NewHTTPResult(http.StatusTooManyRequests, "Rate limit exceeded")
	}

	// Forward the CloudEvent to the consuming service
	return forwardEventToService(ctx, event)
}

// forwardEventToService forwards the CloudEvent to the configured service URL.
func forwardEventToService(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	// Convert the CloudEvent to a byte array
	eventBytes, err := event.MarshalJSON()
	if err != nil {
		log.Printf("❌ Error marshalling CloudEvent: %v", err)
		return cloudevents.NewHTTPResult(http.StatusInternalServerError, "Error marshalling CloudEvent")
	}

	// Create an HTTP POST request with the CloudEvent data
	req, err := http.NewRequestWithContext(ctx, "POST", config.ServiceURL, bytes.NewBuffer(eventBytes))
	if err != nil {
		log.Printf("❌ Error creating HTTP request: %v", err)
		return cloudevents.NewHTTPResult(http.StatusInternalServerError, "Error creating HTTP request")
	}

	// Set the appropriate headers for the CloudEvent
	req.Header.Set("Content-Type", cloudevents.ApplicationJSON) // or cloudevents.ApplicationCloudEventsJSON
	req.Header.Set("Ce-Specversion", event.SpecVersion())
	req.Header.Set("Ce-Id", event.ID())
	req.Header.Set("Ce-Source", event.Source())
	req.Header.Set("Ce-Type", event.Type())

	// Forward the request to the consuming service
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Error forwarding CloudEvent to %s: %v", config.ServiceURL, err)
		return cloudevents.NewHTTPResult(http.StatusInternalServerError, "Error forwarding CloudEvent")
	}
	defer resp.Body.Close()

	log.Printf("✅ Successfully forwarded CloudEvent to %s, response code: %d", config.ServiceURL, resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return cloudevents.ResultACK
	}
	return cloudevents.NewHTTPResult(resp.StatusCode, "Failed to forward CloudEvent")
}

// Initialize the rate controller
func InitRateController(alpha, beta float64) {
	rateController = controller.NewRateController(alpha, beta)
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
