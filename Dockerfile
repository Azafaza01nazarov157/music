FROM golang:1.23-alpine AS builder

RUN apk update && apk add --no-cache ca-certificates git

# Install FFmpeg with all necessary dependencies
RUN apk add --no-cache ffmpeg \
    libass \
    libbluray \
    libdrm \
    libtheora \
    libvorbis \
    libvpx \
    libx264 \
    libx265 \
    opus \
    sdl2 \
    x264-dev \
    x265-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build both applications
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/processor ./cmd/processor
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app ./cmd/app

FROM alpine:3.16

# Install FFmpeg with all necessary dependencies in the runtime image
RUN apk update && apk add --no-cache \
    ffmpeg \
    libass \
    libbluray \
    libdrm \
    libtheora \
    libvorbis \
    libvpx \
    libx264 \
    libx265 \
    opus \
    sdl2 \
    ca-certificates

WORKDIR /app

# Copy both binaries
COPY --from=builder /app/bin/processor .
COPY --from=builder /app/bin/app .

# Copy environment files
COPY application.env .
COPY .env .

ENV GIN_MODE=release

# Use environment variables from docker-compose
ENV DB_HOST=${DB_HOST}
ENV DB_PORT=${DB_PORT}
ENV DB_USER=${DB_USER}
ENV DB_PASSWORD=${DB_PASSWORD}
ENV DB_NAME=${DB_NAME}
ENV REDIS_HOST=${REDIS_HOST}
ENV MINIO_HOST=${MINIO_HOST}
ENV KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS}

# Verify FFmpeg installation
RUN ffmpeg -version

# Default command will run the processor
CMD ["./processor"] 