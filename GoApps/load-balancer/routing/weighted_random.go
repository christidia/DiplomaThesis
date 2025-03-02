package routing

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	rdb "load-balancer/db"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type WeightedRandomRoutingAlgorithm struct {
	mu sync.Mutex
}

func (w *WeightedRandomRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {
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

	// Step 2: Select a service based on Weighted Random Distribution
	rand.Seed(time.Now().UnixNano())
	randomWeight := rand.Intn(totalWeight)

	accumulatedWeight := 0
	var selectedService *rdb.Service

	for i, weight := range weights {
		accumulatedWeight += weight
		if randomWeight < accumulatedWeight {
			selectedService = serviceList[i]
			break
		}
	}

	if selectedService == nil {
		log.Println("❌ Failed to select a service")
		return
	}

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
