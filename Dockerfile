# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency files trước để cache go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ mã nguồn
COPY . .

# Build binary
RUN go build -o main .

# --- Run stage ---
FROM alpine:latest

WORKDIR /root/

# Copy binary
COPY --from=builder /app/main .

# Nếu bạn muốn giữ folder uploads trong image (dev), bỏ comment dòng dưới
# COPY --from=builder /app/uploads ./uploads

EXPOSE 8080

CMD ["./main"]