package config

import (
	"log"
	"os"
	"strconv"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
	YOLOv3ConfigPath      = "/go/src/app/yolov3-tiny.cfg"
	YOLOv3WeightsPath     = "/go/src/app/yolov3-tiny.weights"
	COCONamesPath         = "/go/src/app/coco.names"
	RequestQueue          chan cloudevents.Event
	NumWorkers            int
	RequestLoggingEnabled bool
	ServiceName           string
)

func LoadConfig() {
	queueSizeStr := os.Getenv("QUEUE_SIZE")
	queueSize, err := strconv.Atoi(queueSizeStr)
	if err != nil {
		queueSize = 100 // default queue size
	}
	RequestQueue = make(chan cloudevents.Event, queueSize)

	numWorkersStr := os.Getenv("NUM_WORKERS")
	NumWorkers, err = strconv.Atoi(numWorkersStr)
	if err != nil || NumWorkers < 1 {
		NumWorkers = 4 // default number of workers
	}

	ServiceName = os.Getenv("SERVICE_NAME")
	if ServiceName == "" {
		log.Println("⚠️ Service Name Environmental Var is not declared.")
	}

	RequestLoggingEnabled, _ = strconv.ParseBool(os.Getenv("REQUEST_LOGGING_ENABLED"))
	if RequestLoggingEnabled {
		log.Println("🔍 Request logging enabled, request logging is not recommended for production since it might log sensitive information")
	}
}
