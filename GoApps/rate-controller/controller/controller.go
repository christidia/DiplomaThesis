package controller

import (
	"context"
	"encoding/base64"
	"log"
	"sync"

	"rate-controller/config"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"golang.org/x/time/rate"
)

type RateController struct {
	mu            sync.Mutex
	admissionRate float64
	Limiter       *rate.Limiter
	Client        cloudevents.Client // CloudEvents client to send requests
}

// NewRateController initializes a new RateController with the given alpha and beta values
func NewRateController(alpha, beta float64) *RateController {
	initialRate := 1.0 // Initialize with a default admission rate
	log.Printf("🔧 Creating RateLimiter with initial rate: %f", initialRate)

	// Create a CloudEvents HTTP client
	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create CloudEvents client: %v", err)
	}

	return &RateController{
		admissionRate: initialRate,
		Limiter:       rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
		Client:        client,                                      // Initialize the CloudEvents client
	}
}

// ForwardRequest forwards the buffered request to the service URL as a CloudEvent based on the current admission rate
func (rc *RateController) ForwardRequest(requestBody []byte) {
	// Wait until allowed by the rate limiter
	err := rc.Limiter.Wait(context.Background())
	if err != nil {
		log.Printf("⚠️ Error waiting on rate limiter: %v", err)
		return
	}

	log.Printf("⏩ Forwarding request to %s as CloudEvent", config.ServiceURL)

	// Encode the image data (assuming it's in the request body) in base64
	encodedImageData := base64.StdEncoding.EncodeToString(requestBody)

	// Create a CloudEvent with the expected data structure
	event := cloudevents.NewEvent()
	event.SetSource("rate-controller")
	event.SetType("com.example.ratecontroller.image")
	event.SetData(cloudevents.ApplicationJSON, map[string]string{
		"imageData": encodedImageData,
	})

	// Set CloudEvent headers and target URL
	ctx := cloudevents.ContextWithTarget(context.Background(), config.ServiceURL)

	// Send the event
	result := rc.Client.Send(ctx, event)
	if cloudevents.IsUndelivered(result) {
		log.Printf("❌ Failed to send CloudEvent: %v", result)
	} else {
		log.Printf("✅ CloudEvent successfully sent to %s, result: %v", config.ServiceURL, result)
	}
}

// UpdateAdmissionRateFromRedis updates the admission rate and rate limiter based on the value received from Redis
func (rc *RateController) UpdateAdmissionRateFromRedis(newAdmissionRate float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	log.Printf("📊 Updating admission rate to: %f", newAdmissionRate)

	// Update the internal admission rate
	rc.admissionRate = newAdmissionRate

	// Update the rate limiter to use the new admission rate
	rc.Limiter.SetLimit(rate.Limit(newAdmissionRate))
	log.Printf("🔧 Rate limiter updated with new rate: %f", newAdmissionRate)
}
