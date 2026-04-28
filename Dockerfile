# Build stage
FROM golang:1.25.9-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o conevent ./cmd/conevent

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/conevent .

EXPOSE 3000

CMD ["./conevent"]
