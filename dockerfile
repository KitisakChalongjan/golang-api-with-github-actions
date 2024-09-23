# Step 1: Build the Go application
FROM golang:1.21 AS builder
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o myapp .

# Step 2: Create a lightweight image to run the service
FROM alpine:latest
WORKDIR /root/

# Copy the compiled Go binary from the builder image
COPY --from=builder /app/myapp .

# Expose the port your service uses (if necessary)
EXPOSE 8080

# Command to run the service
CMD ["./myapp"]