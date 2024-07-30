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
)

func LoadConfig() {
	updateIntervalStr := os.Getenv("UPDATE_INTERVAL")
	if updateIntervalStr == "" {
		log.Fatal("❌ UPDATE_INTERVAL environment variable is not set")
	}
	updateInterval, err := strconv.Atoi(updateIntervalStr)
	if err != nil {
		log.Fatalf("❌ Invalid UPDATE_INTERVAL value: %v", err)
	}
	UpdateInterval = time.Duration(updateInterval) * time.Second

	ServiceName = os.Getenv("SERVICE_NAME")
	if ServiceName == "" {
		log.Fatal("❌ SERVICE_NAME environment variable is not set")
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
}
