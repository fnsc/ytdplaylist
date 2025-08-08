FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN apk add --no-cache bash curl ffmpeg && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod +x /usr/local/bin/yt-dlp
