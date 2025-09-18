# Builder stage
FROM golang:1.23.6-alpine AS builder

# Build arguments
ARG GITHUB_TOKEN
ARG VERSION=unknown

# Environment variables for optimized build
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on \
    TOKEN=$GITHUB_TOKEN \
    VERSION=${VERSION}

# Install minimal dependencies including UPX
RUN apk add --no-cache git upx

# Configure git for private repositories
RUN git config --global url."https://${TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Set working directory
WORKDIR /app

# Configure private module access
RUN go env -w GOPRIVATE=github.com/RodolfoBonis/go_key_guardian

# Copy dependency files first (better layer caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install swag for API documentation
# RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy source code
COPY . .

# Update dependencies
RUN go mod tidy

# Generate swagger documentation (temporarily disabled)
# RUN /go/bin/swag init

# Build the application with maximum optimization
RUN go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o spooliq \
    ./main.go

# Compress binary with UPX (maximum compression)
RUN upx --best --lzma spooliq

# Production stage - using scratch for absolute minimal size
FROM scratch AS production

# Metadata
ARG VERSION=unknown
LABEL version=${VERSION}
LABEL maintainer="RodolfoBonis"

# Copy only the compressed binary
COPY --from=builder /app/spooliq /spooliq

# Expose port
EXPOSE 8000

# Run the application
ENTRYPOINT ["/spooliq"]