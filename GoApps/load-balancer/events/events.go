package events

import (
	"context"
	"log"
	"time"

	rdb "load-balancer/redis"
	"load-balancer/routing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"golang.org/x/exp/rand"
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
	routing.SelectedAlgorithm.RouteEvent(event, rdb.ServicesMap)
}
