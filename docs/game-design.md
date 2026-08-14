# Game design

## Core loop
Wake in a kennel. Morning exploration. Visitor hours, 2 or 3 scenes. Evening. Night with Shelter Radio. Repeat. Day 3 is Adoption Day with three honest endings. Then the epilogue reveal. Then "live another life" with a new real dog.

One run is 20 to 35 minutes. Replayability comes from real dogs being different.

## The reveal, most important scene in the product
The player starts knowing only name, age, breed. The playable representation is stylized. The real photo appears for the first time in the epilogue, with pauses: "You were Bailey." then "Bailey is not a character." then the photo, then real age, shelter, distance, days waiting, then the MEET BAILEY button to the real listing. Nothing may leak the photo earlier. The transparency panel (real description next to what the game inferred from it) unlocks only after the reveal.

## Dog sheet compiler
Input: a normalized animal record. Step one, extraction: Gemini pulls verified facts from the description only, temperature 0.1. Step two, generation: personality matrix (energy, confidence, sociability, patience, food drive, noise sensitivity), fears only when the description supports them with a quote, quirks, movement profile, voice profile, radio story seed. Generation may use only the extracted facts. Structured output with schema validation, one retry, then cached default. Dog sheets are cached forever.

## Dog vision
Full screen shader: dichromatic blue and yellow palette, mild blur past midfield, low camera at 50 to 70 cm, wider field feel, moving things slightly highlighted. Never encode meaning in red versus green. Sniff mode on hold: time slows, scent trails appear as particles, color is emotion (calm deep blue, excitement bright yellow, fear flickering white, food warm gold), age is opacity, scents pass weakened through walls, sounds show as ripples.

## Understanding system, how language works
The player always understands their own intent. Understanding humans has three modes:
1. Dog ears, default: human speech is lowpass filtered audio with tone tags on screen instead of words. Learned words come through clearly.
2. Subtitles: full english subtitles, audio stays muffled.
3. Human ears: clean audio plus subtitles for players who just want the story.
The mismatch narrator is always on in every mode. After each player signal show two short lines: "you meant: come play" and "she heard: too loud, too sudden". This is the game's clarity and its heart at the same time.
Growing vocabulary: repeated words like walk, treat, good, no, and the dog's real name unlock and start sounding clean. The first time a visitor says the dog's real name it rings out clear in the middle of muffled speech.

## Communication
Voice: playful bark, alert bark, whine, low growl, howl, and silence as a deliberate option. Body: tail with emotional inertia the player must consciously manage, posture (sit, lie, play bow, stand), distance (approach, hold, retreat). Visitors react to combinations. Mixed signals confuse them. The game never says "wrong", it shows consequence.

## Visitors
Generated as sheets: archetype, hidden preferences, dealbreakers, patience, scent aura. A scene is 4 to 8 beats. Comfort is invisible as a number and readable through body language and aura shifts. Honest matching: some visitors can never pick this dog, the goal then is parting well. Interactions log into bond history.

## Neighbors and night
Neighbor kennels hold other real dogs from the pool with idle behavior from their sheets. A too loud night bark cascades into the whole shelter waking up, comedy on purpose, the game must not be relentlessly sad. Night is Shelter Radio: a real SSE stream. Old Ranger hosts, 3 to 5 dog stories in their own voices grounded in real descriptions, each ends with a real name and place, weather segment from the shelter's real location. The bell rings when the sync worker sees a dog leave the listings, careful wording only.

## Adoption day and endings
Final visitor is synthesized from bond history, a longer scene in a new room with sensory overload the player breathes through in sniff mode. Endings: chosen (quiet and warm), another dog chosen (that night the radio tells your story), nobody today (tomorrow is another day, day 4 exists). No ending is a loss.

## Emotional hooks in core
Good dog as the only reward, no counter shown. Real waiting time in the epilogue: "You spent 40 minutes as Bailey. Bailey has been waiting 214 days." The front door sound makes every ear in the shelter rise. Share card: "I WAS MILO for 38 minutes. He is real."

## Hard content rules
No invented trauma, medical claims, aggression, or surrender reasons. No fake urgency, no invented deadlines, no guilt copy. Never claim adoption without confirmed status. Never turn animals into collectibles. The real situation is enough.
