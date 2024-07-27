package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"load-balancer/config"

	"github.com/go-redis/redis/v8"
)

var (
	Ctx = context.Background()
)

type Service struct {
	Name         string
	CurrWeight   int
	EmptyQWeight int
	Beta         float64
	Alpha        int
}

var (
	ServicesMap         map[string]*Service
	ServiceKeyPrefix    = "service:"
	TkKey               = "tk"
	LastUpdateTime      time.Time
	PrevQueueEmpty      bool
	AdmissionRatesMutex sync.Mutex
	EmptyQWeights       = make(map[string]float64)
)

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass,
	})
}

func SaveServiceToRedis(rdb *redis.Client, service *Service) error {
	key := ServiceKeyPrefix + service.Name
	err := rdb.HSet(Ctx, key, map[string]interface{}{
		"curr_weight":   service.CurrWeight,
		"emptyq_weight": service.EmptyQWeight,
		"beta":          service.Beta,
		"alpha":         service.Alpha,
	}).Err()
	return err
}

func KeyExists(rdb *redis.Client, key string) (bool, error) {
	val, err := rdb.Exists(Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

func InitializeServices(rdb *redis.Client) {
	ServicesMap = make(map[string]*Service)
	for i := 0; i < config.NumServices; i++ {
		name := fmt.Sprintf("service%d", i+1)
		service := &Service{
			Name:         name,
			CurrWeight:   10 * (i + 1),
			EmptyQWeight: 10 * (i + 1),
			Beta:         0.5,
			Alpha:        3 + i,
		}
		ServicesMap[service.Name] = service
		if err := SaveServiceToRedis(rdb, service); err != nil {
			log.Fatalf("❌ Error saving service %s to Redis: %v", service.Name, err)
		}
	}
	LastUpdateTime = time.Now()
	log.Println("✅ Services initialized")
}
