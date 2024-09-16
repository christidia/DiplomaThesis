package controller

import (
	"sync"

	"golang.org/x/time/rate"
)

type RateController struct {
	mu            sync.Mutex
	admissionRate float64
	alpha         float64
	beta          float64
	Limiter       *rate.Limiter
}

func NewRateController(alpha, beta float64) *RateController {
	initialRate := 1.0 // Initialize with a default admission rate
	return &RateController{
		admissionRate: initialRate,
		alpha:         alpha,
		beta:          beta,
		Limiter:       rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
	}
}

func (rc *RateController) GetAdmissionRate() float64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.admissionRate
}

func (rc *RateController) UpdateAdmissionRateFromRedis(admissionRate float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Update admission rate
	rc.admissionRate = admissionRate
	// Update the rate limiter with the new admission rate
	rc.Limiter.SetLimit(rate.Limit(admissionRate))
}
