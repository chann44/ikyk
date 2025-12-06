FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy shared package
COPY pkg/ ./pkg/

# Copy gateway code
COPY gateway/ ./gateway/

WORKDIR /build/gateway

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/cli

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/gateway/gateway .

EXPOSE 8080

CMD ["./gateway"]
