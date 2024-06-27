package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"gocv.io/x/gocv"
)

// CloudEventData represents the data of a CloudEvent
type CloudEventData struct {
	ImageData string `json:"imageData"`
}

func main() {
	// Specify the directory containing images and the output file for CloudEvents
	dirPath := "~/christina/yolo/test_images"
	outputFile := "events.json"

	// Expand the tilde (~) in the file path
	dirPath, err := ExpandTilde(dirPath)
	if err != nil {
		log.Fatal("Error expanding tilde:", err)
	}

	// Get list of image files in directory
	imageFiles, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}

	// Create an array to hold CloudEvents
	var cloudEvents []cloudevents.Event

	// Loop through image files
	for _, fileInfo := range imageFiles {
		// Check if the file is an image (JPEG, PNG, etc.)
		if strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".jpg") ||
			strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".png") {
			// Read image data
			imageData, err := ioutil.ReadFile(filepath.Join(dirPath, fileInfo.Name()))
			if err != nil {
				log.Printf("Error reading image file %s: %v", fileInfo.Name(), err)
				continue
			}
			// Encode image data as base64
			encodedImageData := base64.StdEncoding.EncodeToString(imageData)

			// Create CloudEvent
			ce := cloudevents.NewEvent()
			ce.SetID(uuid.New().String())
			ce.SetType("com.example.image")
			ce.SetSource("example.com")

			// Create CloudEvent data
			eventData := CloudEventData{
				ImageData: encodedImageData,
			}

			// Create a new gocv.Mat from the image data
			frame, err := gocv.IMDecode(imageData, gocv.IMReadColor)
			if err != nil {
				log.Printf("Error decoding image file %s: %v", fileInfo.Name(), err)
				continue
			}

			// Check if the image size is empty
			if frame.Empty() {
				log.Printf("Image file %s is empty, skipping this file", fileInfo.Name())
				continue // Move on to the next file in the directory
			}

			// Marshal CloudEvent data to JSON
			eventDataBytes, err := json.Marshal(eventData)
			if err != nil {
				log.Printf("Error marshaling JSON for file %s: %v", fileInfo.Name(), err)
				continue
			}

			// Set JSON data as CloudEvent data
			ce.SetData(cloudevents.ApplicationJSON, eventDataBytes)

			// Append CloudEvent to array
			cloudEvents = append(cloudEvents, ce)
		}
	}

	// Marshal array of CloudEvents to JSON
	eventsJSON, err := json.MarshalIndent(cloudEvents, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling CloudEvents to JSON: %v", err)
	}

	// Write JSON data to output file
	err = ioutil.WriteFile(outputFile, eventsJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing JSON data to output file: %v", err)
	}

	fmt.Printf("CloudEvents written to %s\n", outputFile)
}

// ExpandTilde expands the tilde (~) in a file path to the user's home directory.
func ExpandTilde(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[1:]), nil
	}
	return path, nil
}
