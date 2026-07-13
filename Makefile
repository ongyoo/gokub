.PHONY: build test vet verify fmt install run extension

build:
	mkdir -p dist
	go build -o dist/gokub ./cmd/gokub

test:
	go test ./...

vet:
	go vet ./...

verify:
	test -z "$$(gofmt -l .)"
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	go test -race ./...
	go vet ./...
	go build ./cmd/gokub
	./scripts/test-installer.sh
	./scripts/test-packaging.sh
	./scripts/test-linux-wizard.sh

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './dist/*')

install:
	go install ./cmd/gokub

run:
	go run ./cmd/gokub

extension:
	cd vscode-extension && npm run package
