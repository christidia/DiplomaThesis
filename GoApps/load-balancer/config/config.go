package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

var (
	RedisURL              string
	RedisPass             string
	RabbitMQURL           string
	RabbitMQUser          string
	RabbitMQPass          string
	CheckInterval         time.Duration
	AdmissionRateInterval time.Duration
	NumServices           int
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
	RabbitMQURL = os.Getenv("RABBITMQ_URL")
	if RabbitMQURL == "" {
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
}
