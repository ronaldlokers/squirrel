# Comp notes: the home screen (`/`)

Comp: `home-screen.html` (open it from its place in the repo — fonts and the
mark are linked relatively to `internal/web/static/`). Two screens in one
file: home with the newest note peeked, then, past a labelled annotation band,
the same screen with the pile empty.

## What was decided, and why

- **The doors are cards, not buttons and not lid links.** Two identical cells
  of one grid, the same card stock, the same 5px-offset sticker depth, side by
  side at every width — a stacked pair on the phone would read as first and
  second, and their equality is the screen's one statement. Each door carries
  its name in the machine's axis (lowercase, `wght` 800, at the Note role's
  floor — 21px, stepping to 23px on the phone, the same documented pair the
  chore name pins) and a headphone-brown meta sub-line: *WHAT YOU SAID* /
  *WHAT COMES BACK*, the latter verbatim from the chores screen's own head.
  A door is a plain `<a>`; it presses flush like everything else you push.
- **The lid has no cross-link.** The Lid Link exists so the pile and the
  chores can reach each other; on the screen whose whole body is those two
  destinations it would be a third copy of the doors. Mark, wordmark, search
  only. Search posts to the pile, where results already live.
- **The peek is the deck card's exact grammar minus the tray.** Purple
  titlebar, acorn badge, arrival date in the titlebar's own format, his words
  on cream — and nothing to press, because the whole card is a single link to
  the pile, where this same note is the top card with all four actions under
  it. Triage in two places would be two views that can disagree (Principle 4),
  so the home screen can show the card but never act on it.
- **No stack behind the peek.** This is the load-bearing safety decision. The
  behind-cards are the system's one licensed way of saying *that* there is
  more, and home does not get to say it: a glimpse of one card must not double
  as a report on the pile's depth. One card, alone, slightly tilted — a card
  someone left on top of the box, not a widget tile (a dashboard tile is
  square; this deliberately is not).
- **The peek's text takes the Result role** (18px / 18.5 phone, `CASL` 1):
  DESIGN.md defines Result as the note "being scanned rather than decided on",
  which is precisely what a glimpse is. Clamped at four lines — a glimpse of a
  long note is still a glimpse, and the whole card is one tap away. (The
  ellipsis says *that* there is more of the text, never how much.)
- **Its label is "THE TOP CARD"**, in the head style the list screens use.
  Shoebox language: it names which card this is, not how many sit under it.
  Rejected: "newest" (imports ordering/queue), anything with "waiting"
  (imports obligation).
- **The empty state is absence, not a message.** The peek and its label are
  simply gone; both doors are pixel-identical in both states; no "all clear",
  no ghost slot, no changed copy. An empty pile changes nothing about the
  room, which is what "stopping partway is a normal ending" looks like
  structurally rather than as reassuring copy.
- **The foot line is `thoughts go in through the chat`.** The first screen of
  an installed app is where a capture box would try to grow; this states where
  capture lives, as a fact, asking nothing.
- Everything is a plain link or a bare form; there is no JavaScript in the
  comp and none needed by the design.

## The door art (added after the owner supplied illustrations)

- **The pile door wears the owner's illustration as supplied**
  (`assets/door-pile.png`): papers in a violet tray. It was approved as-is and
  it earns it — its purple *is* `--violet` to a rounding error, its cream and
  yellow sit on `--card` and `--kept`, and its heavy dark outline is the
  system's own line. Nothing to fix.
- **The chores direction art was refused, not adjusted**
  (`assets/door-chores-direction.png`). Two faults: grey on the clip breaks
  the No Neutral Rule, and — fatally — two ticked boxes and one empty is a
  progress meter drawn as a picture, 2 of 3 done, on the product whose hardest
  rule is never a count in any form. The green ticks also spend the `done`
  state colour as decoration on a chore at rest, the same class of mistake the
  first chores screen made with orange. No amount of recolouring saves a
  checklist; the *idea* had to change.
- **What replaced it: a broom through a violet return loop**, drawn as inline
  SVG in the comp. The loop is the meaning — the chores screen's own line is
  "a chore is a card that comes back on its own", so the drawing is a thing
  circling back, arrowhead pointing home toward the handle, not a list being
  finished. The broom keeps the direction art's one good instinct (its warmth
  and domesticity) in legal colours: straw in the kept amber the pile art
  already wears, handle in the mascot's own headphone brown (never grey),
  ferrule and loop in `--violet`. Nothing ticked, nothing counted, no grey,
  no `done` green, no orange.
- **Why SVG.** The system is flat fills inside a 3px outline, which is exactly
  what SVG is; `vector-effect="non-scaling-stroke"` holds the drawn line at
  3px on screen at every size, the same trick the brim already uses. Being
  authored in the palette by construction, it cannot drift.
- **Placement: art above the name, left-aligned with the text, in a
  fixed-height slot** (86px desktop, 66px phone). The fixed slot is the
  equality mechanism — both drawings render the same height regardless of
  their aspect, so neither door leads. Both drawings carry the same
  violet + amber + cream chord for the same reason. They tilt ±1.5° in
  opposite directions (mirrored, so still equals): a drawing printed slightly
  askew, because nothing handled sits square. No CSS shadow on the art — it
  is printed on the door's stock, not stuck onto it; its own outline is its
  depth. At 390px the doors are ~177px wide and the 66px art clears the
  padding comfortably; side-by-side survives untouched.
- Both images are decorative — the door's name sits directly beneath — so the
  `img` takes empty alt and the SVG is `aria-hidden`.
- **Should the pile icon become SVG too?** Eventually, yes — one format is
  cleaner and the PNG is the one raster in an otherwise vector system — but
  not by tracing. The illustration's charm is in its drawn imperfections
  (the clip's highlight, the papers' layered edges), and a hand-me-down trace
  would keep the format and lose the drawing. At door size the PNG already
  speaks the outline language indistinguishably from the SVG beside it, so
  the mismatch costs nothing today. If the owner's illustrator can re-cut it
  as vectors, take that; a worse drawing for consistency's sake is a bad
  trade.

## New things the system needed (nothing invented quietly)

- **A Door component.** Card stock as a navigational surface does not exist in
  DESIGN.md — cards are notes/chores, links are the quiet lid link. The door
  is deliberately composed only of existing parts (card fill, result-card
  shadow scale, Note-floor size, meta role, hover-to-paper, the matching 5px
  press), but the *composition* is new and should be documented if built.
- **A link-card hover.** Buttons hover by deepening; the door hovers by
  brightening to paper and the peek by lifting 3px (offset 5px→8px, same cast
  family). New behaviour, flagged rather than smuggled: a card you can enter
  wants a "pick it up" affordance that a button's darkening doesn't give.
- No new colours, radii, type steps or shadow values anywhere. The one
  off-ramp size the first draft had (a 20px door name) was replaced with the
  documented 21/23 pair before delivery.

## DESIGN.md amendments proposed

1. **The Door** as a component: a card-stock link to one of the box's two
   halves — name in the precise axis at the Note floor, meta sub-line, result
   shadow, presses flush; hover brightens to paper. Only ever two of them.
2. **The Peek**: the deck card's grammar minus the tray; one link; never a
   stack behind it; never an action, a state colour, or a badge; absent —
   without residue — when the pile is empty. Worth writing down because every
   future "improvement" to this element (a second card, a "more" line, an
   unread dot) is the inbox coming back.
3. **The Lid Link** entry gains one line: *on the home screen the lid carries
   no cross-link — the doors are the body of the page.*
4. Layout section: the home screen is the one surface with two primary
   destinations; they are equals and must render identically in every state.
5. **Door Art** as a component (new with the icons): one drawing per door,
   above the name, in a fixed-height slot so both render equal; flat fills
   from the documented palette inside the 3px outline
   (`vector-effect="non-scaling-stroke"` when SVG); decorative to assistive
   tech; no CSS shadow — printed on the stock, not stuck to it; a small
   mirrored tilt. And its guard rails, which are the refused direction art
   written as rules: door art may never depict a count, a progress state, a
   tick, or completion; never wears a state colour *as a state* (amber as
   straw is a colour, a green tick is a claim); never grey; never orange.
   Only ever two drawings, because there are only ever two doors.

## The door art, and the exception it carries

Both doors carry the owner's supplied illustration, centred in the art slot:
`assets/door-pile.png` and `assets/door-chores.png`.

The chores drawing was refused once in this comp's history and then chosen
anyway, which makes it a decision rather than an oversight, and it is written
down here so nobody re-litigates it or "fixes" it later:

- It carries **grey** on the clipboard's clip, which The No Neutral Rule
  otherwise forbids.
- It shows **two ticked boxes and one empty**, which is a progress reading
  drawn as a picture, and it uses the `done` green as ornament.

The argument against it was that the never-a-count rule does not care whether
the count is a numeral, and that a checklist imports the metaphor the product
exists to avoid. The argument for it is the owner's, and it is his product: the
drawing is warm, it reads instantly as chores, and it is the one he wants on his
own home screen.

What that costs, precisely: this is the only surface in the system where grey
appears and the only place a completion is depicted. It is illustration rather
than interface — nothing here reports state, and the chores screen itself still
says how often and softly-when, and never how many. If a third door is ever
added, this is not the precedent to follow.

The drawn alternative — a broom passing through a return loop, all palette
colours, no ticks — is preserved in this file's history if the question reopens.

## The peek, and its removal

An earlier version of this comp put the newest note on the home screen —
readable, not actionable, one card, no stack behind it, absent without residue
when the pile was empty. The notes argued at length that a preview can be made
safe if you strip every adornment that turns it into an inbox.

The owner looked at it and did not want it. That is the shorter answer to the
question all those guards were arguing about: a home screen that shows what is
waiting greets you with what is waiting, however carefully it is dressed.

It came out cleanly, which the design had already predicted: the peek's own
conditions said that if it ever needed to change it should be deleted rather
than softened, and the empty-state variant was the standing proof that the
doors alone are a complete home screen. Removing it collapsed the comp from two
states to one — with nothing on the screen that depends on what the pile holds,
a full pile and an empty one look identical.

If a preview ever returns, the seven prohibitions return with it: no count, no
stack, no "more", no action, no state colour, no urgency copy, no residue when
empty.
