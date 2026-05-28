# Stage 1: Build static assets
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Compile the high-performance Go application
FROM golang:1.22.0-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy static assets generated from Stage 1 into the Go build path
COPY --from=web-builder /app/web/dist /app/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/nexusai-gateway ./cmd/gateway

# Stage 3: Minimal hardened runner package
FROM alpine:3.19 AS runner
RUN apk add --no-cache ca-certificates tzdata

# Create non-root system user and group
RUN addgroup -S -g 10001 appgroup && \
    adduser -S -u 10001 -G appgroup appuser

WORKDIR /app
COPY --from=go-builder --chown=appuser:appgroup /app/bin/nexusai-gateway /app/nexusai-gateway

# Set standard environment variables with configurable defaults
ENV PORT=20129
ENV DATABASE_URL="postgres://postgres:postgres_secure_pass@postgres-nexus:5432/nexusai_gateway?sslmode=disable"
ENV REDIS_URL="redis://redis-nexus:6379/1"
ENV OIDC_ISSUER="http://localhost:20129"

# Expose port and run as non-root user
EXPOSE 20129
USER 10001:10001

ENTRYPOINT ["/app/nexusai-gateway"]
