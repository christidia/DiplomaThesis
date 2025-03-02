# Knative Image Processing Service
This is a serverless image processing service designed to run on Knative (or other serveless environments). The service receives images via CloudEvents, applies transformations, and logs processing completion without saving any files. It is optimized for high CPU utilization to simulate real-world workloads.

It is based on the image processing function application provided in https://github.com/ddps-lab/serverless-faas-workbench. 

## 📌 Overview
-  **Event-Driven**: Listens for CloudEvents sent to a RabbitMQ broker.
- **CPU-Intensive Processing**:
  - Flipping (left-right, top-bottom)
  - Rotating (90°, 180°, 270°)
  - Filtering (blur, contour, sharpen)
  - Grayscale conversion
  - Resizing (128x128)
- **No File Storage**: Processed images are not saved, making it ideal for benchmarking CPU workloads.

## 🔄 How It Works
1. Receives image processing requests via CloudEvents.
2. Extracts an image URL from the event payload.
3. Downloads the image from the provided URL.
4. Performs CPU-heavy processing on the image.
5. Returns processing metadata (processing time, filename, event ID).
6. No files are saved—all operations happen in memory.

## 📝 Input & Output Details
### 🎯 Input (CloudEvent)
- Content-Type: application/json
- Expected Payload Format:

```
{
  "image_url": "https://picsum.photos/200/300"
}
```

### 📦 Output
- Responds with JSON metadata:
```
{
  "status": "success",
  "processing_time": 1.234,
  "service": "image-processing",
  "event_id": "12345",
  "original_filename": "random.jpg"
}
```

* _Note: No images are saved or returned._
