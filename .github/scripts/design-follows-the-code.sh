#!/usr/bin/env bash
#
# DESIGN.md follows the code.
#
# Twice an outside review has had to hand-audit the design system against the
# shipped markup to find contradictions: seven of them on 20 August 2026, and
# the zoom rule overstating what survives on 22 August. The appearance snapshot
# catches the code side of that drift automatically. Nothing caught the
# document side, because nothing can — it is a claim about what the design
# *means*, and only a person can say whether it still holds.
#
# So this blocks. A change to the stylesheet or the templates does not merge
# unless DESIGN.md moved with it.
#
# It will sometimes be wrong, and that is faced rather than hidden: plenty of
# CSS changes touch nothing DESIGN.md documents. For those, put
# `no-design-change` in the pull request title or in any commit message on the
# branch, and this stands down and says so, making the skip visible rather than
# silent.
#
# If that override starts appearing on most pull requests, this is measuring
# the wrong thing and should be changed rather than habitually overridden.
#
# Usage: design-follows-the-code.sh <base-sha> <head-sha> [pr-title]
set -euo pipefail

base="${1:?base sha}"
head="${2:?head sha}"
title="${3:-}"

# What counts as changing the look. The stylesheet and the templates: the two
# places DESIGN.md is a description of.
looks='^internal/web/(static/pile\.css|templates/.*\.html)$'

changed="$(git diff --name-only "$base" "$head")"

if ! grep -qE "$looks" <<<"$changed"; then
  echo "Nothing here touches the look of the product."
  exit 0
fi

# Looked for in the commits as well as the title, so the reason can be recorded
# where the reasoning is.
if grep -qF 'no-design-change' <<<"$title" ||
   git log --format=%B "$base..$head" | grep -qF 'no-design-change'; then
  echo "Skipped: this change says it documents nothing in DESIGN.md."
  echo "If that is not true, remove the marker and describe the change there."
  exit 0
fi

if grep -qxF 'DESIGN.md' <<<"$changed"; then
  echo "DESIGN.md moved with the code."
  exit 0
fi

echo "::error::This changes how the product looks, and DESIGN.md did not move."
echo
echo "Files that changed the look:"
grep -E "$looks" <<<"$changed" | sed 's/^/  /'
echo
echo "DESIGN.md is the record of what this product looks like and why. It has"
echo "drifted from the code twice, and both times an outside review had to find"
echo "it by reading the markup line by line."
echo
echo "Either describe this change in DESIGN.md, or — if it genuinely documents"
echo "nothing there — put 'no-design-change' in the pull request title or a"
echo "commit message, so the decision is on the record rather than in nobody's."
exit 1
