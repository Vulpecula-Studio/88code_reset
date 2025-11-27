.PHONY: help test run manual build clean docker-build docker-test docker-up docker-down docker-logs

# 默认目标
help:
	@echo "88code FREE 订阅重置工具"
	@echo ""
	@echo "可用命令:"
	@echo "  make test          - 运行测试模式（不执行重置）"
	@echo "  make run           - 启动调度器"
	@echo "  make manual        - 手动触发重置（需要确认）"
	@echo "  make build         - 编译程序"
	@echo "  make clean         - 清理编译文件和数据"
	@echo ""
	@echo "Docker 命令:"
	@echo "  make docker-build  - 构建 Docker 镜像"
	@echo "  make docker-test   - 运行 Docker 测试模式"
	@echo "  make docker-up     - 启动 Docker 调度器"
	@echo "  make docker-down   - 停止 Docker 服务"
	@echo "  make docker-logs   - 查看 Docker 日志"

# 测试模式
test:
	go run cmd/reset/main.go -mode=test

# 运行调度器
run:
	go run cmd/reset/main.go -mode=run

# 手动重置
manual:
	go run cmd/reset/main.go -mode=manual

# 编译
build:
	go build -o reset cmd/reset/main.go
	@echo "编译完成: ./reset"

# 清理
clean:
	rm -f reset
	rm -rf data/*.lock
	@echo "清理完成"

# 清理所有数据（危险）
clean-all: clean
	rm -rf data/*.json logs/*.log
	@echo "已清理所有数据文件"

# Docker 构建
docker-build:
	docker-compose build

# Docker 测试模式
docker-test:
	docker-compose --profile test up reset-test

# Docker 启动调度器
docker-up:
	docker-compose up -d reset-scheduler
	@echo "调度器已在后台启动"
	@echo "使用 'make docker-logs' 查看日志"

# Docker 停止
docker-down:
	docker-compose down

# Docker 日志
docker-logs:
	docker-compose logs -f reset-scheduler

# 查看状态
status:
	@echo "=== 账号信息 ==="
	@cat data/account.json 2>/dev/null || echo "文件不存在"
	@echo ""
	@echo "=== 执行状态 ==="
	@cat data/status.json 2>/dev/null || echo "文件不存在"

# 查看今日日志
logs:
	@tail -50 logs/reset_$$(date +%Y-%m-%d).log 2>/dev/null || echo "日志文件不存在"

# 实时日志
logs-follow:
	@tail -f logs/reset_$$(date +%Y-%m-%d).log
