# Devlog

Three to five lines after every major block: what was decided, what was cut, what surprised. The article gets assembled from this file.

## Day 0
- Steel thread spike ran first, five checks before real work. SSE through the Vite proxy, click gated audio with range support, and the dog vision color matrix shader all went green in the browser on the first real try.
- ElevenLabs went green too: key valid, George voice picked for the radio, one probe call landed a real mp3. Budget guard and disk cache written with tests before the first paid call.
- Gemini code is done and unit tested, but the live call is blocked: the key's Google project has quota_limit_value 0 for the API, every model returns 429. Needs billing enabled or a fresh AI Studio key.
- Surprise of the day: the shader screenshot looked right but the color matrix was applied transposed, uniformMatrix3fv reads column major. A clean context reviewer caught what the eyeball test missed. The guardian also caught the extraction schema forcing invented moods on sparse notes, both fixed before commit.
- Linear space gotcha: the Vienot matrix is defined for linear rgb, applied to srgb encoded pixels it drifts green. Linearize, multiply, desaturate 15 percent in linear, then encode back. readPixels swatches assert red goes muddy yellow, green goes pale yellow, blue stays blue. The corner strip is spike page only, the real game view draws verification offscreen.
- Model retirement gotcha: the 2.5 flash line still shows in ListModels but carries zero quota for new projects, which reads exactly like a dead key. gemini-3.6-flash works on the same key. The server now preflights ListModels at startup and warns when the pin is missing, detect only, never auto switch. The retirement also renamed the thinking control, thinkingBudget 0 became thinkingLevel minimal, a 400 told us.

## Day 1
-

## Day 2
-
