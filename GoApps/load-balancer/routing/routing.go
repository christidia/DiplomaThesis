package routing

import (
	"context"
	"fmt"
	"load/config"
	"log"
	"math/rand"
	"sync"
	"time"

	"load-balancer/redis"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
	SelectedAlgorithm RoutingAlgorithm
)

type RoutingAlgorithm interface {
	RouteEvent(event cloudevents.Event, servicesMap map[string]*redis.Service)
}

type AIMDRoutingAlgorithm struct{}

func (a *AIMDRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*redis.Service) {
	totalRate := 0
	for _, service := range servicesMap {
		totalRate += service.CurrWeight
	}

	randomValue := rand.Intn(totalRate)

	cumulativeRate := 0
	var destination *redis.Service
	for _, service := range servicesMap {
		cumulativeRate += service.CurrWeight
		if randomValue < cumulativeRate {
			destination = service
			break
		}
	}

	if destination == nil {
		log.Println("❌ Error: Destination is empty. Admission rate selection failed.")
		return
	}

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

type RoundRobinRoutingAlgorithm struct {
	counter int
	mu      sync.Mutex
}

func (r *RoundRobinRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*redis.Service) {
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

func StartAdmissionRateUpdater(rdb *redis.Client) {
	ticker := time.NewTicker(config.AdmissionRateInterval)
	defer ticker.Stop()

	for {
		select {
		case currentTime := <-ticker.C:
			redis.UpdateAdmissionRates(rdb, currentTime)
		}
	}
}