package routing

import (
	"context"
	"fmt"
	rdb "load-balancer/db"
	"log"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type WeightedRoundRobinRoutingAlgorithm struct {
	currentWeight int
	index         int
	mu            sync.Mutex
}

func (w *WeightedRoundRobinRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(servicesMap) == 0 {
		log.Println("❌ No available services to route the event.")
		return
	}

	// Step 1: Convert services into a weighted list
	serviceList := make([]*rdb.Service, 0, len(servicesMap))
	weights := make([]int, 0, len(servicesMap))

	totalWeight := 0
	for _, service := range servicesMap {
		serviceList = append(serviceList, service)
		weights = append(weights, service.Weight) // Assuming Service struct has `Weight` field
		totalWeight += service.Weight
	}

	// Step 2: Select service based on Weighted Round Robin logic
	for {
		w.index = (w.index + 1) % len(serviceList)
		if w.index == 0 {
			w.currentWeight--
			if w.currentWeight <= 0 {
				w.currentWeight = max(weights)
			}
		}

		if weights[w.index] >= w.currentWeight {
			break
		}
	}

	selectedService := serviceList[w.index]

	// Step 3: Send the event to the selected service
	destinationURL := fmt.Sprintf("http://%s.rabbitmq-setup.svc.cluster.local", selectedService.Name)
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = cloudevents.ContextWithTarget(ctx, destinationURL)
	log.Printf("☁️ Sending CloudEvent to %s (Weight: %d)", selectedService.Name, selectedService.Weight)

	if result := c.Send(ctx, event); !cloudevents.IsACK(result) {
		log.Printf("❌ Failed to send: %v", result)
		return
	}

	log.Printf("✅ Successfully sent event to %s", selectedService.Name)
}

// Helper function to get the maximum weight from the list
func max(weights []int) int {
	maxWeight := weights[0]
	for _, w := range weights {
		if w > maxWeight {
			maxWeight = w
		}
	}
	return maxWeight
}
