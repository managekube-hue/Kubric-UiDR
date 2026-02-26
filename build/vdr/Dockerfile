# build/vdr/Dockerfile — VDR vulnerability findings service
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/vdr \
    ./cmd/vdr

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/vdr /vdr
EXPOSE 8081
ENTRYPOINT ["/vdr"]
