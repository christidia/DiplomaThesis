package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"load-balancer/config"
	"load-balancer/metrics"

	"github.com/go-redis/redis/v8"
)

var (
	ctx                 = context.Background()
	ServicesMap         map[string]*Service
	ServiceKeyPrefix    = "service:"
	TkKey               = "tk"
	LastUpdateTime      time.Time
	PrevQueueEmpty      bool
	AdmissionRatesMutex sync.Mutex
)

type Service struct {
	Name         string
	CurrWeight   int
	EmptyQWeight int
	Beta         float64
	Alpha        int
}

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPass,
	})
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
		if err := saveServiceToRedis(rdb, service); err != nil {
			log.Fatalf("❌ Error saving service %s to Redis: %v", service.Name, err)
		}
	}
	LastUpdateTime = time.Now()
	log.Println("✅ Services initialized")
}

func saveServiceToRedis(rdb *redis.Client, service *Service) error {
	key := ServiceKeyPrefix + service.Name
	err := rdb.HSet(ctx, key, map[string]interface{}{
		"curr_weight":   service.CurrWeight,
		"emptyq_weight": service.EmptyQWeight,
		"beta":          service.Beta,
		"alpha":         service.Alpha,
	}).Err()
	return err
}

func keyExists(rdb *redis.Client, key string) (bool, error) {
	val, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

func InitializeTkIfNotExists(rdb *redis.Client) error {
	exists, err := keyExists(rdb, TkKey)
	if err != nil {
		return err
	}
	if !exists {
		tk := time.Now().Add(-time.Minute).Unix() // Initialize 'tk' to the current timestamp minus an offset
		err := rdb.Set(ctx, TkKey, tk, 0).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateAdmissionRates(rdb *redis.Client, currentTime time.Time) {
	AdmissionRatesMutex.Lock()
	defer AdmissionRatesMutex.Unlock()

	tk, err := rdb.Get(ctx, TkKey).Int64()
	if err != nil {
		log.Printf("❌ Error retrieving 'tk' value from Redis: %v", err)
		return
	}

	elapsedTime := currentTime.Sub(time.Unix(tk, 0)).Seconds()

	for _, service := range ServicesMap {
		service.CurrWeight = int(service.Beta*float64(service.EmptyQWeight)) + service.Alpha*int(elapsedTime)
		err := rdb.HSet(ctx, ServiceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
		if err != nil {
			log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
		}
		log.Printf("📈 Updated admission rate for %s: %d", service.Name, service.CurrWeight)
	}
}

func updateTkInRedis(rdb *redis.Client, currentTime time.Time) {
	tk := currentTime.Unix()
	err := rdb.Set(ctx, TkKey, tk, 0).Err()
	if err != nil {
		log.Printf("❌ Error updating 'tk' value in Redis: %v", err)
		return
	}
	log.Printf("🗑️ Empty queue event at %s. Updated 'tk' to %d.", currentTime.Format(time.RFC3339), tk)
}

func createEmptyQueueEvent(rdb *redis.Client, currentTime time.Time) {
	if !PrevQueueEmpty {
		log.Println("🛠️ Starting updateEmptyQWeightRoutine")
		updateTkInRedis(rdb, currentTime)

		AdmissionRatesMutex.Lock()
		defer AdmissionRatesMutex.Unlock()

		for _, service := range ServicesMap {
			service.EmptyQWeight = service.CurrWeight
			err := rdb.HSet(ctx, ServiceKeyPrefix+service.Name, "emptyq_weight", service.EmptyQWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating emptyq_weight for service %s in Redis: %v", service.Name, err)
			}

			// Update the Prometheus metric
			metrics.UpdateMetric(service.Name, float64(service.EmptyQWeight))
		}

		log.Printf("⚖️ Updated EmptyQWeight for all services: %v\n", ServicesMap)
	} else {
		log.Printf("ℹ️ Queue already empty, no new empty queue event triggered.")
	}
}

func UpdateEmptyQWeightRoutine() {
	rdb := NewRedisClient()
	currentTime := time.Now()
	createEmptyQueueEvent(rdb, currentTime)
}
