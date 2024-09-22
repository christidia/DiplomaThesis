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
	RequestQueue  chan []byte        // Channel to buffer incoming requests
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

	// Initialize the request queue (buffer) with a size of 100
	requestQueue := make(chan []byte, 100)

	// Start the process to forward requests from the queue with rate limiting
	rc := &RateController{
		admissionRate: initialRate,
		Limiter:       rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
		Client:        client,
		RequestQueue:  requestQueue,
	}

	// Start the goroutine to forward requests from the buffer
	go rc.startForwarding()

	return rc
}

// ForwardRequestToBuffer adds incoming requests to the buffer without rate limiting
func (rc *RateController) ForwardRequestToBuffer(requestBody []byte) {
	log.Println("📥 Request added to buffer")
	rc.RequestQueue <- requestBody // Add the request to the buffer
}

// startForwarding is a goroutine that reads from the request buffer and forwards requests to the service at a rate-limited pace
func (rc *RateController) startForwarding() {
	for requestBody := range rc.RequestQueue {
		// Wait until allowed by the rate limiter
		err := rc.Limiter.Wait(context.Background())
		if err != nil {
			log.Printf("⚠️ Error waiting on rate limiter: %v", err)
			continue
		}
		// Forward the request to the consuming service
		rc.forwardRequest(requestBody)
	}
}

// forwardRequest forwards a single request to the consuming service as a CloudEvent
func (rc *RateController) forwardRequest(requestBody []byte) {
	log.Printf("⏩ Forwarding request to %s as CloudEvent", config.ServiceURL)

	// Encode the request data (assuming it's image data or some payload) in base64
	encodedData := base64.StdEncoding.EncodeToString(requestBody)

	// Create a CloudEvent with the expected data structure
	event := cloudevents.NewEvent()
	event.SetSource("admission-controller")     // Adjust source to match your system
	event.SetType("com.example.admission.rate") // Set appropriate event type for your case

	// Set the event data. Assuming the consuming service expects "imageData" or other structured data
	event.SetData(cloudevents.ApplicationJSON, map[string]string{
		"imageData": encodedData, // or other data as required
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
