package controller

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"sync"

	"rate-controller/config"

	"golang.org/x/time/rate"
)

type RateController struct {
	mu            sync.Mutex
	admissionRate float64
	Limiter       *rate.Limiter
	Client        *http.Client // HTTP client to send requests
}

// NewRateController initializes a new RateController with the given alpha and beta values
func NewRateController(alpha, beta float64) *RateController {
	initialRate := 1.0 // Initialize with a default admission rate
	log.Printf("🔧 Creating RateLimiter with initial rate: %f", initialRate)
	return &RateController{
		admissionRate: initialRate,
		Limiter:       rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
		Client:        &http.Client{},                              // Initialize the HTTP client
	}
}

// ForwardRequest forwards the buffered request to the service URL based on the current admission rate
func (rc *RateController) ForwardRequest(requestBody []byte) {
	// Wait until allowed by the rate limiter
	err := rc.Limiter.Wait(context.Background())
	if err != nil {
		log.Printf("⚠️ Error waiting on rate limiter: %v", err)
		return
	}

	log.Printf("⏩ Forwarding request to %s", config.ServiceURL)

	// Retry mechanism
	const maxRetries = 3
	for retries := 0; retries < maxRetries; retries++ {
		req, err := http.NewRequest("POST", config.ServiceURL, bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("❌ Error creating HTTP request: %v", err)
			return
		}

		resp, err := rc.Client.Do(req)
		if err != nil {
			log.Printf("❌ Error forwarding request to %s: %v (attempt %d/%d)", config.ServiceURL, err, retries+1, maxRetries)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("✅ Successfully forwarded request to %s, response code: %d", config.ServiceURL, resp.StatusCode)
			return
		}
		log.Printf("⚠️ Request to %s failed with response code: %d (attempt %d/%d)", config.ServiceURL, resp.StatusCode, retries+1, maxRetries)
	}
	log.Printf("❌ Failed to forward request to %s after %d retries", config.ServiceURL, maxRetries)
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
