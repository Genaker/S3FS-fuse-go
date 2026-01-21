FROM golang:1.21-alpine

WORKDIR /app

# Install git and other dependencies
RUN apk add --no-cache git bash

# Copy go mod files first
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate go.sum and verify dependencies
RUN go mod tidy && go mod verify

# Run tests
CMD ["go", "test", "-v", "./..."]
