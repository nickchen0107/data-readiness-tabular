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

# Install Chromium for PDF generation via chromedp (headless Chrome)
RUN apk add --no-cache chromium nss freetype harfbuzz ca-certificates ttf-freefont font-noto-cjk

# Set Chromium path for chromedp
ENV CHROME_PATH=/usr/bin/chromium-browser

# Create uploads directory
RUN mkdir -p /app/uploads && chmod 777 /app/uploads

# Download Noto Sans TC Chinese font for PDF report generation
# go-pdf/fpdf requires TrueType (.ttf) fonts — OTF/CFF (magic: OTTO) is NOT supported
RUN mkdir -p /app/assets/fonts

# Copy font files from source (included in repo for reliability)
COPY assets/fonts/NotoSansTC-Regular.ttf /app/assets/fonts/NotoSansTC-Regular.ttf
COPY assets/fonts/NotoSansTC-Bold.ttf /app/assets/fonts/NotoSansTC-Bold.ttf

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy migrations (needed at runtime for DB setup)
COPY --from=builder /app/migrations /app/migrations

# Copy config file
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/server"]
