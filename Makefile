.PHONY: all fmt fmt-check vet test build laws verify verify-json demo clean cross

VERSION ?= v1.0.0

all: fmt-check vet test build laws verify

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "以下文件未格式化:"; gofmt -l .; exit 1)

vet:
	go vet ./...

test:
	go test ./... -count=1

build:
	CGO_ENABLED=0 go build -trimpath -o dist/confluxlab ./cmd/confluxlab

# 代数律门禁：三条定律任一失败即返回非零
laws:
	go run ./cmd/confluxlab laws -rounds 300

# 收敛性门禁：随机并发调度 + 分区 + 重复投递
verify:
	go run ./cmd/confluxlab verify -rounds 500 -replicas 5

verify-json:
	go run ./cmd/confluxlab verify -rounds 200 -json

demo:
	go run ./cmd/confluxlab demo

# 全平台静态二进制（Linux / macOS / Windows）
cross:
	VERSION=$(VERSION) bash scripts/build.sh

clean:
	rm -rf dist coverage.out
