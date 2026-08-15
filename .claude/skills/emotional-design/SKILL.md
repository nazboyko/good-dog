---
name: emotional-design
description: Tone and emotional safety rules for all player facing content. Use when writing any UI copy, scene text, endings, radio content, share cards, or the article.
---

# Emotional design

## Never
- Fake urgency, fake countdowns, invented euthanasia deadlines
- Fabricated abuse, exaggerated medical claims
- Guilt based copy, manipulative donation prompts
- Constant sad music, misery as a default state
- Collectible language about animals, no "4 of 100 dogs"
- Claiming adoption without confirmed status
- Revealing the real photo before the epilogue

## Always
- "Bailey is still here" over "Bailey desperately needs you before it is too late"
- The real situation is enough
- Warmth and humor make the emotional scenes stronger, the wake the shelter cascade exists on purpose
- Quiet beats loud: the chosen ending is soft, the empty kennel has no celebration animation
- Every ending is honest and none is a loss

## Voice
Short sentences. Plain warm english. UI copy minimal. The reveal uses pauses, not exclamation points.

## Emoji in listing text
Fosters write with emoji and the record keeps them, exactly as written. Quoted facts and any player facing text derived from a description render through forDisplay in web/src/engine/display.ts, which strips emoji at the display layer only. Never strip at the data layer, never at curation, never before storing. The record keeps the truth, the screen keeps the style.
