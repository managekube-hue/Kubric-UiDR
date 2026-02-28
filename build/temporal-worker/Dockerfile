# build/temporal-worker/Dockerfile — Temporal workflow worker (Patch + Remediation)
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/temporal-worker \
    ./cmd/temporal

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/temporal-worker /temporal-worker
ENTRYPOINT ["/temporal-worker"]
