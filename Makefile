.PHONY: check fmt vet test test-integration test-browser build dev

check: fmt vet test

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run: gofmt -w ." && exit 1)

vet:
	go vet ./...

# -race: TestConversationsSurviveConcurrentUse has no assertion of its own —
# the detector is the assertion. Without this it runs, passes, and proves
# nothing, which is what it did while its own comment said CI ran it this way.
test:
	go test -race ./...

test-integration:
	# -p 1: internal/boot and internal/squirrel both truncate the same live
	# Postgres in their withStore setup, so their package test binaries must
	# not run concurrently.
	go test -race -tags=integration -p 1 ./...

# The parts of this product that only exist in a browser: the stamp, the
# interval question, the keys, and search that answers as you type. Needs a
# browser on the machine — set BROWSER to point at one, or let it find Chrome
# or Chromium on the path.
test-browser:
	go test -tags=browser -count=1 ./internal/web/

# The screen on a port, with invented contents and no database. Templates and
# static files are read from the working tree, so an edit is a refresh rather
# than a rebuild — which is what impeccable's live mode, the design detector's
# overlay and any by-hand test of the service worker all need.
#
# Behind the `dev` tag, and so is the code that switches internal/web onto the
# working tree: a binary built without it cannot turn any of this on.
dev:
	go run -tags=dev ./cmd/devscreen

build:
	CGO_ENABLED=0 go build -o squirrel ./cmd/squirrel
