# =========================
# Stage 1: Build
# =========================
FROM golang:1.22-alpine AS builder

# Cài git (nếu module cần)
RUN apk add --no-cache git

# Tạo thư mục làm việc
WORKDIR /app

# Copy file go.mod và go.sum trước để cache dependency
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code
COPY . .

# Build binary (chạy go build)
RUN go build -o server main.go

# =========================
# Stage 2: Run
# =========================
FROM alpine:3.20

# Cài chứng chỉ HTTPS (nếu gọi API HTTPS, ví dụ SMTP, Google API)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary từ stage build
COPY --from=builder /app/server .

# Copy thư mục uploads để container có thể lưu file tạm
COPY uploads ./uploads

# Expose port backend
EXPOSE 8080

# Dòng lệnh khởi chạy server
CMD ["./main"]
