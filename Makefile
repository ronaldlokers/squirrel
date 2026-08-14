.PHONY: check fmt vet test test-integration build

check: fmt vet test

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run: gofmt -w ." && exit 1)

vet:
	go vet ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

build:
	CGO_ENABLED=0 go build -o squirrel ./cmd/squirrel
