APP := micro-health-checker
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/christiandente/micro-health-checker/internal/version.Version=$(VERSION) -X github.com/christiandente/micro-health-checker/internal/version.Commit=$(COMMIT) -X github.com/christiandente/micro-health-checker/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: all build test test-race docs-check fmt vet tidy run docker-build clean

all: test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="$(LDFLAGS)" -o bin/$(APP) ./cmd/micro-health-checker

test:
	go test ./...

test-race:
	go test -race ./...

docs-check:
	python3 scripts/check_docs.py

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

tidy:
	go mod tidy

run:
	MHC_CONFIG=./configs/config.example.yml go run ./cmd/micro-health-checker

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t $(APP):$(VERSION) .

clean:
	rm -rf bin coverage.out
