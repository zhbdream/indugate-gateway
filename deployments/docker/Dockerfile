# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/gateway \
    ./cmd/gateway

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /build/bin/gateway /app/gateway
COPY configs/ /app/configs/

RUN mkdir -p /app/data && \
    addgroup -S indugate && \
    adduser -S indugate -G indugate && \
    chown -R indugate:indugate /app

USER indugate

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gateway"]
