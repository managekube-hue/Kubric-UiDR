# build/ksvc/Dockerfile — K-SVC tenant management service
# Multi-stage: builder compiles Go binary; final image is distroless.

# ── Stage 1: builder ─────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /build

# Cache module downloads before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy all source
COPY . .

# Build the ksvc binary (statically linked)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/ksvc \
    ./cmd/ksvc

# ── Stage 2: final ───────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/ksvc /ksvc
EXPOSE 8080
ENTRYPOINT ["/ksvc"]
