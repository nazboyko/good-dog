# Frontend notes

Rules for every screen the player sees. The tokens and classes live in web/src/index.css under "the game". New screens use these, they do not invent their own.

## Type

One family for everything the player reads: Georgia, the serif. The mono is for labels only: the small heading over the run, the narrator's "you meant" and "she heard", the source line under a quote, the About panel's field names. Reading size is `--text`, which scales gently between 17px and 20px with the viewport and never goes below 17px. Line height 1.55 for scene text.

## The button pair

Two buttons, one job each. Both are the same serif as the text, both at least 44px tall, both padded to a shape you can tap with a thumb.

- **primary**, class `btn btn-primary`: the one loud thing on a screen. Light ground, dark text, small caps, a little letter spacing. At most one per screen. On the reveal it is MEET VENUS and nothing else on that screen competes with it. Use it for the action the screen exists for.
- **quiet**, the plain `button` or class `btn-quiet`: dark ground, thin line, warm text. Everything else: get up, follow it, the vocalization panel, back.
- **link**, class `btn-link`: no chrome, mono, underlined, still 44px tall for the tap. For the one quiet link that sits under a primary, like About Venus, from the listing.

Never mix fonts on a button, never let a primary go lowercase, never put two primaries on one screen.

## The reading frame

Anything quoted from a listing at length goes in `.quoted`: a measure around 65ch, line height 1.7, a paragraph gap of one line, a left rule and a faint ground so it reads as the shelter's words and not the game's, and a `.quoted-source` line under it naming the org, city and state. The sanitizer keeps each block of the listing on its own line, so paragraphs come from splitting the description on newlines and every `<p>` inside the frame is one real paragraph as the shelter wrote it.

## Motion

Reserved space plus fade, never insert and push. Everything that appears or disappears is in the DOM from the first frame with its space held, and appearing only changes opacity: `.fade` at 520ms for quiet moments, `.fade-quick` at 200ms for interactive feedback, the reveal photo at 1400ms with its blur to sharp. Reduced motion sets every transition to none and touches no layout property, so it never reintroduces a jump. Photos reserve their box from real dimensions before they load. Details in the dog-perception skill under Presentation.

## Screens breathe

Every screen change is a transition, never a swap: the current screen fades out (360ms), a beat of empty dark (240ms), then the next screen fades in (520ms). Leaving is quicker than arriving. The fetch for the next view runs under the fade out, so the network hides inside the motion. The whole screen rides one opacity layer, `.run-screen`, driven by the run shell. The button that was pressed disables the moment it is clicked, so a double click cannot skip a beat. Reduced motion makes the swap instant but the disable still applies. A signal during the visitor beat is not a screen change: the panel and the narrator crossfade inside that screen. The timings live in `TRANSITION` in web/src/engine/run.ts and are unit tested against these ranges.

## Composition

Run screens (wake, scent, visitor, night, the reveal) are a centered composition, vertically centered in the viewport, and each one fits the viewport with no phantom scroll. The shell is border box so its padding lives inside the viewport height. The listing panel is the exception: it is a scrollable document, it opens at its top with the heading in view, and it scrolls as one page (never as a scroll box inside the shell, which paints as black past the fold). It calls `useViewTop()` on mount. Reserved space is a reveal staging rule and stays inside the reveal: the panel is its own view and the reveal is unmounted underneath it.

## The reveal fits the fold

The reveal is sized to fit one viewport by design: the photo takes what is left after the lines, the button and the shell padding (measured at 488px) and never more than half the screen, `min(50vh, 100vh - 500px)`, with a tighter cap on phones (42vh) and a tighter line stack under 760px tall. Verified centered and inside the fold at 390x844, 584x867, 768x1024 and 1280x720. If a line would still land below the fold, the view eases down to it over 520ms in step with the fade, and any manual scroll cancels the follow for the rest of the reveal.

## Reserved slots grow, they never hide

Where two things swap in one reserved space (the vocalization panel and the narrator), stack them in a single grid cell rather than absolutely positioning both. Absolute children contribute no height, so longer copy overflows silently and no test catches it. In one grid cell the slot is at least its `min-height`, the reserved floor, and grows if the copy needs more. The floor is measured against the tallest real case plus headroom for one wrapped line: 420px desktop, 515px under 480px wide, checked at 320, 390, 768 and 1280. The slot holds every settled exchange as well as the newest one, so it grew when the visit became a conversation.

Measure the floor by enumerating every state the screen can actually reach, never by pasting the longest line of each kind together. The naive method overstated the visitor close by 50px, 422 against a real 369 at 320 wide, and 50px of reserved space nobody can ever use is a hole in the composition on a phone.

The two numbers differ because the lines are not independent. A visitor close is five of them: meant and heard always come from the same signal, and the body and the parting both come from the same band. Pairing the longest meant with the longest heard describes a screen no player will ever see. So loop over the real combinations, archetype by signal by band by shape, write each full set of lines into the live slot, read `getBoundingClientRect` after each, and keep the tallest. It is a few hundred iterations and it runs in under a second.

When the reserved block is much taller than the panel that shares it, the panel is centered in the cell (`align-self: center`) so the space the close will need reads as breathing room rather than a hole. The acceptance test is not the height, it is that the lines above the slot do not move: sample the first arrival line's `top` before the click, all through the crossfade, and after.

Re-measure whenever the copy changes, in both directions, and whenever the composition changes at all. Rewriting the close shortened the 320px worst case from 399 to 369, which retired a whole extra media query. Showing the exchanges already past took it to 481, which put it back. A floor nobody rechecks only ever grows.

## A repeated beat needs a changing composition

Four exchanges of a visitor scene read as one frame reprinted, because the scene-setting lines were the largest thing on screen and stayed at full weight while the only thing that changed was the smallest. The fix was composition, not flow. Three rules came out of it and they apply to any beat that repeats.

The scene-setting recedes once it has done its job. The arrival is the event on the first exchange, full size, full colour, both lines. From the second on it is context: one line, `--text-small`, `--ink-mute`. It never comes back.

What has already happened stays on screen a step back. Each earlier exchange keeps one line dimmed behind a faint rail, so the player reads a visit accumulating rather than a screen redrawn, and the newest exchange is always the loudest thing present. The column is also the shape of the visit, which is what makes the arc line at the close land: the player can see the shape it names.

Stacking copy makes it checkable, and copy that was true alone can become false in a column. The body lines lean on "still", "again" and "has not", which is right for the loud line, where the narrator above names one signal and the body has to speak for the whole visit. Put the same wording in a column and the player can read it against the line above: "she still has her eyes on you" directly under "she glances down the row" is simply a lie. A past exchange is a moment that happened, so the column uses the reaction wording and the loud line keeps the settled wording. Two readings of the same band, and which one is true depends on what is next to it.

Runs collapse. When the visitor does not move between two answers the row repeats word for word, and four identical sentences is a render bug to anybody reading it whatever it is to the engine. One row per change. A visit where nothing moved is then one row, which is the truth said once.

The way forward says where you are through wording. "and then", "the visit goes on", "a moment more", "the day goes on". Never a count, never a fraction, never the same twice in one visit.

Watch what the receding is keyed on. Collapsing the arrival when the exchange history became non empty meant it sat at full height while the panel was up and collapsed the instant an answer landed, shifting the centered composition 26px on every single exchange. Key it on the visit being under way rather than on the answer arriving, so the collapse happens once and the crossfade moves nothing. The check is the same one as always: sample the first arrival line's `top` before the click, through the crossfade, and after.

## The control that ends a beat comes last

Where a screen stages its lines, the button that carries the player forward waits for the last one and is disabled until it arrives. A button rendered at delay zero next to lines at 800ms and 1400ms is live before the player has read them, so a fast hand skips the ending. Fading it in is not enough on its own: an element at `opacity: 0` under `animation-fill-mode: both` is still clickable, so the wait has to disable it too. Delays live next to the transition timings in `web/src/engine/run.ts` and go to zero under reduced motion, where the disable still applies.

## The scene sits behind everything, and that is a paint order problem

The room is a fixed, full viewport canvas at `z-index: 0` under a flat veil, mounted by `SceneBackdrop`. A fixed positioned element with `z-index: 0` paints in the positioned layer, and plain non positioned block content paints *below* that layer, so adding the scene put the whole run screen under the veil. It did not look like that, which is what made it slow to see: the lines that kept rendering were the ones running the `settle` animation, because an animating opacity gives an element its own stacking context and lifts it back above. So the body line was fine and the mismatch narrator beside it, which has no animation, silently vanished behind the veil along with its rail.

The fix is that `.run` and `.run-screen` both carry `position: relative; z-index: 1`. Any new full bleed layer needs the same treatment, and the check is to look at a screen whose text is *not* animating.

No test catches this. It is invisible to the DOM: the element is in the tree, `getComputedStyle` reports `opacity: 1`, and `document.elementFromPoint` returns it, because hit testing and painting are not the same question. Only a screenshot shows it.

Ship 1 leaves one pop behind, deliberately. The scene layer is outside `.run-screen`, so it does not ride the screen's fade: when the beat moves from visitor to night the room is still there through the beat of dark and then disappears the instant the new view mounts. Fixing it means holding the previous room through the fade out, which cannot apply to the reveal, where the canvas must be gone the moment the epilogue mounts and not a frame later. That special case belongs with the per beat rooms in ship 2.

## Measuring anything visual in a backgrounded pane

When the browser pane is not fronted, `document.hidden` is true and screenshots come back as stale frames. A style change made from a script computes correctly in the DOM and does not appear in the next screenshot, which reads exactly like a rendering bug in the app. The tell is to set `document.body.style.background = "red"` and screenshot: if the page is not red, the frame is stale and every visual conclusion from that screenshot is worthless.

A navigation forces a fresh capture, so the way to screenshot a specific state is to park the session at that beat through the API, then navigate and shoot. Layout and computed styles stay trustworthy throughout, so measurement by `getBoundingClientRect` is fine while the pane is hidden. Paint is not.

## Small screens

Checked at 390 wide and 768 wide, whole run: no horizontal scroll anywhere, every tap target 44px or larger, the vocalization panel goes to two columns under 480px, the About facts stack to one column, and the photo never overflows. Safe area insets are respected on the sides.

320 wide is the narrow end and it is checked too. A whole visitor close needs a little more room than the viewport is tall there, so the page scrolls by about 15px, but the button that carries the beat forward sits at 534 of 568 and is reachable without scrolling. That is the line: content may run past the fold, the control that moves the player never does.

One warning about measuring any of this in a browser pane that is not fronted. When `document.hidden` is true the transitions do not tick and timers throttle to about a second, so computed opacity reads 0 on a layer that is fading in and a click can look like it did nothing while its request is still in flight. Layout is still correct, so `getBoundingClientRect` and class names can be trusted. Check `document.hidden` before reading anything animated, and assert the class the client applied rather than the opacity the browser has not got round to.
