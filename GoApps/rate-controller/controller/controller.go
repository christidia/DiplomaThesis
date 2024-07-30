package controller

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateController struct {
	mu                 sync.Mutex
	admissionRate      float64
	lastEmptyQueueTime time.Time
	alpha              float64
	beta               float64
	limiter            *rate.Limiter
}

func NewRateController(alpha, beta float64) *RateController {
	initialRate := 1.0 // Initialize with a default admission rate
	return &RateController{
		admissionRate:      initialRate,
		lastEmptyQueueTime: time.Now(),
		alpha:              alpha,
		beta:               beta,
		limiter:            rate.NewLimiter(rate.Limit(initialRate), 1), // Create a rate limiter
	}
}

func (rc *RateController) GetAdmissionRate() float64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.admissionRate
}

func (rc *RateController) UpdateAdmissionRate(queueEmpty bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	currentTime := time.Now()
	elapsedTime := currentTime.Sub(rc.lastEmptyQueueTime).Seconds()

	if queueEmpty {
		rc.admissionRate = rc.beta * rc.admissionRate
		rc.lastEmptyQueueTime = currentTime
	} else {
		rc.admissionRate = rc.beta*rc.admissionRate + rc.alpha*elapsedTime
	}

	// Update the rate limiter with the new admission rate
	rc.limiter.SetLimit(rate.Limit(rc.admissionRate))
}
