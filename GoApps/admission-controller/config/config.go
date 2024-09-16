package config

import (
	"log"
	"os"
	"strconv"
)

var (
	ServiceName string
	Alpha       float64
	Beta        float64

	RedisURL  string
	RedisPass string
)

func LoadConfig() {
	ServiceName = os.Getenv("SERVICE_NAME")
	if ServiceName == "" {
		log.Fatal("❌ SERVICE_NAME environment variable is not set")
	}

	alphaStr := os.Getenv("ALPHA")
	if alphaStr == "" {
		Alpha = 3
	} else {
		var err error
		Alpha, err = strconv.ParseFloat(alphaStr, 64)
		if err != nil {
			log.Fatalf("❌ Invalid ALPHA value: %v", err)
		}
	}

	betaStr := os.Getenv("BETA")
	if betaStr == "" {
		Beta = 0.5
	} else {
		var err error
		Beta, err = strconv.ParseFloat(betaStr, 64)
		if err != nil {
			log.Fatalf("❌ Invalid BETA value: %v", err)
		}
	}

	RedisURL = os.Getenv("REDIS_URL")
	if RedisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is not set")
	}
	RedisPass = os.Getenv("REDIS_PASSWORD")
	if RedisPass == "" {
		log.Fatal("❌ REDIS_PASSWORD environment variable is not set")
	}

	log.Printf("✅ Configuration loaded: SERVICE_NAME=%s, ALPHA=%.2f, BETA=%.2f", ServiceName, Alpha, Beta)
}
