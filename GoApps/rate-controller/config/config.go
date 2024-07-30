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
}
