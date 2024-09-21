package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

var (
	UpdateInterval time.Duration
	ServiceName    string
	Alpha          float64
	Beta           float64
	ServiceURL     string // URL for the consuming service

	RedisURL  string
	RedisPass string
)

func LoadConfig() {
	var err error

	ServiceName = os.Getenv("SERVICE_NAME")
	if ServiceName == "" {
		log.Fatal("❌ SERVICE_NAME environment variable is not set")
	}

	ServiceURL = os.Getenv("SERVICE_URL")
	if ServiceURL == "" {
		log.Fatal("❌ SERVICE_URL environment variable is not set")
	}

	alphaStr := os.Getenv("ALPHA")
	if alphaStr == "" {
		Alpha = 0.1
	} else {
		Alpha, err = strconv.ParseFloat(alphaStr, 64)
		if err != nil {
			log.Fatalf("❌ Invalid ALPHA value: %v", err)
		}
	}

	betaStr := os.Getenv("BETA")
	if betaStr == "" {
		Beta = 0.5
	} else {
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
}
