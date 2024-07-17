package processor

import (
	"encoding/base64"
	"fmt"
	"log"
	"sync"

	"github.com/wimspaargaren/yolov3"
	"gocv.io/x/gocv"
	"consumer/config"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type CloudEventData struct {
	ImageData string `json:"imageData"`
}

func StartProcessor(workerID int, wg *sync.WaitGroup) {
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

func createYoloNet() (yolov3.Net, error) {
	return yolov3.NewNet(config.YOLOv3WeightsPath, config.YOLOv3ConfigPath, config.COCONamesPath)
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

func Display(event cloudevents.Event) {
	config.RequestQueue <- event
	log.Println("📥 Event queued for processing")
}