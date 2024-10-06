package weights

import (
	"fmt"
	"log"
	"time"

	"load-balancer/config"
	"load-balancer/db"
	"load-balancer/metrics"

	"github.com/go-redis/redis/v8"
)

var (
	maxAdmissionRate int
	minAdmissionRate int
)

func InitializeWeights() {
	// Call LoadConfig() to load env variables
	config.LoadConfig()

	// Set minAdmissionRate from the loaded config
	minAdmissionRate = config.MinAdmissionRate
	maxAdmissionRate = config.MaxAdmissionRate

	log.Printf("📋 Admission Rate Config: min=%d, max=%d", minAdmissionRate, maxAdmissionRate)
}

func InitializeTkIfNotExists(rdb *redis.Client) error {
	// Always initialize tk to the current time minus 0.1 seconds
	tk := time.Now().Add(-100 * time.Millisecond).Unix() // Initialize 'tk' to the current timestamp minus 0.1 seconds
	err := rdb.Set(db.Ctx, db.TkKey, tk, 0).Err()
	if err != nil {
		log.Printf("ERROR INITIALIZING TK IN REDIS: %v", err)
		return err
	}
	log.Printf("TK INITIALIZED OR UPDATED WITH TIMESTAMP: %d", tk)
	return nil
}

func UpdateAdmissionRates(rdb *redis.Client, currentTime time.Time) {
	db.AdmissionRatesMutex.Lock()
	defer db.AdmissionRatesMutex.Unlock()

	log.Println("STARTING ADMISSION RATE UPDATE")

	tk, err := rdb.Get(db.Ctx, db.TkKey).Int64()
	if err != nil {
		log.Printf("ERROR RETRIEVING TK VALUE FROM REDIS: %v", err)
		return
	}
	log.Printf("RETRIEVED TK VALUE: %d", tk)

	elapsedTime := currentTime.Sub(time.Unix(tk, 0)).Seconds()
	log.Printf("ELAPSED TIME SINCE TK: %f SECONDS", elapsedTime)

	// Store the raw admission rates before normalizing for routing
	admissionRates := make(map[string]int)

	for _, service := range db.ServicesMap {
		log.Printf("UPDATING RAW ADMISSION RATE FOR SERVICE: %s", service.Name)
		replicas := metrics.FetchReplicaNum(service.Name)
		replicas = max(replicas, 1)
		// Apply AIMD on the raw admission rate with `EmptyQWeight` as the baseline
		rawRate := int(service.Beta*float64(service.EmptyQWeight)) + service.Alpha*int(elapsedTime)*replicas

		log.Printf("CALCULATED RAW ADMISSION RATE FOR %s: %d", service.Name, rawRate)

		// Ensure the admission rate is within the logical bounds
		// if rawRate > maxAdmissionRate {
		// 	rawRate = maxAdmissionRate
		// 	log.Printf("RAW ADMISSION RATE FOR %s EXCEEDED MAX LIMIT, SET TO: %d", service.Name, maxAdmissionRate)
		// } else if rawRate < minAdmissionRate {
		if rawRate < minAdmissionRate {
			rawRate = minAdmissionRate
			log.Printf("RAW ADMISSION RATE FOR %s FELL BELOW MIN LIMIT, SET TO: %d", service.Name, minAdmissionRate)
		}

		service.RawAdmissionRate = rawRate
		admissionRates[service.Name] = service.RawAdmissionRate

		// Save the raw admission rate in Redis for the respective service
		err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "raw_admission_rate", service.RawAdmissionRate).Err()
		if err != nil {
			log.Printf("ERROR UPDATING RAW ADMISSION RATE FOR SERVICE %s IN REDIS: %v", service.Name, err)
		} else {
			log.Printf("UPDATED RAW ADMISSION RATE FOR %s: %d", service.Name, service.RawAdmissionRate)
		}
	}

	// Publish raw admission rates for admission controllers
	publishAdmissionRates(rdb)

	// Normalize the raw admission rates for routing
	normalizeWeights(rdb)

	log.Println("COMPLETED ADMISSION RATE UPDATE")
}

func updateTkInRedis(rdb *redis.Client, currentTime time.Time) {
	tk := currentTime.Unix()
	err := rdb.Set(db.Ctx, db.TkKey, tk, 0).Err()
	if err != nil {
		log.Printf("ERROR UPDATING TK VALUE IN REDIS: %v", err)
		return
	}
	log.Printf("UPDATED TK TO %d AT %s", tk, currentTime.Format(time.RFC3339))
}

func createEmptyQueueEvent(rdb *redis.Client, currentTime time.Time) {
	if !db.PrevQueueEmpty {
		log.Println("STARTING EMPTY QUEUE EVENT ROUTINE")
		updateTkInRedis(rdb, currentTime)

		db.AdmissionRatesMutex.Lock()
		defer db.AdmissionRatesMutex.Unlock()

		for _, service := range db.ServicesMap {
			log.Printf("UPDATING EMPTY QUEUE WEIGHT FOR SERVICE: %s", service.Name)
			service.EmptyQWeight = service.RawAdmissionRate
			err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "emptyq_weight", service.EmptyQWeight).Err()
			if err != nil {
				log.Printf("ERROR UPDATING EMPTYQ WEIGHT FOR SERVICE %s IN REDIS: %v", service.Name, err)
			}

			// Update the Prometheus metric
			db.EmptyQWeights[service.Name] = float64(service.EmptyQWeight)
			metrics.UpdateGamma()
		}

		db.PrevQueueEmpty = true
		log.Println("COMPLETED EMPTY QUEUE EVENT ROUTINE")
	} else {
		log.Println("QUEUE ALREADY EMPTY, NO NEW EVENT TRIGGERED")
	}
}

func UpdateEmptyQWeightRoutine() {
	rdb := db.NewRedisClient()
	currentTime := time.Now()
	createEmptyQueueEvent(rdb, currentTime)
}

func publishAdmissionRates(rdb *redis.Client) {
	log.Println("📢 PUBLISHING ADMISSION RATES TO REDIS")
	for _, service := range db.ServicesMap {
		admissionRate := float64(service.CurrWeight)

		// Convert admissionRate to a string before publishing
		admissionRateStr := fmt.Sprintf("%f", admissionRate)
		channel := "admission_rate:" + service.Name

		err := rdb.Publish(db.Ctx, channel, admissionRateStr).Err() // Publish the string
		if err != nil {
			log.Printf("❌ ERROR PUBLISHING ADMISSION RATE FOR SERVICE %s: %v", service.Name, err)
		} else {
			log.Printf("✅ PUBLISHED ADMISSION RATE FOR %s: %s", service.Name, admissionRateStr)
		}
	}
	log.Println("📤 ALL ADMISSION RATES PUBLISHED SUCCESSFULLY!")
}

func normalizeWeights(rdb *redis.Client) {
	log.Println("STARTING WEIGHT NORMALIZATION")

	totalWeight := 0
	for _, service := range db.ServicesMap {
		totalWeight += service.RawAdmissionRate
	}

	if totalWeight == 0 {
		log.Println("ERROR: TOTAL WEIGHT IS ZERO, CANNOT NORMALIZE")
		return
	}

	normalizationFactor := 100.0 / float64(totalWeight)
	roundedWeights := make(map[string]int)
	totalRoundedWeight := 0

	for _, service := range db.ServicesMap {
		normalizedWeight := float64(service.RawAdmissionRate) * normalizationFactor
		roundedWeight := int(normalizedWeight)
		roundedWeights[service.Name] = roundedWeight
		totalRoundedWeight += roundedWeight
		log.Printf("NORMALIZED WEIGHT FOR %s: %d", service.Name, roundedWeight)
	}

	roundingError := 100 - totalRoundedWeight

	for _, service := range db.ServicesMap {
		if roundingError == 0 {
			break
		}
		if roundedWeights[service.Name] > 0 {
			roundedWeights[service.Name]++
			roundingError--
		}
	}

	for _, service := range db.ServicesMap {
		service.CurrWeight = roundedWeights[service.Name]

		err := rdb.HSet(db.Ctx, db.ServiceKeyPrefix+service.Name, "curr_weight", service.CurrWeight).Err()
		if err != nil {
			log.Printf("ERROR UPDATING CURR WEIGHT FOR SERVICE %s IN REDIS: %v", service.Name, err)
		} else {
			log.Printf("UPDATED CURR WEIGHT FOR %s: %d", service.Name, service.CurrWeight)
		}
	}

	log.Println("COMPLETED WEIGHT NORMALIZATION")
}
