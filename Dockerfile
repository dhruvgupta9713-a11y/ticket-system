# --- Stage 1: Build React Frontend ---
FROM node:alpine AS frontend-builder

WORKDIR /web
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Build Go Backend ---
FROM golang:alpine AS backend-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/ticket-server ./cmd/server

# --- Stage 3: Final Runtime ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
RUN mkdir -p /data
WORKDIR /app

# Copy compiled backend binary
COPY --from=backend-builder /app/ticket-server /app/ticket-server

# Copy compiled frontend assets to the directory expected by the Go binary
COPY --from=frontend-builder /web/dist /app/frontend/dist

# Expose port
EXPOSE 8080

# Environment variables
ENV PORT=8080
ENV JWT_SECRET=default_super_secret_jwt_key
ENV DB_PATH=/data/tickets.db

# Start unified service
CMD ["/app/ticket-server"]
