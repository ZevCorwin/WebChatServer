# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency files trước để cache go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ mã nguồn
COPY . .

# Tạo thư mục uploads để tránh lỗi checksum
RUN mkdir -p /app/uploads

# Build binary
RUN go build -o main .

# --- Run stage ---
FROM alpine:latest

WORKDIR /root/

# Copy binary và thư mục cần thiết
COPY --from=builder /app/main .
COPY --from=builder /app/uploads ./uploads

# Expose port (trùng với APP_PORT trong .env)
EXPOSE 8080

# Lệnh chạy app
CMD ["./main"]
