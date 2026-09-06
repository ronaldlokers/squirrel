# Static binary, so the final stage needs no libc and no runtime. This is the
# whole footprint argument for Go: roughly 20MB of image against 200MB, and
# nothing inside it to patch.
#
# --platform=$BUILDPLATFORM keeps the compiler native and cross-compiles to
# TARGETARCH. Without it buildx runs the Go toolchain itself under QEMU to
# produce an arm64 binary, which is minutes rather than seconds — expensive
# enough that arm64 was only ever built on a tag, which is why an arm64 break
# was a thing you found out about during a release.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
# Migrations are embedded via embed.FS, so there is no directory to copy and
# nothing to forget.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /squirrel ./cmd/squirrel

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /squirrel /squirrel
USER nonroot:nonroot
ENTRYPOINT ["/squirrel"]
