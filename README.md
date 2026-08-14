    # GOOD DOG

A game where you are the dog. The dog is real.

You live a few days in a shelter as a real adoptable dog pulled from live adoption data. You cannot speak. You have a bark, a tail, a nose, and time. At the end you learn the dog you were is real and still waiting.

## Demo

Coming during the challenge weekend. Deployed link and a short video will land here.

## Run it

```
# backend
go run ./cmd/server

# frontend, dev
cd web && npm install && npm run dev
```

Copy .env.example to .env and fill in the keys.

## How it works

Real listing data goes through an extraction step that keeps only verified facts, then a generation step builds the personality from those facts alone. The game engine owns everything true, the AI only interprets. Details in docs/game-design.md and docs/tech-stack.md.

## Dependencies and credits

- modernc.org/sqlite
- (keep this list current, every non trivial dependency gets a line)

## Post deadline commits

Everything after the tag v0.1.0-challenge came after the challenge deadline.

## Setup after cloning

```
git config core.hooksPath .githooks
```
