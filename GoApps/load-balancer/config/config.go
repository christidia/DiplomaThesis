package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	RedisURL              string
	RedisPass             string
	RabbitMQURLhttp       string
	RabbitMQURL           string
	RabbitMQUser          string
	RabbitMQPass          string
	CheckInterval         time.Duration
	AdmissionRateInterval time.Duration
	NumServices           int
	RoutingAlgorithm      string
	MaxAdmissionRate      int
	MinAdmissionRate      int

	// Maps for service-specific parameters
	InitialCurrWeights   = make(map[int]int)
	InitialEmptyQWeights = make(map[int]int)
	RawAdmissionRates    = make(map[int]int)
	Alphas               = make(map[int]int)
	Betas                = make(map[int]float64)
)

func LoadConfig() {
	RedisURL = os.Getenv("REDIS_URL")
	if RedisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is not set")
	}
	RedisPass = os.Getenv("REDIS_PASSWORD")
	if RedisPass == "" {
		log.Fatal("❌ REDIS_PASSWORD environment variable is not set")
	}
	RabbitMQURLhttp = os.Getenv("RABBITMQ_URL")
	if RabbitMQURLhttp == "" {
		log.Fatal("❌ RABBITMQ_URL environment variable is not set")
	}
	RabbitMQUser = os.Getenv("RABBITMQ_USERNAME")
	if RabbitMQUser == "" {
		log.Fatal("❌ RABBITMQ_USERNAME environment variable is not set")
	}
	RabbitMQPass = os.Getenv("RABBITMQ_PASSWORD")
	if RabbitMQPass == "" {
		log.Fatal("❌ RABBITMQ_PASSWORD environment variable is not set")
	}

	// Construct RabbitMQ URL with username and password
	RabbitMQURL = fmt.Sprintf("amqp://%s:%s@rabbitmq.rabbitmq-setup.svc.cluster.local:5672/",
		RabbitMQUser, RabbitMQPass)
	log.Printf("🔗 Constructed RabbitMQ URL: %s", RabbitMQURL)

	checkIntervalStr := os.Getenv("CHECK_INTERVAL")
	if checkIntervalStr == "" {
		CheckInterval = 500 * time.Millisecond
	} else {
		checkInterval, err := strconv.Atoi(checkIntervalStr)
		if err != nil || checkInterval <= 0 {
			log.Printf("⚠️ Invalid value for CHECK_INTERVAL: %s. Using default: 500ms", checkIntervalStr)
			CheckInterval = 500 * time.Millisecond
		} else {
			CheckInterval = time.Duration(checkInterval) * time.Millisecond
		}
	}
	AdmissionRateInterval = CheckInterval
	numServicesStr := os.Getenv("NUM_SERVICES")
	if numServicesStr == "" {
		log.Fatal("❌ NUM_SERVICES environment variable is not set")
	}
	numServices, err := strconv.Atoi(numServicesStr)
	if err != nil {
		log.Fatalf("❌ Invalid NUM_SERVICES value: %v", err)
	}
	NumServices = numServices

	RoutingAlgorithm = os.Getenv("ROUTING_ALGORITHM")
	if RoutingAlgorithm == "" {
		log.Println("⚠️ ROUTING_ALGORITHM environment variable is not set. Using default: AIMD")
		RoutingAlgorithm = "AIMD"
	}

	// Load the max and min admission rate from environment variables
	maxRateStr := os.Getenv("MAX_ADMISSION_RATE")
	if maxRateStr != "" {
		maxRate, err := strconv.Atoi(maxRateStr)
		if err != nil {
			log.Printf("⚠️ Invalid MAX_ADMISSION_RATE value: %v. Using default: 100", err)
			MaxAdmissionRate = 100
		} else {
			MaxAdmissionRate = maxRate
		}
	} else {
		MaxAdmissionRate = 100
	}

	minRateStr := os.Getenv("MIN_ADMISSION_RATE")
	if minRateStr != "" {
		minRate, err := strconv.Atoi(minRateStr)
		if err != nil {
			log.Printf("⚠️ Invalid MIN_ADMISSION_RATE value: %v. Using default: 1", err)
			MinAdmissionRate = 1
		} else {
			MinAdmissionRate = minRate
		}
	} else {
		MinAdmissionRate = 1
	}

	log.Printf("📋 Admission Rate Config: min=%d, max=%d", MinAdmissionRate, MaxAdmissionRate)

	// Load the parameters for each service from environment variables
	for i := 0; i < NumServices; i++ {
		serviceIndex := i // 0-based index for array access, 1-based for logs

		// Load Initial CurrWeight
		currWeightStr := os.Getenv(fmt.Sprintf("SERVICE%d_INITIAL_CURR_WEIGHT", serviceIndex+1)) // Use serviceIndex + 1 for environment variable name
		currWeight, err := strconv.Atoi(currWeightStr)
		if err != nil || currWeight <= 0 {
			log.Printf("⚠️ Invalid or missing value for SERVICE%d_INITIAL_CURR_WEIGHT. Using default: %d", serviceIndex+1, 10*(serviceIndex+1))
			InitialCurrWeights[serviceIndex] = 10 * (serviceIndex + 1)
		} else {
			InitialCurrWeights[serviceIndex] = currWeight
		}

		// Load Initial EmptyQWeight
		emptyQWeightStr := os.Getenv(fmt.Sprintf("SERVICE%d_INITIAL_EMPTYQ_WEIGHT", serviceIndex+1))
		emptyQWeight, err := strconv.Atoi(emptyQWeightStr)
		if err != nil || emptyQWeight <= 0 {
			log.Printf("⚠️ Invalid or missing value for SERVICE%d_INITIAL_EMPTYQ_WEIGHT. Using default: %d", serviceIndex+1, 10+(serviceIndex+1)-1)
			InitialEmptyQWeights[serviceIndex] = 10 + (serviceIndex + 1) - 1
		} else {
			InitialEmptyQWeights[serviceIndex] = emptyQWeight
		}

		// Load Raw Admission Rate
		rawAdmissionRateStr := os.Getenv(fmt.Sprintf("SERVICE%d_RAW_ADMISSION_RATE", serviceIndex+1))
		rawAdmissionRate, err := strconv.Atoi(rawAdmissionRateStr)
		if err != nil || rawAdmissionRate <= 0 {
			log.Printf("⚠️ Invalid or missing value for SERVICE%d_RAW_ADMISSION_RATE. Using default: %d", serviceIndex+1, 10+(serviceIndex+1)-1)
			RawAdmissionRates[serviceIndex] = 10 + (serviceIndex + 1) - 1
		} else {
			RawAdmissionRates[serviceIndex] = rawAdmissionRate
		}

		// Load Alpha
		alphaStr := os.Getenv(fmt.Sprintf("SERVICE%d_ALPHA", serviceIndex+1))
		alpha, err := strconv.Atoi(alphaStr)
		if err != nil || alpha <= 0 {
			log.Printf("⚠️ Invalid or missing value for SERVICE%d_ALPHA. Using default: %d", serviceIndex+1, 3+(serviceIndex+1)-1)
			Alphas[serviceIndex] = 3 + (serviceIndex + 1) - 1
		} else {
			Alphas[serviceIndex] = alpha
		}

		// Load Beta
		betaStr := os.Getenv(fmt.Sprintf("SERVICE%d_BETA", serviceIndex+1))
		beta, err := strconv.ParseFloat(betaStr, 64)
		if err != nil || beta <= 0 {
			log.Printf("⚠️ Invalid or missing value for SERVICE%d_BETA. Using default: 0.5", serviceIndex+1)
			Betas[serviceIndex] = 0.5
		} else {
			Betas[serviceIndex] = beta
		}
	}

	log.Println("✅ All service-specific parameters loaded")

}
