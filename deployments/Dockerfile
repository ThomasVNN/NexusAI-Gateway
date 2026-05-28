# Stage 1: Build static assets
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN if [ -f package.json ]; then npm install || echo "Fallback"; fi
COPY web/ ./
RUN if [ -d src ]; then npm run build; else mkdir -p dist && echo "<h1>Placeholder</h1>" > dist/index.html; fi

# Stage 2: Compile the high-performance Go application
FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download || echo "No dependencies"
COPY . .
# Copy static assets generated from Stage 1 into the Go build path
COPY --from=web-builder /app/web/dist /app/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/nexusai-gateway ./cmd/gateway

# Stage 3: Minimal hardened runner package
FROM alpine:3.19 AS runner
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/bin/nexusai-gateway /app/nexusai-gateway

EXPOSE 20129
ENV PORT=20129
ENV DATABASE_URL=postgres://postgres:postgres@localhost:5432/vectors?sslmode=disable

CMD ["/app/nexusai-gateway"]
