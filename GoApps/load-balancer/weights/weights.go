package weights

import (
	"log"
	"time"

	"load-balancer/config"
	"load-balancer/db"
	"load-balancer/metrics"

	"github.com/go-redis/redis/v8"
)

var (
	maxAdmissionRate = config.MaxAdmissionRate
	minAdmissionRate = config.MinAdmissionRate
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

	// Store the raw admission rates before normalizing for routing
	admissionRates := make(map[string]int)

	for _, service := range db.ServicesMap {
		// Apply AIMD on the raw admission rate with `EmptyQWeight` as the baseline
		rawRate := int(service.Beta*float64(service.EmptyQWeight)) + service.Alpha*int(elapsedTime)*metrics.FetchReplicaNum(service.Name)
		// Ensure the admission rate is within the logical bounds
		if rawRate > maxAdmissionRate {
			rawRate = maxAdmissionRate
		}
		// else if rawRate < minAdmissionRate {
		// 	rawRate = minAdmissionRate
		// }

		service.RawAdmissionRate = rawRate
		admissionRates[service.Name] = service.RawAdmissionRate

		// Save the raw admission rate in Redis for the respective service
		err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "raw_admission_rate", service.RawAdmissionRate).Err()
		if err != nil {
			log.Printf("❌ Error updating raw_admission_rate for service %s in Redis: %v", service.Name, err)
		} else {
			log.Printf("📥 Updated raw admission rate for %s: %d", service.Name, service.RawAdmissionRate)
		}
	}

	// Publish raw admission rates for admission controllers
	publishAdmissionRates(rdb)

	// Normalize the raw admission rates for routing
	normalizeWeights(rdb)
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
			service.EmptyQWeight = service.RawAdmissionRate
			err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "emptyq_weight", service.EmptyQWeight).Err()
			if err != nil {
				log.Printf("❌ Error updating emptyq_weight for service %s in Redis: %v", service.Name, err)
			}

			// Update the Prometheus metric
			db.EmptyQWeights[service.Name] = float64(service.EmptyQWeight)
			metrics.UpdateGamma()

			// Queue has emptied for the first time
			db.PrevQueueEmpty = true
		}

		log.Printf("⚖️ Updated EmptyQWeight for all services\n")
	} else {
		log.Printf("ℹ️ Queue already empty, no new empty queue event triggered.")
	}
}

func UpdateEmptyQWeightRoutine() {
	rdb := db.NewRedisClient()
	currentTime := time.Now()
	createEmptyQueueEvent(rdb, currentTime)
}

// Publish individual admission rates to Redis after normalizing them.
func publishAdmissionRates(rdb *redis.Client) {
	for _, service := range db.ServicesMap {
		admissionRate := float64(service.RawAdmissionRate)
		channel := "admission_rate:" + service.Name
		err := rdb.Publish(db.Ctx, channel, admissionRate).Err()
		if err != nil {
			log.Printf("❌ Error publishing admission rate for service %s: %v", service.Name, err)
		} else {
			log.Printf("📡 Published admission rate for %s: %f", service.Name, admissionRate)
		}
	}
}

// Normalize the raw admission rates so that they sum up to 100
func normalizeWeights(rdb *redis.Client) {
	totalWeight := 0
	for _, service := range db.ServicesMap {
		totalWeight += service.RawAdmissionRate
	}

	if totalWeight == 0 {
		log.Println("❌ Error: Total weight is zero. Cannot normalize.")
		return
	}

	normalizationFactor := 100.0 / float64(totalWeight)
	roundedWeights := make(map[string]int)
	totalRoundedWeight := 0

	// First pass: normalize and round weights
	for _, service := range db.ServicesMap {
		normalizedWeight := float64(service.RawAdmissionRate) * normalizationFactor
		roundedWeight := int(normalizedWeight)
		roundedWeights[service.Name] = roundedWeight
		totalRoundedWeight += roundedWeight
	}

	// Calculate the rounding error
	roundingError := 100 - totalRoundedWeight

	// Second pass: distribute the rounding error
	for _, service := range db.ServicesMap {
		if roundingError == 0 {
			break
		}
		if roundedWeights[service.Name] > 0 {
			roundedWeights[service.Name]++
			roundingError--
		}
	}

	// Update the service weights with the normalized values
	for _, service := range db.ServicesMap {
		service.CurrWeight = roundedWeights[service.Name]

		// Save the normalized CurrWeight in Redis for the respective service
		err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
		if err != nil {
			log.Printf("❌ Error updating curr_weight for service %s in Redis: %v", service.Name, err)
		} else {
			log.Printf("📥 Updated curr_weight for %s: %d", service.Name, service.CurrWeight)
		}
	}

	// Log the final weights to verify correctness
	totalWeight = 0
	for _, service := range db.ServicesMap {
		totalWeight += service.CurrWeight
		log.Printf("📈 Normalized weight for %s: %d", service.Name, service.CurrWeight)
	}
	log.Printf("📊 Total normalized weight: %d", totalWeight)
}
