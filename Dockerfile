# 多阶段构建
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reset ./cmd/reset

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/reset .

# 创建数据和日志目录
RUN mkdir -p /app/data /app/logs

# 设置时区为北京时间
ENV TZ=Asia/Shanghai

# 暴露数据和日志目录
VOLUME ["/app/data", "/app/logs"]

# 默认运行调度器模式
ENTRYPOINT ["./reset"]
CMD ["-mode=run"]
