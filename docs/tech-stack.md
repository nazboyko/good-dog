# Tech stack, decisions and traps

## Decisions
- Go backend, stdlib net/http, no framework
- modernc.org/sqlite, pure Go, no CGO, PRAGMA journal_mode=WAL
- One multiplexed SSE stream at /events with event types (radio, bell, neighbors, weather), never one stream per feature
- go embed for the built frontend, single binary, no CORS
- Frontend: Vite, TypeScript 5, React 18 for UI chrome only
- Game engine is plain TS: own rAF loop, own state, draws to canvas via ref, React never renders the scene
- Shader: WebGL1 compatible fragment shader, handle webglcontextlost and restore, 2D canvas fallback with CSS filters, cap devicePixelRatio at 2
- Gemini: gemini flash, structured outputs with responseSchema, flat schemas, maxOutputTokens with 2x headroom, extraction at temperature 0.1, generation at 0.8, schema validation then one retry then a canned default that is never cached
- ElevenLabs: library voices plus voice settings, not the voice design api, stories under 2500 chars, sfx generated once and cached forever, disk cache keyed by hash of text plus voice plus settings, budget guard counts characters before every call, Creator plan for one month
- RescueGroups API v5 behind AnimalProvider, FixtureProvider until the key arrives, sync every 4 hours with polite pauses, sanitize html on input, cache photos locally at sync time, always link the original listing
- Weather: Open-Meteo, no api key
- Deploy: Fly.io, min_machines_running 1, one region, volume for sqlite and audio cache, multi stage docker to a distroless image
- Later, night chorus: SSE plus POST /bark, no websocket

## Trap table, read before touching SSE, audio, or the shader

| Pair | Symptom | Fix |
| --- | --- | --- |
| SSE plus gzip middleware | events never arrive | no compression on /events |
| SSE plus http1 in dev | tabs hang silently | one multiplexed stream |
| SSE plus Fly auto stop | radio dies at night | min_machines_running 1 |
| Audio plus autoplay policy | silence, no errors | sound only after a user gesture, play button on /radio |
| Radio timing plus background tab | stream desyncs | schedule lives on the server, client only plays |
| React state plus particles | fps drops to 10 | engine outside React, canvas via ref |
| React StrictMode plus SSE | double subscriptions in dev | idempotent singleton connect |
| mattn sqlite plus docker | CGO build pain | modernc.org/sqlite |
| WebGL plus mobile Safari | context loss | restore handler plus 2D fallback |
| Provider descriptions plus prompts | injection and XSS | sanitize plus untrusted text boundary |
| ElevenLabs free tier plus radio pool | credits gone on Saturday | Creator for a month plus budget guard |
| Hotlinked photos plus dead urls | empty images in the epilogue | cache photos at sync |
| Gemini free tier plus a burst of calls | 429 quota per minute | one shared client, 8 rpm queue, backoff honoring retryDelay, flash pinned |
| Model lines retire for new projects | pinned model 429s with quota limit 0 while the key is fine | startup preflight lists models and warns, repin by hand |
| overflow-x hidden on the flex shell | overflow-y forced to auto, the shell becomes its own scroll box, scrollable views paint black past the fold | clip horizontal overflow on html and body only, never on the shell, and never add position sticky under a body with overflow set |
| A fixed full bleed layer at z-index 0 plus plain block content | the layer paints over the text, and only elements with a running animation stay visible, so a screen loses half its lines and keeps the other half | give the content its own stacking context, position relative plus z-index 1, and check a screen whose text is not animating |
| img.decode() plus a backgrounded tab | the promise never settles and never rejects, so anything awaiting it hangs forever and the scene simply never arrives | decode with createImageBitmap, which does not depend on the document being visible |
| go.mod plus a hand pinned base image | the deploy fails on the remote builder after everything local is green, because CI reads go-version-file and the Dockerfile is a separate copy nobody updates | make toolchain-check compares the two before a deploy, and go.mod stays the only place the version is decided |
| A new server plus a stale one already on the port | the new process logs its whole startup, fails to bind, exits, and the old binary keeps answering, so you debug code that is not running | fixed at the source: cmd/server takes the port with net.Listen before any other startup work, so a taken port is the first line of the log and the only one. Never move the bind back below the startup work |
| ImageBitmap plus UNPACK_FLIP_Y_WEBGL | the flag is silently ignored for a bitmap source, so switching a loader from an img element to createImageBitmap turns the whole scene upside down | set imageOrientation when the bitmap is created, and note the 2D path wants the opposite of what webgl wants |
| Screenshots plus a pane that is not fronted | frames come back stale, so a DOM change reads as correct and the picture disagrees | check document.hidden, probe with a red body background, and force a fresh capture by navigating |

## Steel thread spike, Friday, about one hour, before real work
1. Go serves SSE through the Vite proxy and an event shows in the browser
2. A button resumes AudioContext and plays an mp3 from Go with range support
3. One Gemini call returns valid JSON against the schema
4. One ElevenLabs call lands an mp3 in the cache and plays through the same button
5. A canvas with the color matrix fragment shader renders over a test image
Five green checks mean the risky perimeter is verified and the weekend is assembly, not debugging.

## Heartbeats and reconnect
SSE: comment ping every 20 seconds, flush after every event, support Last-Event-ID or resend current state on reconnect. EventSource reconnects by itself, the server must tolerate it.
