package routing

import (
	"context"
	"fmt"
	"log"
	"time"

	rdb "load-balancer/db"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type AIMDRoutingAlgorithm struct{}

func (a *AIMDRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {
	// Generate a random value between 0 and 100
	randomValue := localRand.Intn(100)

	cumulativeRate := 0
	var destination *rdb.Service
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
