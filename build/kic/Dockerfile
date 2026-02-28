# build/kic/Dockerfile — KIC compliance assessments service
# CGO_ENABLED=1 required for go-duckdb (embedded OLAP analytics engine).
FROM golang:1.24-alpine AS builder
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -mod=mod \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/kic \
    ./cmd/kic

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=builder /out/kic /kic
EXPOSE 8082
ENTRYPOINT ["/kic"]
