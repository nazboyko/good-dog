---
name: shelter-radio
description: Rules for the night radio. Use when generating radio stories, building the SSE stream, the pregeneration queue, the scheduler, or the standalone /radio page.
---

# Shelter radio

## Format
Static jingle, Old Ranger host (slow, warm, gravelly library voice), 3 to 5 dog stories in distinct voices, each grounded in a real description and ending with the real name and place, a weather line from the shelter's real location, the bell only on real removals with careful wording.

## Tone
Restraint. Short lines with pauses beat paragraphs. "My name is Moose. I like tennis balls. I have been waiting here for a while." Never cartoon characters, never a sad piano wall. Poetic interpretation of grounded facts only, narrative-guardian reviews every story.

## Pipeline
Pregeneration worker: Gemini text (through verified-animal-data rules), ElevenLabs audio to disk cache, the stream reads only ready segments. Zero generation under a live listener. Budget guard counts characters before every ElevenLabs call.

## Streaming
The schedule lives on the server. SSE announces segment and position, the client only plays. Heartbeat comment every 20 seconds, flush after every event, tolerate reconnects, no compression on the stream route. The standalone /radio page starts with a press play button because of autoplay policy.
