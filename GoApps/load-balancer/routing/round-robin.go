package routing

import (
	"context"
	"fmt"
	rdb "load-balancer/redis"
	"log"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type RoundRobinRoutingAlgorithm struct {
	counter int
	mu      sync.Mutex
}

func (r *RoundRobinRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	serviceNames := make([]string, 0, len(servicesMap))
	for name := range servicesMap {
		serviceNames = append(serviceNames, name)
	}

	destinationName := serviceNames[r.counter%len(serviceNames)]
	r.counter++

	destination := servicesMap[destinationName]

	destinationURL := fmt.Sprintf("http://%s.rabbitmq-setup.svc.cluster.local", destination.Name)
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ctx = cloudevents.ContextWithTarget(ctx, destinationURL)
	log.Printf("☁️ Sending CloudEvent to %s", destination.Name)

	if result := c.Send(ctx, event); !cloudevents.IsACK(result) {
		log.Printf("❌ Failed to send: %v", result)
		return
	}

	log.Printf("✅ Successfully sent event to %s", destination.Name)
}
