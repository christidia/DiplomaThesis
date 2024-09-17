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
}
