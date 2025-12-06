FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy shared package
COPY pkg/ ./pkg/

# Copy apps code
COPY apps/ ./apps/

WORKDIR /build/apps

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o management-api ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/apps/management-api .

EXPOSE 9090

CMD ["./management-api"]
