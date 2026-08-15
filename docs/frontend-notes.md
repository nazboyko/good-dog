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

## The reveal fits the fold

The reveal is sized to fit one viewport by design: the photo is capped by viewport height (50vh on wide screens, 42vh on phones, 40vh under 760px tall) and the lines tighten on short screens, so photo plus lines plus the button fit a phone, a tablet and a 720 tall laptop without scrolling. Verified at 390x844, 768x1024 and 1280x720. If a line would still land below the fold, the view eases down to it over 520ms in step with the fade, and any manual scroll cancels the follow for the rest of the reveal.

## Views own their layout from zero

Every view in the game opens at its top, with no dead space above or below. Each view calls `useViewTop()` on mount, which resets the scroll before the first paint, so the previous screen's offset never leaks into the next. The run shell is top aligned, not centered, so a short beat sits at the top and its button lands where a thumb can reach it. Reserved space is a reveal staging rule and stays inside the reveal: the About panel is its own view and the reveal is unmounted underneath it, never hidden in flow. The mobile pass checks this on every transition, top of content is within the top padding of the viewport and scrollY is zero.

## Small screens

Checked at 390 wide and 768 wide, whole run: no horizontal scroll anywhere, every tap target 44px or larger, the vocalization panel goes to two columns under 480px, the About facts stack to one column, and the photo never overflows. Safe area insets are respected on the sides.
