FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/wallet-nutrition-score ./cmd/app

FROM alpine:latest

WORKDIR /root/

# Copy the executable from builder stage
COPY --from=builder /app/wallet-nutrition-score .

# Copy configuration
COPY config/config.yaml config/
COPY .env .

# Copy Swagger files
COPY docs/swagger.json docs/
COPY docs/swagger.yaml docs/

EXPOSE 8080

# Run the application
CMD ["./wallet-nutrition-score"]
