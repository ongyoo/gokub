.PHONY: build test fmt install run extension

build:
	mkdir -p dist
	go build -o dist/gokub ./cmd/gokub

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './dist/*')

install:
	go install ./cmd/gokub

run:
	go run ./cmd/gokub

extension:
	cd vscode-extension && npm run package
