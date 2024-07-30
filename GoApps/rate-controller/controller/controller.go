package controller

import (
	"sync"
	"time"
)

type RateController struct {
	mu                 sync.Mutex
	admissionRate      float64
	lastEmptyQueueTime time.Time
	alpha              float64
	beta               float64
}

func NewRateController(alpha, beta float64) *RateController {
	return &RateController{
		admissionRate:      1.0, // Initialize with a default admission rate
		lastEmptyQueueTime: time.Now(),
		alpha:              alpha,
		beta:               beta,
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
}
