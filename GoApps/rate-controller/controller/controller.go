package controller

import (
	"log"
	"sync"

	"golang.org/x/time/rate"
)

type RateController struct {
	mu            sync.Mutex
	admissionRate float64
	Limiter       *rate.Limiter
}

// NewRateController initializes a new RateController with the given alpha and beta values
func NewRateController() *RateController {
	initialRate := 10.0 // Initialize with a default admission rate
	return &RateController{
		admissionRate: initialRate,
		Limiter:       rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
	}
}

// UpdateAdmissionRateFromRedis updates the admission rate and rate limiter based on the value received from Redis
func (rc *RateController) UpdateAdmissionRateFromRedis(newAdmissionRate float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Update the internal admission rate
	rc.admissionRate = newAdmissionRate

	// Update the rate limiter to use the new admission rate
	rc.Limiter.SetLimit(rate.Limit(newAdmissionRate))

	log.Printf("🔄 Updated rate limiter to admission rate: %f", newAdmissionRate)
}
