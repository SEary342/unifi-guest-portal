# Stage 1: Build the app (Go and Vite)
FROM golang:1.23-bookworm AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy Go source code and other files to the container
COPY backend/go.mod backend/go.sum ./
COPY backend/cmd ./cmd
RUN go mod download

# Build the Go application
RUN go build -o app ./cmd/main.go

# Stage 2: Serve the Vite-built files
FROM node:22-bookworm AS vite-builder

# Set the working directory inside the container
WORKDIR /vite

# Copy package.json and install dependencies
COPY frontend/ ./
RUN npm ci
RUN npm run build

# Stage 3: Final image to serve the app
FROM fedora:41

# Install required packages for the server
RUN dnf install -y \
    ca-certificates \
    && dnf clean all

# Set the working directory
WORKDIR /app

# Environment variables
ENV UNIFI_USERNAME= \
    UNIFI_PASSWORD= \
    UNIFI_URL= \
    UNIFI_SITE= \
    UNIFI_DURATION=480 \
    DISABLE_TLS=false \
    VITE_PAGE_TITLE="Guest Wi-Fi Portal" \
    PORT=3030 \
    DB_PATH="/data/db"

# Copy SSL certificate
COPY certificate.pem /etc/ssl/certs/

# Copy built Go application and Vite distribution files
COPY --from=builder /app/app ./
COPY --from=vite-builder /vite/dist ./

# Expose the application's port
EXPOSE $PORT

# Run the application
CMD ["./app"]
