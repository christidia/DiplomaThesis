package weights

import (
	"log"
	"time"

	"load-balancer/db"
	"load-balancer/metrics"

	"github.com/go-redis/redis/v8"
)

func InitializeTkIfNotExists(rdb *redis.Client) error {
	exists, err := db.KeyExists(rdb, db.TkKey)
	if err != nil {
		return err
	}
	if !exists {
		tk := time.Now().Add(-time.Minute).Unix() // Initialize 'tk' to the current timestamp minus an offset
		err := rdb.Set(db.Ctx, db.TkKey, tk, 0).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateAdmissionRates(rdb *redis.Client, currentTime time.Time) {
	db.AdmissionRatesMutex.Lock()
	defer db.AdmissionRatesMutex.Unlock()

	tk, err := rdb.Get(db.Ctx, db.TkKey).Int64()
	if err != nil {
		log.Printf("❌ Error retrieving 'tk' value from Redis: %v", err)
		return
	}

	elapsedTime := currentTime.Sub(time.Unix(tk, 0)).Seconds()

	for _, service := range db.ServicesMap {
		service.CurrWeight = int(service.Beta*float64(service.EmptyQWeight)) + service.Alpha*int(elapsedTime)*metrics.FetchReplicaNum(service.Name)
		err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
		if err != nil {
			log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
		}
		log.Printf("📈 Updated admission rate for %s: %d", service.Name, service.CurrWeight)
	}
}

func updateTkInRedis(rdb *redis.Client, currentTime time.Time) {
	tk := currentTime.Unix()
	err := rdb.Set(db.Ctx, db.TkKey, tk, 0).Err()
	if err != nil {
		log.Printf("❌ Error updating 'tk' value in Redis: %v", err)
		return
	}
	log.Printf("🗑️ Empty queue event at %s. Updated 'tk' to %d.", currentTime.Format(time.RFC3339), tk)
}

func createEmptyQueueEvent(rdb *redis.Client, currentTime time.Time) {
	if !db.PrevQueueEmpty {
		log.Println("🛠️ Starting updateEmptyQWeightRoutine")
		updateTkInRedis(rdb, currentTime)

		db.AdmissionRatesMutex.Lock()
		defer db.AdmissionRatesMutex.Unlock()

		for _, service := range db.ServicesMap {
			service.EmptyQWeight = service.CurrWeight
			err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "emptyq_weight", service.EmptyQWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating emptyq_weight for service %s in Redis: %v", service.Name, err)
			}

			// Update the Prometheus metric
			metrics.UpdateMetric(service.Name, float64(service.EmptyQWeight))
			db.EmptyQWeights[service.Name] = float64(service.EmptyQWeight)
			metrics.CalculateGamma()
			db.PrevQueueEmpty = true
		}

		log.Printf("⚖️ Updated EmptyQWeight for all services: %v\n", db.ServicesMap)
	} else {
		log.Printf("ℹ️ Queue already empty, no new empty queue event triggered.")
	}
}

func UpdateEmptyQWeightRoutine() {
	rdb := db.NewRedisClient()
	currentTime := time.Now()
	createEmptyQueueEvent(rdb, currentTime)
}
