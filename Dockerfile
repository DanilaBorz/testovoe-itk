FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
COPY config.env .

EXPOSE 8080

CMD ["./server"]