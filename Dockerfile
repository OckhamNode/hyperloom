# Build stage: compile the Go binary in a clean environment
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o hyperloom .

# Runtime stage: scratch-thin container, zero attack surface
FROM alpine:3.20

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/hyperloom .

EXPOSE 8080

ENTRYPOINT ["./hyperloom"]
