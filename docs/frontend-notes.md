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

## Small screens

Checked at 390 wide and 768 wide, whole run: no horizontal scroll anywhere, every tap target 44px or larger, the vocalization panel goes to two columns under 480px, the About facts stack to one column, and the photo never overflows. Safe area insets are respected on the sides.
