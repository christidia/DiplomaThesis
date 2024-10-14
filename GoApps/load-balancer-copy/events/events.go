package events

import (
	"context"
	"log"
	"math/rand"
	"time"

	"load-balancer/db"
	"load-balancer/routing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
	localRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func StartReceiver() {
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
	routing.SelectedAlgorithm.RouteEvent(event, db.ServicesMap)
}
