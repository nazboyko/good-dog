# Submission article draft

Working titles:
- GOOD DOG: every life you live in this game belongs to a real shelter dog
- I built a game where you are the dog. The dog is real, and he is still waiting.

Structure:
1. What I Built. The thesis plus an epilogue screenshot, no mechanic spoilers.
2. Demo. Link plus a 60 to 90 second video that ends on the reveal.
3. Code. Embedded repo.
4. How I Built It. Four parts with numbers: two step grounding in the compiler, the shader as free art consistency, the radio with pregeneration and SSE, what I cut and why.
5. Prize Categories. Google AI: grounding, structured outputs. ElevenLabs: a voice per dog plus sound effects.
6. The Bell. Short section on the adoption detector and careful wording. Emotional close.
7. Every dog in this post is real. Three or four photos with listing links.

Rules: natural english, no fluff, no em dashes, one strong paragraph beats three average ones, first image is the epilogue screen. Pull quotes from docs/devlog.md.

## Bella went home

While this was being built, one of the dogs in it was adopted.

Bella is an American Pit Bull Terrier mix, six years and four months old, and she was at the Animal Humane Society adoption center in Coon Rapids, Minnesota. She was one of the twelve dogs I curated on Friday night, checked by hand against her real listing, with her real photo and her own words from the shelter. Her listing began:

> My family was unable to care for me, so I came to Animal Humane Society to find a new home. I'm an affectionate dog that loves being close and sharing snuggles.

By Saturday her page had changed. It now says:

> We think Bella is pretty great, too, but she is no longer available for adoption. Bella was adopted on August 15, 2026!

She was in the pool while I was writing the code that tells players she is still waiting. Somebody met her, and she has a home. Congratulations, Bella.

Then the question the whole project is built around: what does the game do about it.

The easy answer is to quietly delete her. That would be a lie by omission, and it would also throw away the one thing this game exists to say, which is that these are real animals whose situations change while you are not looking. So she stays in. Her status is ADOPTED_CONFIRMED and her adoption date is the one her own listing gives.

Three things follow from that.

Her reveal tells the truth. A player who spends three days as Bella does not reach the end and get told she is real and still waiting, because she is not. The reveal says she was adopted, and it says the date, and the date comes from her listing rather than from me.

The radio rings the bell. The night broadcast is a host reading the row after lights out, dog by dog. When Bella comes up he does not say where she is waiting. He says she went home. One line, in the same quiet voice as everything else, no music and no fuss.

And the word adopted is now allowed in exactly one place. Everywhere else the game says listed, or left, and never guesses at why a listing went quiet, because some of those reasons are not happy ones. Bella's listing said the word first. That is the only reason the game gets to say it.

The rule underneath all of this: what the listing says right now outranks whatever happened in your three days. The game can write an ending. It cannot write a fact.
