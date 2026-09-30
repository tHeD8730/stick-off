FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o nim-server .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/nim-server .
COPY frontend/ ./frontend/
EXPOSE 8080
CMD ["./nim-server"]
