package events

import (
	"context"
	"load-balancer/routing"
	"log"
	"math/rand"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func StartReceiver() {
	rand.Seed(time.Now().UnixNano())
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		Receive(event)
	})
	if err != nil {
		log.Fatalf("❌ Failed to start receiver: %v", err)
	}
}

func Receive(event cloudevents.Event) {
	routing.SelectedAlgorithm.RouteEvent(event, redis.ServicesMap)
}
