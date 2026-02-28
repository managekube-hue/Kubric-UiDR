# build/kic/Dockerfile — KIC compliance assessments service
FROM golang:1.24-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod \
    -ldflags="-s -w" \
    -o /out/kic \
    ./cmd/kic

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/kic /kic
EXPOSE 8082
ENTRYPOINT ["/kic"]
