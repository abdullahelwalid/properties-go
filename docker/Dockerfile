# Use golang base image to build
FROM golang:1.22.10-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache gcc musl-dev

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Start a new stage from scratch
FROM alpine:3.18

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose port 8085 to be accessible from outside
EXPOSE 8085

# Wait for PostgreSQL to be ready before starting the Go app
CMD ["./main"]
