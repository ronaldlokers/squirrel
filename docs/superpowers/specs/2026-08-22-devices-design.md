# Devices: phone primary, desktop first-class

**Status:** settled, 2026-08-22. The one question this spec could not answer
has been answered — see *The lid is closed* — and the acceptance criteria below
are the definition of done for the roadmap's Devices item.

The roadmap has carried one line for this since 20 August — *"Phone primary and
better; desktop first-class"* — and nothing else. It is the only
decided-not-built product item, and in the meantime phone defects have been
accumulating one at a time: a views nav that wraps and strands the tab you are
on, an iOS search field that cannot be cleared without a keyboard, an entire
branch about zooming, a chore's destructive button eight pixels under its
primary one.

Fixing those one at a time is not the same as making the phone primary, and
without a definition of done the second will be declared finished by accretion
of the first.

## What "primary" means, and what it does not

**Primary is about which surface the product is designed against, not which one
gets features.** The desktop keeps everything. Nothing here is a proposal to
make the desktop worse, or to hide anything on it.

What changes is the order of the questions. Today a screen is designed and then
checked on a phone. Primary means the phone is where a screen is decided, and
the desktop is where it is then allowed more room.

Three concrete consequences:

1. **A layout is right when it is right at 390px.** Not "acceptable at 390px
   once it is right at 1280".
2. **A control is sized for a thumb first.** 44px is a floor rather than a
   target, and the reach zone matters as much as the size.
3. **A regression on a phone is a bug of the same severity as one on the
   desktop.** Today they are found later and fixed smaller.

## The precondition, now settled

The screen is LAN-only via a Traefik ipAllowList, and one of the two confirmed
usage scenes is a phone in a queue. Those were mutually impossible until 22
August, when the missing fact was written into `PRODUCT.md`: **the phone is on
an always-on VPN into the homelab.**

That is load-bearing for everything below. If it ever stops being true, this
spec stops being buildable and the answer is an architecture decision rather
than a paragraph.

## The reach zone

A 390×844 screen held one-handed has three bands, and they are not equal:

| Band | Roughly | What belongs there |
| --- | --- | --- |
| **Thumb** | bottom 45% | The answers. What you press many times a session. |
| **Stretch** | middle 35% | What you read, and what you press once. |
| **Reach** | top 20% | Where you are, and the ways out. Pressed deliberately. |

The lid is in the reach band and that is correct: it is what you look past.
The card's answers are in the thumb band and that is correct too.

**What this rules out**, and these are the acceptance criteria:

- No destructive control directly above or below a primary one in the thumb
  band. *(The chores screen had `STOP ASKING` eight pixels under `DID IT`;
  widened to twenty on 22 August, which is a mitigation rather than a fix.)*
- No control that ends a session — a way out, a stop — inside the thumb band.
- The rarest answer on a surface is never its largest object. *(`MAKE A CHORE`
  spanned the card's full width; fixed 22 August.)*

## What "done" looks like

The spec is met when all of these are true and have tests:

1. **Every interactive element is at least 44×44**, including ones added
   later. The current rule lists its selectors by hand, so a new control opts
   out of the floor by being forgotten — a lid button added on 22 August
   rendered at 37px for exactly that reason. This wants to be a test that walks
   the rendered page rather than a rule that has to be remembered.
2. **No horizontal scroll at 320px**, which is the narrowest phone still in
   use. Not tested today at all.
3. **Every screen's primary action is reachable in the thumb band** without
   scrolling, from a cold open.
4. **The lid holds three icons and does not grow.** Not "fits with room for one
   more": three is the number, and a fourth is a design change rather than an
   addition. See *The lid is closed* below.
5. **The known-open phone items are closed or re-affirmed in writing**: the
   wrapping views nav, iOS's uncancellable search field, and the zoom
   trade-off. The third is written down as accepted; the other two are not.

## What this does not do

- **It does not add a mobile-only surface.** Two views, one pile. A screen that
  exists only on a phone is a third place a thought can be.
- **It does not touch the zoom decision.** That is recorded, argued and
  accepted, with its cost stated. Revisiting it is a separate question.
- **It does not add a fourth lid control.** That is now a decision rather than
  an open question, and reversing it is a redesign of the lid.
- **It does not make the desktop a second-class citizen.** The keyboard path is
  first-class *by rule*, and the desktop is where it lives.

## The lid is closed

**Three icons, and the wordmark stays.** A fourth was tried on 22 August and
reverted, and the measurement is why. At 390px:

```
brand      14 → 152   (138px)
tobuddy   224 → 268
findbox   278 → 322
tellbox   332 → 376
where      14 →  58   ← wrapped to a second row
```

28px padding + 138px brand + five 10px gaps + four 44px icons = 392px on a
390px screen. Two pixels, and the menu fell to a second row — which does not
merely look wrong, it doubles the height of the one band the reach zone above
gives the least room to.

The four options were dropping the wordmark on phones, moving new controls
behind the menu, shaving the gaps, or stopping at three. **Stopping at three.**

The reasoning, so it does not have to be had again:

- **The wordmark is not decoration.** The shoebox has one opening and the acorn
  is it — pressing it is how you get home from all thirteen screens. Removing it
  on phones removes the way back on the surface this spec calls primary.
- **Shaving the gaps buys two pixels and spends the slack.** It would fit at
  390px and wrap at 375px, which is a phone people still hold.
- **Behind the menu is not the lid.** A control one press deeper is a different
  control. If something belongs there, it belongs in the menu on its own terms
  rather than as an overflow.

**What this decides.** Capture-from-the-lid (#99) is not buildable and is
closed. The lid grants search its ambient treatment and cannot grant capture
the same, and the honest answer is that capture stays on home rather than that
it gets a worse version of the same idea somewhere tighter. The four-step route
from `/pile` to the slot is a real cost and it is now a known one.

**When this reopens.** If the lid ever needs a fourth control, the question is
not where to squeeze it — it is whether the lid is still the right shape, which
is a redesign and gets its own spec.

## Testing

The appearance snapshot added on 22 August records computed shape at 1280px
only. Extending it to 390px is most of the mechanism for criteria 1–3: the same
recording, a second viewport, and assertions rather than a diff for the ones
that are rules rather than records.
