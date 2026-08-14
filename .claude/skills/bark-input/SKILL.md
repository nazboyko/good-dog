---
name: bark-input
description: Rules for player vocal input. Use when building the vocalization panel, real bark mode, microphone capture, audio feature extraction, or Gemini audio calls.
---

# Bark input

## Model
The AI never translates dog language. It estimates how the vocalization is perceived. Player intent versus perceived signal, the mismatch is the mechanic.

## Panel mode, default
Six options: playful bark, alert bark, whine, low growl, howl, silence. Silence is a deliberate choice. Always available, full keyboard access.

## Real bark mode, October
- getUserMedia after an explicit opt in, https only
- Feature extraction on the client with AnalyserNode: intensity, duration, attack, rough pitch
- Raw audio is never stored and never leaves the client by default, Gemini receives features plus at most a short clip with consent
- Permission denied or any failure falls back to the panel silently, journey B stays green

## Perception mapping
Features map to the nearest panel vocalization plus an intensity modifier through a pure tested function. Gemini interprets perception, the engine applies consequence.
