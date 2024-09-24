# Step 1: Build the Go application
FROM golang:1.23.1 AS builder

# Set the current working directory inside the builder container
WORKDIR /app

# Copy go mod and sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Set CGO_ENABLED to 0 for building a statically linked binary
ENV CGO_ENABLED=0

# Build the Go application
RUN go build -o golang-api .

# Step 2: Create a lightweight image to run the service
FROM alpine:latest

# Install necessary packages (if needed)
# For example, if you're using networking features
RUN apk --no-cache add ca-certificates

# Set the working directory for the final image
WORKDIR /root/

# Copy the compiled binary from the builder image
COPY --from=builder /app/golang-api .

# Ensure the binary has the correct permissions
RUN chmod +x golang-api

# Expose the port your service uses (if necessary)
EXPOSE 8000

# Command to run the service
CMD ["./golang-api"]