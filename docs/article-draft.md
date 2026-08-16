---
title: "GOOD DOG: you are the dog, and the dog is real"
published: false
tags: devchallenge, weekendchallenge, gamedev, go
cover_image: docs/img/bella-reveal.png
---

*This is a submission for [Weekend Challenge: Dog Days Edition](https://dev.to/challenges/weekend-2026-08-13)*

## What I Built

A game where you spend three days in a shelter as a dog. Not a dog. A dog: one of twelve real, adoptable animals, taken from real shelter listings and checked by hand. You cannot speak. You have a bark, a whine, a growl, a howl and silence, and you watch every one of them land wrong. You mean "I am glad you are here." She hears "this dog is loud." She moves on to the next kennel.

At the end you learn the dog you were is real, you see her photo in her real colours for the first time, and you get the link to her listing. If she is still waiting, the game says so. One of the twelve went home on Saturday, halfway through the weekend, and the game had to learn to say that instead. That is the last section of this post.

![The reveal. Bella, a real dog at Animal Humane Society. That was your three days. Bella is real, and was adopted on August 15, 2026.](docs/img/bella-reveal.png)

## Demo

Play it: **https://good-dog.fly.dev**

{% youtube 4mwOHnW--B0 %}

The night broadcast is voiced, every line, and it is worth hearing with the sound on.

## Code

{% embed https://github.com/nazboyko/good-dog %}

Go standard library, one binary, one SQLite file, React for the chrome and plain TypeScript for the game. 57 commits inside the window.

## How I Built It

One rule: **the engine owns truth and the model owns wording.** Real name, photo, shelter and status are never invented, and a model never decides what happens next.

The through line, which the devlog surfaced on its own: **something that looked right to a person and was wrong on inspection, caught by a check rather than a glance.** It happened four times that mattered.

### The dog sheet: two model calls and a verifier that cannot be argued with

The first call extracts facts and may only quote: anything not a verbatim substring of the listing is dropped. The second writes the personality, every inference cites the fact ids it came from, and a verifier rejects anything it cannot trace to the page.

The number: **19 eval profiles**, real listings with hand-written expectations. The first live run failed two, both real: a "gentle senior" reached a radio seed for a high energy dog, and a placement rule ("must be an only dog") became a fear. Both are forbidden by the prompt and, separately, by the verifier, because a prompt is a request and a verifier is a wall.

The trade off: the model writes flourishes where I asked for neutral, so uncited voice and movement snap to fixed neutral sentences. That costs texture on thin listings. The alternative is describing a real animal wrong.

### Dog vision: a colour matrix, and the room I called a room

You see the whole game through a dog's eyes: dichromatic blue and yellow, a Viénot deuteranopia matrix in a WebGL shader over a photograph of a kennel. Three days of that is what makes the real photo at the end land as the first true image in the game. The shader is not switched off at the reveal, it ends there, and a test fails if a canvas is ever emitted past that line.

![The kennel as you see it, then as the dog sees it. Both frames rendered through the game's own shader path.](docs/img/dogvision.gif)

The number: assets went from **12MB of PNG to 512KB of webp**.

Here is the first of the four. I swapped the image loader to `createImageBitmap` and the room shipped upside down, because `UNPACK_FLIP_Y_WEBGL` is ignored for a bitmap source. Every acceptance check passed: the colour swatch is three texels in a row so a flip does nothing to it, and the scene check reads the centre pixel, the one pixel a flip leaves alone. **I looked at four screenshots, twice each at three widths, and read every one as a shelter kennel.** An inverted kennel in dim two colour light is still a floor, a gate and a wall. The blanket was on the ceiling and I did not ask the pixels.

A reviewer with no memory of the session caught it with pixel measurements. Then the guard I wrote for it was worthless in the same way: it measured the source by drawing the same bitmap the renderer was about to use, so when the picture flipped the measurement flipped with it. I put the bug back to watch the guard fail, and it passed. The fix is a second decode, explicitly upright, as independent ground truth. From the devlog, verbatim: *"a check that shares its source with the thing it checks is not a check."*

### The visitor: a comfort function, and the test that disarmed itself

A visit is four exchanges. Each signal lands in one of five bands and the only readout is a body: a phone, a glance at the next kennel, a hand flat on the gate. A narrator names the mismatch, "you meant / she heard," and the whole game is the distance between those two lines. The number: **1,296 reachable four-exchange scenes**, and the test walks all of them.

That is the second of the four. There was a rule that a badly heard signal can never leave the visitor at their warmest, and a test named after it. When the scene grew from one exchange to four, the band became cumulative while the narrator stayed per signal, so silence, silence, silence, growl put "she heard: maybe not this one" directly above "she puts a hand flat against the gate." The test did not fail. Updating it for four exchanges had meant building a scene of four identical answers, and a uniform scene is the one family where those two lines can never disagree. It still named the rule in its comment and no longer checked any part of it. Two reviewers found it from opposite ends: one read the screen and called it a contradiction, one read the diff and called it a rule the test had stopped enforcing. Neither would have been enough alone. It is a rule in the repo now: every new guard is checked by putting its bug back and watching it fail before the unit ships.

### The radio: pregenerated, paced by the server, and 81 of 81

At lights out a host reads the row: real dogs from the pool, one true thing each does, their name and where they are. Your own dog goes last and is the only one allowed to say the reveal line. Every word is recorded ahead of time and shipped in the image; nothing is synthesized while somebody is listening.

The numbers: **82 lines across twelve dogs, eight library voices picked by size and age off the listing**, about **1,250 ElevenLabs credits** including five sound effects for the dog's own voice. The trade off: voiced, the night runs about 55 seconds; muted, it keeps the 32 second reading pace, because a judge with the sound off deserves a night too. Every line is a subtitle first and a voice second.

Here is the third of the four. **The recorder reported 81 of 81 done while Lutsen's closing line, "She is real, and she is still here," played silent on the live URL.** Three dogs share one library voice. Lutsen's sheet says she is quiet, so she reads at a steadier stability, and the cache key includes that setting. The recorder deduped its plan on the voice alone: it recorded one dog's read of the sentence, never planned Lutsen's, and told me the cache was warm. The close of her night was the one line nobody said out loud. It dedupes on the exact key the night looks up now, and the test builds the collision on purpose, two dogs, one voice, two stabilities, because the first version read my local sheet cache and CI does not have one, so it could not have failed where it runs.

### What I cut and why

- **Live adoption data.** The provider layer is built behind an interface and the RescueGroups sync is not. The twelve dogs are read from a curated file, each verified by hand, and nothing in the game claims otherwise. A live sync in a weekend would have meant extraction against listings nobody had read, and the whole point is that somebody read them.
- **A thirteenth dog.** Smudge was cut because every sentence on his page also appears on another dog's page. A player could not have told you afterward who he was.
- **Sniff mode and the bark cascade.** Cut Saturday night, written down as the plan rather than a slip.
- **The prettier link.** A rescue's own listings page had no per-dog link, so the reveal would have dropped you into a grid of other dogs. The link resolves to this dog or it does not ship.
- **Two rules that fight.** Names are deduped so a night never has two Bellas, and shelters are spread so the closing line varies. In this pool the only dog at a second shelter is a third Bella. The code is right, and I wrote it down rather than tune around it.

### Dependencies and credit

Go 1.25 standard library, plus [modernc.org/sqlite](https://modernc.org/sqlite). [React](https://react.dev) 18, [Vite](https://vite.dev) 6, [TypeScript](https://typescriptlang.org) 5.8, [Vitest](https://vitest.dev) 3, and [jsdom](https://github.com/jsdom/jsdom) with [Testing Library](https://testing-library.com) for the tests that click a real button. [Google Gemini](https://ai.google.dev) structured outputs. [ElevenLabs](https://elevenlabs.io) library voices and the sound effects API. [Fly.io](https://fly.io). Listings from [Animal Humane Society](https://www.animalhumanesociety.org) and [Ruff Start Rescue](https://www.ruffstartrescue.org), Minnesota, quoted rather than rewritten. Colour matrix from Viénot, Brettel and Mollon (1999).

202 tests in Go and 70 in TypeScript. The repo is tagged `v0.1.0-challenge` at submission, and anything after it is listed in the README.

## Prize Categories

**Google AI.** Gemini structured outputs on both steps of the sheet compiler. The interesting part is what the schema is not allowed to say: extraction returns quotes and field references, generation returns inferences with citation ids, and a verifier rejects anything the listing did not support. Structured outputs made grounding checkable. Two prompt injection profiles are in the eval set and both pass. Model pinned to `gemini-3.6-flash`, with a boot preflight that warns if the pin has gone.

**ElevenLabs.** Nine voice buckets, three sizes by three ages, eight in use across these twelve dogs, settings moved only by the sheet's voice profile so a dog the listing calls quiet reads steadier without becoming a different person. The sound effects API for five vocalizations, described as one medium dog close to the microphone and nothing else. Everything pregenerated into a disk cache keyed on text plus voice plus settings, with a budget guard before every call. The bell line, "That was Bella. She went home," is one of the recordings.

## The Bell

While this was being built, one of the dogs in it was adopted.

Bella is an American Pit Bull Terrier mix, six years and four months old, and she was at the Animal Humane Society adoption center in Coon Rapids, Minnesota. She was one of the twelve I curated on Friday night, checked by hand against her real listing, with her real photo and her own words from the shelter. On Friday, August 14, her listing began like this, and these words are the ones saved into the game's fixture file that night, committed in `f116bca`:

> My family was unable to care for me, so I came to Animal Humane Society to find a new home. I'm an affectionate dog that loves being close and sharing snuggles.

By Saturday her page had changed. Those words are gone from it. It now says, and this is what her page says today:

> We think Bella is pretty great, too, but she is no longer available for adoption. Bella was adopted on August 15, 2026!

She was in the pool while I was writing the code that tells players she is still waiting. Somebody met her, and she has a home. Congratulations, Bella.

Here is the fourth of the four. **`make verify-fixtures` passed all twelve dogs that morning, green, while Bella had already been adopted.** It checked that her page returned 200 and that her name was on it. Both were true. The page also said, in a sentence, that she was gone, and nothing was reading the sentence. I found it by hand, and then I made the tool read what the shelters actually write, tested against saved snippets of the real pages rather than strings I typed to make it pass. When I set her back to ACTIVE to check the check, it said: *"listing says adopted on August 15, 2026, fixture says ACTIVE."*

Then the question the whole project is built around: what does the game do about her.

The easy answer is to quietly delete her. That would be a lie by omission, and it would throw away the one thing this game exists to say, which is that these are real animals whose situations change while you are not looking. So she stays in. Three things follow.

Her reveal tells the truth. A player who spends three days as Bella does not reach the end and get told she is still waiting. The reveal says *"Bella is real, and was adopted on August 15, 2026,"* and the date is her listing's, not mine. This is what it looks like, and it is the frame the video ends on:

![Bella's reveal: That was your three days. Bella is real, and was adopted on August 15, 2026. Bella was listed among Animal Humane Society's long stay dogs.](docs/img/bella-reveal.png)
 The seam line, "That was your three days," shows on every one of her endings, because for her the fiction and the listing disagree whatever you did: you spent three days in a kennel with a dog who was not in it.

The radio rings the bell. When Bella comes up on somebody else's night, the host does not say where she is waiting. He says *"That was Bella. She went home."* One line, the same slot as everyone else's, no music, read last so the night walks down the row and then names the one who is not in it.

And the word adopted is allowed in exactly one place. Everywhere else the game says listed, or no longer listed, and never guesses at why a page went quiet, because some of the reasons are not happy ones. Bella's listing said the word first. That is the only reason the game gets to say it.

The rule underneath all of this: what the listing says right now outranks whatever happened in your three days. The game can write an ending. It cannot write a fact.

## Every dog in this post is real

Twelve dogs, two shelters, curated by hand on August 14 and re-checked by hand on August 16. Eleven are still listed as I write this. One went home.

![Arya, ten years old, at her own reveal.](docs/img/reveal.png)

Arya, Animal Humane Society, Golden Valley: https://www.animalhumanesociety.org/animal/adoption/61293730

Preston, seventy eight pounds, Animal Humane Society: https://www.animalhumanesociety.org/animal/adoption/61204992

Venus, four, Ruff Start Rescue: https://new.shelterluv.com/embed/animal/RSMN-A-9548
Sugar Bear, eleven years and six pounds, Ruff Start Rescue: https://new.shelterluv.com/embed/animal/RSMN-A-12209
Lutsen, Ruff Start Rescue: https://new.shelterluv.com/embed/animal/RSMN-A-11938

And Bella, who is not waiting: https://www.animalhumanesociety.org/animal/adoption/60796517

If you play, you will be one of them. Go and meet the one you were.
