# Use the official Go image as the base image
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /build

# Copy the Go module files
COPY go.mod go.sum ./

# Download and install the Go dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go program
RUN go build -o app

FROM alpine:latest
WORKDIR /
COPY --from=builder /build/app ./

# Set the entry point for the container
ENTRYPOINT ["./app"]
