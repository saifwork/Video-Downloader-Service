# Use official Golang image
FROM golang:1.20 AS builder

# Set working directory
WORKDIR /app

# Install dependencies (FFmpeg + yt-dlp)
RUN apt-get update && apt-get install -y ffmpeg curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

# Copy Go project files
COPY . .

# Build the Go application
RUN go build -o bot main.go

# Final lightweight image
FROM debian:stable-slim

# Install FFmpeg & yt-dlp in the final container
RUN apt-get update && apt-get install -y ffmpeg curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

# Set working directory
WORKDIR /app

# Copy the compiled Go binary from the builder stage
COPY --from=builder /app/bot /app/bot

# Expose port (if needed)
EXPOSE 8080

# Run the bot
CMD ["/app/bot"]