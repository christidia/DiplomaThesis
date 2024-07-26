package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"consumer/config"
	"consumer/logging"
	"consumer/metrics"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/wimspaargaren/yolov3"
	"gocv.io/x/gocv"
)

var (
	yolov3ConfigPath  = "/go/src/app/yolov3-tiny.cfg"
	yolov3WeightsPath = "/go/src/app/yolov3-tiny.weights"
	cocoNamesPath     = "/go/src/app/coco.names"
)

type CloudEventData struct {
	ImageData string `json:"imageData"`
}

func createYoloNet() (yolov3.Net, error) {
	return yolov3.NewNet(yolov3WeightsPath, yolov3ConfigPath, cocoNamesPath)
}

func processImage(event cloudevents.Event, yolonet yolov3.Net) {
	// Parse the CloudEvent payload
	var eventData CloudEventData
	if err := event.DataAs(&eventData); err != nil {
		log.Printf("⚠️ Error parsing CloudEvent data: %v", err)
		return
	}

	// Decode the base64-encoded image data
	decodedImageData, err := base64.StdEncoding.DecodeString(eventData.ImageData)
	if err != nil {
		log.Printf("❌ Unable to decode image data: %v", err)
		return
	}

	// Check if the decoded image data is empty
	if len(decodedImageData) == 0 {
		log.Println("⚠️ Decoded image data is empty")
		return
	}

	// Create a new gocv.Mat from the image data
	frame, err := gocv.IMDecode(decodedImageData, gocv.IMReadColor)
	if err != nil {
		log.Printf("❌ Unable to decode image: %v", err)
		return
	}
	defer frame.Close()

	// Check if the image size is empty
	if frame.Empty() {
		log.Println("⚠️ Image size is empty")
		return
	}

	// Perform image detection
	detections, err := yolonet.GetDetections(frame)
	if err != nil {
		log.Printf("❌ Unable to retrieve predictions: %v", err)
		return
	}

	// Print detections to the screen
	fmt.Println("--- Image Detection Results ---")
	for _, detection := range detections {
		fmt.Printf("🔍 Class: %s, Confidence: %f, Bounding Box: %v\n", detection.ClassName, detection.Confidence, detection.BoundingBox)
	}
	fmt.Println("------")
}

func startProcessor(workerID int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Initialize YOLO net for this worker
	yolonet, err := createYoloNet()
	if err != nil {
		log.Fatalf("❌ Worker %d: Unable to create YOLO net: %v", workerID, err)
	}

	for event := range config.RequestQueue {
		processImage(event, yolonet)
	}
}

func display(event cloudevents.Event) {
	config.RequestQueue <- event
	log.Println("📥 Event queued for processing")
	metrics.UpdateMetric(float64(len(config.RequestQueue)))
}

func main() {
	config.LoadConfig()

	// Initialize and start the metrics server
	metrics.InitMetrics()
	go metrics.StartMetricsServer()

	run(context.Background())
}

func run(ctx context.Context) {

	// Check if request logging is enabled
	requestLoggingEnabled, _ := strconv.ParseBool(os.Getenv("REQUEST_LOGGING_ENABLED"))
	if requestLoggingEnabled {
		log.Println("🔍 Request logging enabled, request logging is not recommended for production since it might log sensitive information")
	}

	// Create CloudEvents client
	c, err := cloudevents.NewClientHTTP(
		cloudevents.WithMiddleware(healthzMiddleware),
		cloudevents.WithMiddleware(requestLoggingMiddleware(requestLoggingEnabled)),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	// Start the processor goroutines
	var wg sync.WaitGroup
	for i := 0; i < config.NumWorkers; i++ {
		wg.Add(1)
		go startProcessor(i, &wg)
	}

	// Start the receiver
	if err := c.StartReceiver(ctx, display); err != nil {
		log.Fatalf("❌ Error during receiver's runtime: %v", err)
	}

	// Wait for all workers to finish
	wg.Wait()
}

// HTTP path of the health endpoint used for probing the service.
const healthzPath = "/healthz"

// healthzMiddleware is a cloudevents.Middleware which exposes a health endpoint.
func healthzMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.RequestURI == healthzPath {
			w.WriteHeader(http.StatusNoContent)
		} else {
			next.ServeHTTP(w, req)
		}
	})
}

// requestLoggingMiddleware is a cloudevents.Middleware which logs incoming requests.
func requestLoggingMiddleware(enabled bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if enabled {
				logging.LogRequest(req)
			}
			next.ServeHTTP(w, req)
		})
	}
}
