FROM golang:1.20-alpine AS builder

RUN apk update && apk add --no-cache ca-certificates git

RUN apk add --no-cache ffmpeg

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/processor ./cmd/processor

FROM alpine:3.16

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/processor .

COPY application.env .
COPY .env .

ENV GIN_MODE=release

CMD ["./processor"] 