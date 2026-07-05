# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Allow Go to auto-download required toolchain version
ENV GOTOOLCHAIN=auto

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19

WORKDIR /app

# Create uploads directory
RUN mkdir -p /app/uploads && chmod 777 /app/uploads

# Download Noto Sans TC Chinese font for PDF report generation
# go-pdf/fpdf requires static .ttf/.otf files
# Using GitHub releases from the official notofonts/noto-cjk project
RUN mkdir -p /app/assets/fonts && \
    apk add --no-cache --virtual .font-deps wget unzip ca-certificates && \
    wget --max-redirect=5 -q -O /tmp/NotoSansTC.zip \
      "https://github.com/notofonts/noto-cjk/releases/download/Sans2.004/09_NotoSansCJKtc.zip" && \
    unzip -q -j /tmp/NotoSansTC.zip "NotoSansCJKtc-Regular.otf" -d /app/assets/fonts/ && \
    unzip -q -j /tmp/NotoSansTC.zip "NotoSansCJKtc-Bold.otf" -d /app/assets/fonts/ && \
    mv /app/assets/fonts/NotoSansCJKtc-Regular.otf /app/assets/fonts/NotoSansTC-Regular.ttf && \
    mv /app/assets/fonts/NotoSansCJKtc-Bold.otf /app/assets/fonts/NotoSansTC-Bold.ttf && \
    rm -f /tmp/NotoSansTC.zip && \
    apk del .font-deps || \
    echo "WARNING: Font download failed, PDF Chinese support unavailable"

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy migrations (needed at runtime for DB setup)
COPY --from=builder /app/migrations /app/migrations

# Copy config file
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/server"]
