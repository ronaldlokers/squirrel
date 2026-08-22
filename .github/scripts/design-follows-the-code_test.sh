#!/usr/bin/env bash
#
# What the gate does, proved rather than described.
#
# A blocking check is the one kind of CI job whose own bugs are expensive in
# both directions: wrong in one direction it blocks work that was fine, wrong
# in the other it is decoration. So it gets a test, and the test builds real
# git history rather than stubbing `git diff` — the thing most likely to be
# wrong here is the shape of what git returns, and a stub would agree with
# whatever I assumed.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
gate="$here/design-follows-the-code.sh"

fails=0
check() { # check <name> <want-exit> <actual-exit> <output>
  if [ "$2" = "$3" ]; then
    echo "ok   $1"
  else
    echo "FAIL $1: wanted exit $2, got $3"
    printf '       %s\n' "$4"
    fails=$((fails + 1))
  fi
}

# One scratch repository, a branch per case off the same base.
repo="$(mktemp -d)"
trap 'rm -rf "$repo"' EXIT
cd "$repo"
git init -q -b main
git config user.email t@example.com
git config user.name Test
mkdir -p internal/web/static internal/web/templates
echo 'body{}' > internal/web/static/pile.css
echo '<p>' > internal/web/templates/home.html
echo 'the look' > DESIGN.md
echo 'package main' > main.go
git add -A && git commit -qm base
base="$(git rev-parse HEAD)"

on() { git checkout -q -B "$1" "$base"; }
run() { # run <title> -> sets out/code
  set +e
  out="$("$gate" "$base" "$(git rev-parse HEAD)" "${1-}" 2>&1)"
  code=$?
  set -e
}

# The whole point: look changed, record did not.
on css-alone
echo 'body{color:red}' > internal/web/static/pile.css
git commit -qam 'restyle'
run 'fix: restyle'
check "css without DESIGN.md blocks" 1 "$code" "$out"
grep -q 'internal/web/static/pile.css' <<<"$out" ||
  { echo "FAIL css-alone: does not name the file"; fails=$((fails + 1)); }

# A template counts too — DESIGN.md describes markup as much as style.
on template-alone
echo '<p>hello' > internal/web/templates/home.html
git commit -qam 'reword'
run 'fix: reword'
check "template without DESIGN.md blocks" 1 "$code" "$out"

# The ordinary passing case.
on together
echo 'body{color:red}' > internal/web/static/pile.css
echo 'and why' >> DESIGN.md
git commit -qam 'restyle and say why'
run 'fix: restyle and say why'
check "css with DESIGN.md passes" 0 "$code" "$out"

# Nothing that changes the look: the gate must not have an opinion.
on go-only
echo '// more' >> main.go
git commit -qam 'server'
run 'fix: server'
check "code that is not the look passes" 0 "$code" "$out"

# The escape hatch, in the title.
on override-title
echo 'body{color:blue}' > internal/web/static/pile.css
git commit -qam 'restyle'
run 'fix: restyle (no-design-change)'
check "override in the title passes" 0 "$code" "$out"

# ...and in a commit message, which is where the reason can actually be given.
on override-commit
echo 'body{color:blue}' > internal/web/static/pile.css
git commit -qam 'fix: restyle

no-design-change: a selector list, nothing DESIGN.md names.'
run 'fix: restyle'
check "override in a commit message passes" 0 "$code" "$out"

# The override is a marker, not a mood. A title that merely talks about design
# must not skip the gate.
on near-miss
echo 'body{color:blue}' > internal/web/static/pile.css
git commit -qam 'restyle'
run 'fix: no design change here really'
check "a title that only sounds like the override blocks" 1 "$code" "$out"

# Files whose names contain the stylesheet's but are not it — one extending it
# on the right, one on the left. The pattern is anchored at both ends and these
# are what catch either anchor going missing; without them the anchors are
# decoration nothing would notice being removed.
on trailing-neighbour
echo '{}' > internal/web/static/pile.css.map
git add -A && git commit -qm sourcemap
run 'chore: sourcemap'
check "a name the stylesheet's is a prefix of passes" 0 "$code" "$out"

on leading-neighbour
mkdir -p docs/internal/web/static
echo 'body{}' > docs/internal/web/static/pile.css
git add -A && git commit -qm 'a copy in the docs'
run 'docs: a copy'
check "a name the stylesheet's is a suffix of passes" 0 "$code" "$out"

echo
if [ "$fails" -gt 0 ]; then
  echo "$fails failed"
  exit 1
fi
echo "all passed"
