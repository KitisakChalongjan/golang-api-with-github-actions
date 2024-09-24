# Step 1: Build the Go application
FROM golang:1.23.1 AS builder
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o golang-api .

# Step 2: Create a lightweight image to run the service
FROM alpine:latest
WORKDIR /root/

# Copy the compiled Go binary from the builder image
COPY --from=builder /app/golang-api .

# Expose the port your service uses (if necessary)
EXPOSE 8000

# Command to run the service
CMD ["./golang-api"]