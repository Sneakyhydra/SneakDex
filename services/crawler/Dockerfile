# Use Go with Alpine
FROM golang:1.24-alpine

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata curl

# Install Air
RUN go install github.com/air-verse/air@latest

# Set working directory
WORKDIR /app

# Add Go bin to PATH
ENV PATH="/go/bin:$PATH"
ENV GO_ENV=development

# Copy go.mod and go.sum first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code (Fallback for volume mounts)
COPY . .

# Create tmp directory for Air
RUN mkdir -p tmp

# Default command runs Air for hot reload
CMD ["air", "-c", ".air.toml"]