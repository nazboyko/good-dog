---
name: game-architecture
description: Architecture rules for the GOOD DOG engine. Use when creating or changing game state, the session machine, events, module boundaries, or when deciding where any new logic lives.
---

# Game architecture

## The law
The game engine owns truth: state, rules, consequences, verified facts, progression. AI owns interpretation only: personality phrasing, narration, dialogue flavor. Never build prompt to model to "what happens next".

## Boundaries
- engine/ is plain TypeScript on the client for scene logic and rendering, and Go on the server for session truth
- The server session state machine is the single source of truth: day, beat, bond history, ending. The client renders and requests transitions, it never invents them
- React components contain zero gameplay logic. UI chrome only. The engine draws to canvas via a ref with its own rAF loop
- One multiplexed SSE stream at /events carries all server pushed events with a type field
- Provider payloads never reach the UI, everything goes through normalization into internal types

## State rules
- Every state transition is a pure function where possible and unit tested without any LLM call
- The comfort function (signal x preferences to shift) is a pure table driven function with golden tests
- Session state is serializable, a refresh mid scene resumes cleanly

## When adding anything
Run gate 0 first. If the new code wants to live in a React component and it is not presentation, it is in the wrong place.
