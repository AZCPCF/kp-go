.PHONY: check fmt vet test build

check: fmt vet test

fmt:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

vet:
	go vet ./...

test:
	go test -race ./...

build:
	go build -o kp ./cmd/kp
