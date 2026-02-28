# build/noc/Dockerfile — NOC cluster & agent heartbeat service
FROM golang:1.25-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/noc \
    ./cmd/noc

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/noc /noc
EXPOSE 8083
ENTRYPOINT ["/noc"]
