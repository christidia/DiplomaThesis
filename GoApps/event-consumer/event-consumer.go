package main

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "sync"

    "gocv.io/x/gocv"
    "github.com/wimspaargaren/yolov3"
    cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
    yolov3ConfigPath  = "/go/src/app/yolov3-tiny.cfg"
    yolov3WeightsPath = "/go/src/app/yolov3-tiny.weights"
    cocoNamesPath     = "/go/src/app/coco.names"
    yolonetInitOnce   sync.Once
    yolonet           yolov3.Net
    requestQueue      chan cloudevents.Event
)

type CloudEventData struct {
    ImageData string `json:"imageData"`
}

func init() {
    queueSizeStr := os.Getenv("QUEUE_SIZE")
    queueSize, err := strconv.Atoi(queueSizeStr)
    if err != nil {
        queueSize = 100 // default queue size
    }
    requestQueue = make(chan cloudevents.Event, queueSize)
}

func initYoloNet() {
    var err error
    yolonet, err = yolov3.NewNet(yolov3WeightsPath, yolov3ConfigPath, cocoNamesPath)
    if err != nil {
        log.Fatalf("❌ Unable to create YOLO net: %v", err)
    }
}

func processImage(event cloudevents.Event) {
    // Ensure YOLOv3 net is initialized only once
    yolonetInitOnce.Do(initYoloNet)

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

func startProcessor() {
    for event := range requestQueue {
        processImage(event)
    }
}

func display(event cloudevents.Event) {
    select {
    case requestQueue <- event:
        log.Println("📥 Event queued for processing")
    default:
        log.Println("⚠️ Request queue is full, discarding event")
    }
}

func main() {
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

    // Start the processor goroutine
    go startProcessor()

    // Start the receiver
    if err := c.StartReceiver(ctx, display); err != nil {
        log.Fatalf("❌ Error during receiver's runtime: %v", err)
    }
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
                logRequest(req)
            }
            next.ServeHTTP(w, req)
        })
    }
}

func logRequest(req *http.Request) {
    b, err := json.MarshalIndent(toReq(req), "", "  ")
    if err != nil {
        log.Println("⚠️ Failed to marshal request:", err)
    }

    log.Println(string(b))
}

type LoggableRequest struct {
    Method           string      `json:"method,omitempty"`
    URL              *url.URL    `json:"URL,omitempty"`
    Proto            string      `json:"proto,omitempty"`
    ProtoMajor       int         `json:"protoMajor,omitempty"`
    ProtoMinor       int         `json:"protoMinor,omitempty"`
    Header           http.Header `json:"headers,omitempty"`
    Body             string      `json:"body,omitempty"`
    ContentLength    int64       `json:"contentLength,omitempty"`
    TransferEncoding []string    `json:"transferEncoding,omitempty"`
    Host             string      `json:"host,omitempty"`
    Trailer          http.Header `json:"trailer,omitempty"`
    RemoteAddr       string      `json:"remoteAddr"`
    RequestURI       string      `json:"requestURI"`
}

func toReq(req *http.Request) LoggableRequest {
    body, err := io.ReadAll(req.Body)
    if err != nil {
        log.Println("⚠️ Failed to read request body")
    }
    _ = req.Body.Close()
    // Replace the body with a new reader after reading from the original
    req.Body = io.NopCloser(bytes.NewBuffer(body))
    return LoggableRequest{
        Method:           req.Method,
        URL:              req.URL,
        Proto:            req.Proto,
        ProtoMajor:       req.ProtoMajor,
        ProtoMinor:       req.ProtoMinor,
        Header:           req.Header,
        Body:             string(body),
        ContentLength:    req.ContentLength,
        TransferEncoding: req.TransferEncoding,
        Host:             req.Host,
        Trailer:          req.Trailer,
        RemoteAddr:       req.RemoteAddr,
        RequestURI:       req.RequestURI,
    }
}
