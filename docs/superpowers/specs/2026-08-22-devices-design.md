# Devices: phone primary, desktop first-class

**Status:** draft, 2026-08-22. Written for review, not approved.

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
4. **The lid fits, with room for one more control.** It does not today: four
   44px icons plus the wordmark need 392px on a 390px screen, which is what
   stopped the lid-capture work on 22 August. Either the wordmark goes on
   phones or the lid stops being where new controls land — and that decision
   is the first thing this spec needs and does not have.
5. **The known-open phone items are closed or re-affirmed in writing**: the
   wrapping views nav, iOS's uncancellable search field, and the zoom
   trade-off. The third is already written down as accepted; the other two are
   not.

## What this does not do

- **It does not add a mobile-only surface.** Two views, one pile. A screen that
  exists only on a phone is a third place a thought can be.
- **It does not touch the zoom decision.** That is recorded, argued and
  accepted, with its cost stated. Revisiting it is a separate question.
- **It does not make the desktop a second-class citizen.** The keyboard path is
  first-class *by rule*, and the desktop is where it lives.

## The open question this spec cannot answer

**What gives, so the lid can hold a fourth control?** Measured at 390px:

```
brand      14 → 152   (138px)
tobuddy   224 → 268
findbox   278 → 322
tellbox   332 → 376
where      14 →  58   ← wrapped
```

28px padding + 138px brand + five 10px gaps + four 44px icons = 392px.

The options are dropping the wordmark on phones, putting new controls behind
the menu instead of beside it, shaving the gaps to zero slack, or accepting
that the lid is closed. Each is a design decision with a different cost, and
the answer determines whether capture-from-the-lid (#99) is buildable at all.

**This spec is not ready for approval until that is decided.**

## Testing

The appearance snapshot added on 22 August records computed shape at 1280px
only. Extending it to 390px is most of the mechanism for criteria 1–3: the same
recording, a second viewport, and assertions rather than a diff for the ones
that are rules rather than records.
