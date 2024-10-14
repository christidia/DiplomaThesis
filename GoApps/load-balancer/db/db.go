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
	Name             string
	RawAdmissionRate float64 // Raw value used for AIMD and admission controllers
	CurrWeight       float64 // Normalized value used for routing
	EmptyQWeight     float64 // Baseline value for raw admission rate when queue is empty
	Beta             float64
	Alpha            int
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
		"raw_admission_rate": service.RawAdmissionRate,
		"curr_weight":        service.CurrWeight,
		"emptyq_weight":      service.EmptyQWeight,
		"beta":               service.Beta,
		"alpha":              service.Alpha,
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
			Name:             name,
			CurrWeight:       config.InitialCurrWeights[i],   // Use the loaded value from config
			EmptyQWeight:     config.InitialEmptyQWeights[i], // Use the loaded value from config
			RawAdmissionRate: config.RawAdmissionRates[i],    // Use the loaded value from config
			Beta:             config.Betas[i],                // Use the loaded value from config
			Alpha:            config.Alphas[i],               // Use the loaded value from config
		}

		ServicesMap[service.Name] = service

		// Save the initialized values to Redis
		if err := SaveServiceToRedis(rdb, service); err != nil {
			log.Fatalf("❌ Error saving service %s to Redis: %v", service.Name, err)
		}
	}
	LastUpdateTime = time.Now()
	log.Println("✅ Services initialized")
}
