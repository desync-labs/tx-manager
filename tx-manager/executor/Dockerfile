# Build Stage
FROM golang:1.22.6 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o executor ./cmd/main.go

# Final Stage
FROM alpine:latest

# Install CA Certificates
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the Pre-built binary from the builder stage
COPY --from=builder /app/executor .

# (Optional) Copy configuration files if needed
# COPY --from=builder /app/config ./config

# Expose port 8080 (Ensure this matches docker-compose.yml)
EXPOSE 50052

# Command to run the executable
CMD ["./executor"]