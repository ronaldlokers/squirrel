# Static binary, so the final stage needs no libc and no runtime. This is the
# whole footprint argument for Go: roughly 20MB of image against 200MB, and
# nothing inside it to patch.
FROM golang:1.26-bookworm AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
# Migrations are embedded via embed.FS, so there is no directory to copy and
# nothing to forget.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /squirrel ./cmd/squirrel

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /squirrel /squirrel
USER nonroot:nonroot
ENTRYPOINT ["/squirrel"]
