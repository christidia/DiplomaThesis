package routing

import (
	"context"
	"fmt"
	rdb "load-balancer/db"
	"log"
	"math/rand"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type RandomRoutingAlgorithm struct {
	mu sync.Mutex
}

func (r *RandomRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Seed the random number generator (can be done once in init if preferred)
	rand.Seed(time.Now().UnixNano())

	// Collect service names
	serviceNames := make([]string, 0, len(servicesMap))
	for name := range servicesMap {
		serviceNames = append(serviceNames, name)
	}

	// Select a random destination
	randomIndex := rand.Intn(len(serviceNames))
	destinationName := serviceNames[randomIndex]

	// Get the selected destination
	destination := servicesMap[destinationName]

	// Construct the destination URL
	destinationURL := fmt.Sprintf("http://%s.rabbitmq-setup.svc.cluster.local", destination.Name)
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}

	// Set up the context for the CloudEvent
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = cloudevents.ContextWithTarget(ctx, destinationURL)
	log.Printf("☁️ Sending CloudEvent to %s", destination.Name)

	// Send the CloudEvent
	if result := c.Send(ctx, event); !cloudevents.IsACK(result) {
		log.Printf("❌ Failed to send: %v", result)
		return
	}

	log.Printf("✅ Successfully sent event to %s", destination.Name)
}
