# Echo Trace Server Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制源代码
COPY backend/ ./

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o echo_trace_server .

# 运行时镜像
FROM alpine:latest

WORKDIR /app

# 复制二进制文件
COPY --from=builder /app/echo_trace_server .

# 复制配置文件
COPY config/ ./config/

# 暴露端口
EXPOSE 9191

# 运行
CMD ["./echo_trace_server", "-port", "9191"]
