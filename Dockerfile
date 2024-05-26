# Build Stage
FROM docker.io/golang:1.22.3-bookworm AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Ensure the module directory is writable
RUN chmod -R 777 /go/pkg/mod

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main ./cmd/worker


# Serve Stage
FROM docker.io/golang:1.22.3-bookworm

# Set necessary environment variables
ENV PORT=8080

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
