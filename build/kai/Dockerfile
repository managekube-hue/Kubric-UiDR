# build/kai/Dockerfile — KAI Go API gateway (reverse proxy to Python FastAPI)
FROM golang:1.25-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/kai \
    ./cmd/kai

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/kai /kai
EXPOSE 8100
ENTRYPOINT ["/kai"]
