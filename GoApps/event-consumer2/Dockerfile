# Use the pre-built OpenCV image
FROM ghcr.io/hybridgroup/opencv:4.9.0

# Set Go environment
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Set the working directory inside the container
WORKDIR /go/src/app

# Install Git
RUN apt-get update && apt-get install -y git

# Clone the GoCV repository
RUN git clone https://github.com/hybridgroup/gocv.git /go/src/gocv.io/x/gocv

# Copy the necessary configuration files for YOLOv3
COPY yolov3-tiny.cfg yolov3-tiny.weights coco.names ./

# Copy the Go application source code to the container
COPY . .

# Install Go dependencies
RUN go mod download

# Build the Go application
RUN go build -o myapp .

# Expose the port that your Go app runs on (if any)
EXPOSE 8080

# Specify the command to run the Go application
CMD ["./myapp"]
