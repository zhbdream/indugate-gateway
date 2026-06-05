# Frontend build stage
FROM node:20-alpine AS web-builder

WORKDIR /web
COPY web/package.json ./
RUN npm install --registry=https://registry.npmmirror.com

COPY web/ ./
RUN npm run build

# Backend build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY . .
COPY --from=web-builder /web/dist ./web/dist

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/bin/gateway \
    ./cmd/gateway

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite-libs wget

ENV TZ=Asia/Shanghai
ENV INDUGATE_SERVER_MODE=release
ENV INDUGATE_DATABASE_DRIVER=sqlite
ENV INDUGATE_DATABASE_DSN=/app/data/indugate.db
ENV INDUGATE_WEB_STATIC_DIR=/app/web/dist
ENV INDUGATE_SIMULATOR_OPCUA_AUTO_START=true
ENV INDUGATE_SIMULATOR_MODBUS_AUTO_START=true
ENV INDUGATE_SIMULATOR_MQTT_AUTO_START=true

WORKDIR /app

COPY --from=builder /build/bin/gateway /app/gateway
COPY --from=builder /build/configs/ /app/configs/
COPY --from=builder /build/web/dist /app/web/dist

RUN mkdir -p /app/data && \
    addgroup -S indugate && \
    adduser -S indugate -G indugate && \
    chown -R indugate:indugate /app

USER indugate

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gateway"]
