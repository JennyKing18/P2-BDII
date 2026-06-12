FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

FROM alpine:3.23.4
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]