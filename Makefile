APP_NAME=download-accelerator
BIN_DIR=bin
CMD=.

.PHONY: help build run clean fmt vet test docker

help:
	@echo "make build     - 构建可执行文件"
	@echo "make run       - 本地运行"
	@echo "make clean     - 清理构建产物"
	@echo "make fmt       - gofmt 格式化"
	@echo "make vet       - go vet 静态检查"
	@echo "make test      - 运行单元测试"
	@echo "make docker    - 构建Docker镜像"

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD)

run:
	go run $(CMD)

clean:
	rm -rf $(BIN_DIR)

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

docker:
	docker build -t $(APP_NAME):local .

