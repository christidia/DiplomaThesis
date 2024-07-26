package routing

import (
	"log"
	"math/rand"
	"time"

	"load-balancer/config"
	rdb "load-balancer/redis"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	redis "github.com/go-redis/redis/v8"
)

var (
	SelectedAlgorithm RoutingAlgorithm
	localRand         = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type RoutingAlgorithm interface {
	RouteEvent(event cloudevents.Event, servicesMap map[string]*rdb.Service)
}

func StartAdmissionRateUpdater(rdbClient *redis.Client) {
	ticker := time.NewTicker(config.AdmissionRateInterval)
	defer ticker.Stop()

	for currentTime := range ticker.C {
		rdb.UpdateAdmissionRates(rdbClient, currentTime)
	}
}

func init() {
	config.LoadConfig()

	switch config.RoutingAlgorithm {
	case "AIMD":
		SelectedAlgorithm = &AIMDRoutingAlgorithm{}
	case "RoundRobin":
		SelectedAlgorithm = &RoundRobinRoutingAlgorithm{}
	default:
		log.Fatalf("❌ Invalid or unsupported ROUTING_ALGORITHM value: %s", config.RoutingAlgorithm)
	}
}
