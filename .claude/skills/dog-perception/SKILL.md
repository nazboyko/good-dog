---
name: dog-perception
description: Sensory constants and rules for dog vision, scent, hearing, and the understanding system. Use when touching the shader, particles, audio filters, subtitles, tone tags, or any presentation of senses.
---

# Dog perception

## Vision
- Dichromatic palette, blue and yellow. Color matrix in the fragment shader, WebGL1 compatible
- Mild blur past midfield, camera at 50 to 70 cm height, wider field feel, slight highlight on movement
- Hard rule: never encode meaning in red versus green anywhere in the game
- Cap devicePixelRatio at 2, handle webglcontextlost with a restore and a 2D CSS filter fallback

## Scent
- Particle trails: color is emotion (calm deep blue, excitement bright yellow, fear flickering white, food warm gold)
- Age is opacity and spread, fresh trails are dense and directional
- Sniff mode on hold: time slows, visuals dim, scents brighten, sounds render as ripples through walls
- Scent is navigation, the player follows history, not quest markers

## Understanding system
- Player intent is always clear to the player
- Human speech modes: dog ears (lowpass about 400Hz via BiquadFilterNode plus tone tags), subtitles (full text, muffled audio), human ears (clean)
- Mismatch narrator after every signal, two short lines, you meant and they heard, in every mode
- Vocabulary: repeated words unlock and play clean inside muffled speech, the dog's real name rings clear the first time it is spoken

## Presentation
- Reserved space plus fade, never insert and push. Every line, photo, and panel that will appear has its space from the start, and appears by opacity in place. Nothing above it moves by a pixel. The screen breathes, it does not jump
- Nothing pops. All appearing and disappearing UI fades or eases: 400 to 600ms for quiet moments like the reveal, 150 to 250ms for interactive feedback like the vocalization panel. No instant show or hide
- Photos reserve their box from the real dimensions before they load, so a load never reflows text
- Reduced motion turns fades instant but keeps every reserved space, so it never reintroduces the jump

## Accessibility
Subtitles for all audio always. Reduced motion disables particles and camera moves. Everything readable without color alone.
