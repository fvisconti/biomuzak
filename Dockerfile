# Placeholder Dockerfile for the Go API
FROM golang:1.21-alpine

WORKDIR /app

# The Go application will be built here in future tasks
# For now, this is just a placeholder to make docker-compose happy
CMD ["echo", "API service placeholder"]
