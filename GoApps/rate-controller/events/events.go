package events

import (
	"context"
	"log"

	"rate-controller/controller"
	"rate-controller/metrics"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
	rateController *controller.RateController
)

type EmptyQueueEvent struct {
	ServiceName string `json:"serviceName"`
}

type RequestEvent struct {
	RequestData string `json:"requestData"`
}

func InitRateController(alpha, beta float64) {
	rateController = controller.NewRateController(alpha, beta)
}

func HandleEmptyQueueEvent(event cloudevents.Event) {
	var eqEvent EmptyQueueEvent
	if err := event.DataAs(&eqEvent); err != nil {
		log.Printf("⚠️ Error parsing EmptyQueueEvent data: %v", err)
		return
	}

	rateController.UpdateAdmissionRate(true)
	log.Printf("Handled empty queue event for service: %s", eqEvent.ServiceName)
}

func HandleRequestEvent(event cloudevents.Event) {
	var reqEvent RequestEvent
	if err := event.DataAs(&reqEvent); err != nil {
		log.Printf("⚠️ Error parsing RequestEvent data: %v", err)
		return
	}

	// Wait for the rate limiter to allow the event to be processed
	err := rateController.limiter.Wait(context.Background())
	if err != nil {
		log.Printf("⚠️ Error waiting for rate limiter: %v", err)
		return
	}

	admissionRate := rateController.GetAdmissionRate()
	metrics.UpdateMetric(admissionRate)
	log.Printf("Handled request event with data: %s", reqEvent.RequestData)

	// Process the request based on the current admission rate
	// Forward the request to another service or process it locally
	log.Printf("Processed request event with data: %s", reqEvent.RequestData)
}

func StartReceiver() {
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("❌ Failed to create CloudEvents client: %v", err)
	}

	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
		switch event.Type() {
		case "com.example.emptyqueue":
			HandleEmptyQueueEvent(event)
		case "com.example.request":
			HandleRequestEvent(event)
		default:
			log.Printf("Received unsupported event type: %s", event.Type())
		}
	})
	if err != nil {
		log.Fatalf("❌ Failed to start receiver: %v", err)
	}
}
