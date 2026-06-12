FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o search-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search-service .

EXPOSE 8080
CMD ["./search-service"]