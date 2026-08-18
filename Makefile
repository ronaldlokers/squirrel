.PHONY: check fmt vet test test-integration test-browser build

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

# The parts of this product that only exist in a browser: the stamp, the
# interval question, the keys, and search that answers as you type. Needs a
# browser on the machine — set BROWSER to point at one, or let it find Chrome
# or Chromium on the path.
test-browser:
	go test -tags=browser -count=1 ./internal/web/

build:
	CGO_ENABLED=0 go build -o squirrel ./cmd/squirrel
