# Use the official golang image as the base image
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go application source code into the container
COPY go.* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN go build -o service

# Use a smaller base image for the final container
FROM alpine:latest

# Set the working directory inside the final container
WORKDIR /app

# Copy only the compiled binary from the builder image
COPY --from=builder /app/service .

# Expose the port the application will run on
EXPOSE 8080

# Command to run the application
CMD ["./service"]
