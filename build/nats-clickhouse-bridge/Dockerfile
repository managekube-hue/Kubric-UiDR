# build/nats-clickhouse-bridge/Dockerfile — NATS JetStream to ClickHouse bridge
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/nats-clickhouse-bridge \
    ./cmd/nats-clickhouse-bridge

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/nats-clickhouse-bridge /nats-clickhouse-bridge
ENTRYPOINT ["/nats-clickhouse-bridge"]
