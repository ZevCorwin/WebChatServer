# --- Build Stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod và go.sum để tải dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ mã nguồn
COPY . .

# Tạo thư mục uploads (fix lỗi Render)
RUN mkdir -p /app/uploads

# Build binary
RUN go build -o main .

# --- Run Stage ---
FROM alpine:latest

WORKDIR /root/

# Copy binary từ builder
COPY --from=builder /app/main .
COPY --from=builder /app/uploads ./uploads

EXPOSE 8080

CMD ["./main"]
