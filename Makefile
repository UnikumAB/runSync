default:        test

test:   golangci-lint
	go test -v -race ./...
	( rm sync.ts; go run ./cmd/runSync/main.go sleep 1 & go run ./cmd/runSync/main.go sleep 1 & wait ) && echo done

fmt:
	gofmt -w .

golangci-lint:
ifeq (, $(shell which golangci-lint))
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.46.1
endif
	golangci-lint run --fix ./...

release-build: golangci-lint test
	go build ./...
	GOOS=darwin go build -o bin/runSync-darwin-x86_64 ./cmd/runSync
	GOOS=linux go build -o bin/runSync-linux-x86_64 ./cmd/runSync

build: golangci-lint test
	go build ./cmd/runSync

mod:
	go mod tidy

all: fmt mod test

.PHONY: imports test fmt mod all default release-build

release:
ifeq (, $(shell which goreleaser))
        curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh
endif
	goreleaser --snapshot --skip-publish --rm-dist

