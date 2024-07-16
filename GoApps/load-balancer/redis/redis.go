package redis

import (
    "context"
    "log"
    "time"

    "github.com/go-redis/redis/v8"
    "load-balancer/config"
)

var (
    ctx             = context.Background()
    servicesMap     map[string]*Service
    serviceKeyPrefix = "service:"
    tkKey            = "tk"
    lastUpdateTime   time.Time
)

type Service struct {
    Name         string
    CurrWeight   int
    EmptyQWeight int
    Beta         float64
    Alpha        int
)

func NewRedisClient() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     config.RedisURL,
        Password: config.RedisPass,
    })
}

func InitializeServices(rdb *redis.Client) {
    servicesMap = make(map[string]*Service)
    for i := 0; i < config.NumServices; i++ {
        name := fmt.Sprintf("service%d", i+1)
        service := &Service{
            Name:         name,
            CurrWeight:   10 * (i + 1),
            EmptyQWeight: 10 * (i + 1),
            Beta:         0.5,
            Alpha:        3 + i,
        }
        servicesMap[service.Name] = service
        if err := saveServiceToRedis(rdb, service); err != nil {
            log.Fatalf("❌ Error saving service %s to Redis: %v", service.Name, err)
        }
    }
    lastUpdateTime = time.Now()
    log.Println("✅ Services initialized")
}

func saveServiceToRedis(rdb *redis.Client, service *Service) error {
    key := serviceKeyPrefix + service.Name
    err := rdb.HSet(ctx, key, map[string]interface{}{
        "curr_weight":   service.CurrWeight,
        "emptyq_weight": service.EmptyQWeight,
        "beta":          service.Beta,
        "alpha":         service.Alpha,
    }).Err()
    return err
}

// Other Redis-related functions...
