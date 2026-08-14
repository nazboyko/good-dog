# Build plan

Phase A is the challenge weekend. Phase B is a 3 day sprint in October after results. Between phases, everything committed after the deadline and before September 3 gets noted in the readme. After September 3 development is free.

## Phase A

### Day 0, Friday evening, 2 to 3 hours
1. Request the RescueGroups API key first, before anything else
2. Steel thread spike, about an hour, see docs/tech-stack.md, five green checks before real work
3. Repo, CLAUDE.md, skills, agents, hooks, CI, readme skeleton, LICENSE
4. AnimalProvider interface plus FixtureProvider with 12 real dogs and links
5. VerifiedFact and NarrativeInference types, dog sheet compiler with two step grounding, eval dataset of 15 profiles, golden tests
6. Article skeleton in docs/article-draft.md

### Day 1, Saturday
Morning is the emotional prototype law: prove the ending works. Skeleton run on fixture data: wake, one scent, one visitor, night, reveal with a real photo. Playtest it. If the reveal does not land, fix the reveal before building anything else.
Day: session state machine, scene player, vocalization panel and tail with inertia, two visitor archetypes, pure tested comfort function, shader v1 (color matrix, blur, low camera), four location backgrounds.
Evening: sniff mode with scent particles, radio pipeline (Ranger voice plus 5 pregenerated stories, SSE stream, night scene), the wake the shelter cascade.

### Day 2, Sunday
Morning: adoption day, three endings, epilogue with the reveal staging, share card.
Day: deploy to Fly, full playthrough three times on desktop and a 390px viewport, e2e journeys A, C, D, screenshots, architecture gif, 60 to 90 second demo video ending on the reveal.
Evening, publish by 20:00 CDT: finish the article, publish with #weekendchallenge, share on LinkedIn, answer early comments, tag v0.1.0-challenge.

### Checkpoints and the expansion ladder
Checkpoint 1, Saturday 14:00: emotional prototype works? If not, cut game day 1 and one archetype.
Checkpoint 2, Saturday 21:00: radio streams, day 1 core closed? If yes, one ladder step allowed. If not, sleep.
Checkpoint 3, Sunday 14:00: endings, epilogue, deploy, e2e green? If yes, one more ladder step until 17:00. If not, polish only.

Ladder, fixed order: 1 adoption detector light plus the bell (about 2h), 2 one minute as a dog (1.5h), 3 weather via Open-Meteo (1h), 4 real bark mode (3h, only if Sunday is free before 15:00), 5 empty kennel (1h, needs step 1).

Never in the weekend, no exceptions: night chorus multiplayer, dog of the day, parallel lives, solana, streamer mode, dreams.

If even the core slips, reverse cutting order: neighbors go static, game day 1 disappears, one archetype remains, radio shrinks to 3 stories. Never cut: reveal, dog vision, sniff mode, one full visitor scene, night with radio, the article.

## Phase B, October, three days

### Day 3, the world remembers
Adoption detector v2 with statuses ACTIVE, REMOVED_UNKNOWN, ADOPTED_CONFIRMED where possible. The bell in the radio stream. The empty kennel for returning players. Public bell wall page with the counter. Dog of the day. The constellation screen, dogs you have known, no collecting language. Article of the day: the bell is real.

### Day 4, the body
Real bark mode: mic, client side feature extraction, Gemini audio as a perception model, buttons stay as fallback. Scent memory fragments from verified facts only. Real weather and seasons in scenes and radio. The walk scene into the yard. Shader v2 with motion highlight and reduced motion support. Full soundscape pack. Article of the day: teaching Gemini to hear how a bark feels.

### Day 5, the others
Night chorus: players share one shelter, communication only by bark and body, no chat, no names until the epilogue. SSE plus POST, no websocket. Fallback for empty rooms: last night's recorded barks. Parallel lives map, city level aggregates only. One minute as a dog if not pulled earlier. Streamer mode. Solana pawprint wall only if time remains. Article of the day: a multiplayer game where nobody can talk.
