#!/usr/bin/env bash
#
# The image starts, on the architecture it says it is.
#
# A build proves the code compiles for an architecture. It does not prove the
# image runs: a wrong-architecture binary, a missing entrypoint or a base image
# that cannot execute a static binary all build cleanly and fail at `docker
# run`. Production is arm64 and staging is amd64, so both have to be started.
#
# With no POSTGRES_SERVER the binary fails its own configuration check and exits
# 1. That is the success case here — it means the process ran far enough to read
# its own environment. An image that cannot execute exits with something else
# ("exec format error" is 125 or 255), and telling those two apart is the whole
# job.
set -uo pipefail

image=${1:?usage: it-starts.sh <image> [platform]}
platform=${2:-}

run=(docker run --rm)
if [ -n "$platform" ]; then
  run+=(--platform "$platform")
fi

said=$("${run[@]}" "$image" 2>&1)
code=$?

echo "$said"

if [ "$code" -ne 1 ]; then
  echo "==> the image exited $code; expected 1, its own configuration check." >&2
  echo "    An image that cannot execute here fails this way." >&2
  exit 1
fi

case "$said" in
  *"boot failed"*) ;;
  *)
    echo "==> the image ran and exited 1 without reaching its configuration check." >&2
    exit 1
    ;;
esac

echo "==> starts on ${platform:-the runner architecture}"
