package routing

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	rdb "load-balancer/db"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type AIMDRoutingAlgorithm struct{}

// Helper function to generate prefix sums
func generatePrefixSums(servicesMap map[string]*rdb.Service) []int {
	prefixSums := make([]int, len(servicesMap))
	sum := 0
	index := 0
	for _, service := range servicesMap {
		sum += service.CurrWeight
		prefixSums[index] = sum
		index++
	}
	return prefixSums
}

// Binary search to find the appropriate service based on the random value
func binarySearch(prefixSums []int, randomValue int) int {
	return sort.Search(len(prefixSums), func(i int) bool {
		return prefixSums[i] > randomValue
	})
}

func (a *AIMDRoutingAlgorithm) RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service) {

	// Generate prefix sums for the services' weights
	prefixSums := generatePrefixSums(servicesMap)
	totalWeight := prefixSums[len(prefixSums)-1]

	// Log the prefix sums and total weight
	log.Printf("🧮 Prefix sums: %v, Total weight: %d", prefixSums, totalWeight)

	// Generate a random value between 0 and the total sum of weights
	randomValue := rand.Intn(totalWeight)
	log.Printf("🎲 Generated random value: %d", randomValue)

	// Use binary search to find the selected service index
	selectedIndex := binarySearch(prefixSums, randomValue)
	log.Printf("🎯 Selected service index: %d", selectedIndex)

	// Retrieve the selected service based on the index
	var destination *rdb.Service
	i := 0
	for _, service := range servicesMap {
		if i == selectedIndex {
			destination = service
			break
		}
		i++
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
