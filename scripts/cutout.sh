#!/bin/sh
set -eu

usage() {
	cat <<'USAGE'
cutout — turn a generated face into a mood asset.

    scripts/cutout.sh <render> <out.png>
    scripts/cutout.sh --check <file.png>

Gemini returns a JPEG with a checkerboard painted into it rather than a PNG
with an alpha channel: a picture of transparency. Dropped on the board it
would sit on purple with a grey chessboard around it.

Two things have to be true for a sixth face to look like it belongs, and both
were measured off the five rather than guessed.

The canvas is 260x209 and the templates say so in their width and height
attributes, so the size is a contract rather than a preference.

The head sits on the same spot on all five: its bottom-left corner within a
pixel or two of (58, 205), and its bottom edge 115 to 123 pixels wide. A new
one is aimed at 121 and accepted within 3, because the five vary by more than
that between themselves. That is
what makes them read as one set rather than five drawings when the mood
changes, so a sixth is placed to match rather than centred.

The width is measured on the finished canvas and the scale corrected until it
lands, rather than computed once from the render. A rounded corner means the
bottom three rows of a 900px drawing and the bottom three rows of a 209px one
are different fractions of the same curve, so one calculation overshoots by
about 8%. Measuring where the number actually has to be true costs three more
passes of a second each.

Both numbers are taken off the bottom of the largest shape because that is the
only part that is reliably the head. Furniture — a scribble, a bolt, a
raincloud — attaches above and to the right and never below or left, and it
often touches, so head and furniture arrive as one shape. Its bottom-left
corner is still the head's, and only the head reaches the bottom edge.

The background goes in two steps rather than one. Flooding the checkerboard
away directly does not work: JPEG leaves every flat grey speckled, so a flood
tight enough to spare the face stops at the compression noise, and one loose
enough to cross it reaches the head, which is nearer the light squares than
they are to the dark ones. So the picture is first reduced to light-and-dark,
which makes the whole checkerboard one colour and the head another, and only
then flooded in from the four corners. What that flood reaches becomes the
transparent part. The eyes are light too and survive, because the black
outline is a wall the flood cannot cross.

--check prints the same measurements for a file that already exists, which is
how a drawing that sits wrong is seen sitting wrong.

One thing it cannot do: a light gap enclosed by the drawing stays opaque,
because the flood comes from outside and the outline stops it. That is what
keeps the eyes, and it is also what fills the holes in a scribble. A drawing
whose background shows through a loop needs that loop opened, or the gap
painted out by hand afterwards.

Needs ImageMagick 7 on the path.
USAGE
}

die() {
	echo "cutout: $1" >&2
	exit 1
}

command -v magick >/dev/null 2>&1 || die "no magick on the path (ImageMagick 7)"

WIDTH=260
HEIGHT=209
FOOT=121
FOOT_SLACK=3
HEAD_LEFT=58
HEAD_BOTTOM=205
LIGHT=70

biggest() {
	magick "$1" -alpha extract -threshold 10% \
		-define connected-components:verbose=true \
		-define connected-components:area-threshold=200 \
		-connected-components 8 null: 2>/dev/null |
		awk 'NR>1 && $NF=="srgb(255,255,255)" {
			a = $4 + 0
			if (a > best) { best = a; split($2, b, /[x+]/) }
		} END {
			if (best == 0) exit 1
			print b[1], b[2], b[3], b[4]
		}'
}

measure() {
	f="$1"
	fields="$(biggest "$f")" || die "$f has no shape in it"
	# shellcheck disable=SC2086
	set -- $fields
	printf '%-22s head %3dx%-3d left %3d bottom %3d\n' \
		"$(basename "$f")" "$1" "$2" "$3" "$(($2 + $4))"
}

alongside() {
	for f in "$(dirname "$0")"/../internal/web/static/mood-*.png; do
		[ -f "$f" ] && measure "$f"
	done
}

report() {
	out="$1"
	got="$(magick "$out" -format '%wx%h' info:)"
	[ "$got" = "${WIDTH}x${HEIGHT}" ] ||
		die "that is ${got}, and the templates say ${WIDTH}x${HEIGHT}"
	[ "$(magick "$out" -format '%A' info:)" != "Undefined" ] ||
		die "that has no alpha channel, so its background is still painted on"
	corners="$(magick "$out" -alpha extract -format \
		'%[fx:max(max(p{0,0},p{w-1,0}),max(p{0,h-1},p{w-1,h-1}))]' info:)"
	awk -v a="$corners" 'BEGIN { exit !(a < 0.02) }' ||
		die "the corners are opaque, so the background is still painted on"
	echo "the new one, and the five it joins:"
	measure "$out"
	alongside
}

case "${1:-}" in
"" | -h | --help)
	usage
	exit 0
	;;
--check)
	[ $# -eq 2 ] || die "usage: scripts/cutout.sh --check <file.png>"
	[ -f "$2" ] || die "no such file: $2"
	report "$2"
	exit 0
	;;
esac

[ $# -eq 2 ] || die "usage: scripts/cutout.sh <render> <out.png>"
render="$1"
out="$2"
[ -f "$render" ] || die "no such file: $render"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

read -r w h <<EOF
$(magick "$render" -format '%w %h' info:)
EOF

magick "$render" -colorspace gray -threshold "${LIGHT}%" -type truecolor \
	-fill red -fuzz 0% \
	-floodfill +0+0 white \
	-floodfill "+$((w - 1))+0" white \
	-floodfill "+0+$((h - 1))" white \
	-floodfill "+$((w - 1))+$((h - 1))" white \
	-fill white +opaque red -fill black -opaque red "$tmp/mask.png"

magick "$render" -alpha off "$tmp/mask.png" \
	-compose copy_opacity -composite "$tmp/cut.png"

fields="$(biggest "$tmp/cut.png")" || die "nothing survived the cut — is $render a face?"
# shellcheck disable=SC2086
set -- $fields
bw=$1
bh=$2
bx=$3
by=$4
[ "$bw" -lt "$w" ] || [ "$bh" -lt "$h" ] ||
	die "nothing was cut away — is $render already a cutout?"

footwidth() {
	magick "$1" -crop "${WIDTH}x3+0+$((HEAD_BOTTOM - 3))" +repage \
		-alpha extract -threshold 10% -trim +repage \
		-format '%w' info: 2>/dev/null || echo 0
}

place() {
	magick "$tmp/cut.png" -resize "$1%" "$tmp/scaled.png"
	fields="$(biggest "$tmp/scaled.png")" || die "the shape vanished when it was scaled"
	# shellcheck disable=SC2086
	set -- $fields
	magick "$tmp/scaled.png" -background none -gravity northwest \
		-extent "${WIDTH}x${HEIGHT}+$(($3 - HEAD_LEFT))+$(($4 + $2 - HEAD_BOTTOM))" \
		-depth 16 "$out"
}

seen="$(magick "$tmp/cut.png" -crop "${bw}x3+${bx}+$((by + bh - 3))" +repage \
	-trim +repage -format '%w' info: 2>/dev/null || echo 0)"
[ "$seen" -gt 0 ] || die "the shape does not reach its own bottom edge"

scale="$(awk -v want="$FOOT" -v seen="$seen" 'BEGIN { printf "%.4f", want / seen * 100 }')"
place "$scale"

tries=0
while [ "$tries" -lt 4 ]; do
	got="$(footwidth "$out")"
	[ "$got" -gt 0 ] || die "the placed shape does not reach the baseline"
	off=$((got - FOOT))
	[ "$off" -lt 0 ] && off=$((-off))
	[ "$off" -le "$FOOT_SLACK" ] && break
	scale="$(awk -v s="$scale" -v want="$FOOT" -v got="$got" \
		'BEGIN { printf "%.4f", s * want / got }')"
	place "$scale"
	tries=$((tries + 1))
done

report "$out"
