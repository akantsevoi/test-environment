# Build stage
FROM golang:1.23 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /maroon ./cmd/app

# Run stage
FROM alpine:latest

COPY --from=builder /maroon /maroon

CMD ["/maroon"]