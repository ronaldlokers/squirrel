.PHONY: check fmt vet test test-integration build

check: fmt vet test

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run: gofmt -w ." && exit 1)

vet:
	go vet ./...

test:
	go test ./...

test-integration:
	# -p 1: internal/boot and internal/squirrel both truncate the same live
	# Postgres in their withStore setup, so their package test binaries must
	# not run concurrently.
	go test -tags=integration -p 1 ./...

build:
	CGO_ENABLED=0 go build -o squirrel ./cmd/squirrel
