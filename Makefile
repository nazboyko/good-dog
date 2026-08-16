.PHONY: fetch-photos verify-fixtures

# Shelter photos stay out of the repo. This fills cache/photos/ from the
# photo_url of every real fixture dog, skipping ones already cached.
fetch-photos:
	go run ./cmd/fetchphotos

# One polite request per dog. Reports which listings still answer and still
# name the dog, and whether the cached photo still matches the manifest.
# Read only: a failure is a hand recheck, never an automatic status change.
verify-fixtures:
	go run ./cmd/verifyfixtures

# Record every radio line the game can play, for all twelve dogs, into
# the disk cache. Nothing is ever synthesized while somebody is
# listening, so this has to run before the first player arrives. Safe to
# repeat: a warm cache costs nothing.
voices:
	go run ./cmd/prepvoices

# What it would cost, without spending anything.
voices-dry:
	go run ./cmd/prepvoices -dry-run

# Build the frontend into the place the binary embeds it. The Dockerfile
# does the same thing; this is for running the real single binary
# locally, which is the only way to test what actually deploys.
embed:
	cd web && npm ci --silent && npm run build
	rm -rf internal/webui/dist
	cp -r web/dist internal/webui/dist

# One binary with the game, the api, the stream and the audio in it.
build: embed
	go build -o bin/good-dog ./cmd/server

# Refuse to ship an empty page. A deploy that embeds nothing still
# builds, still boots and still answers 200, which is the worst way to
# find out.
embed-check:
	@test -f internal/webui/dist/index.html || (echo "internal/webui/dist is empty, run make embed" && exit 1)
	@ls internal/webui/dist/assets/*.js >/dev/null 2>&1 || (echo "no js bundle embedded" && exit 1)
	@echo "embedded: $$(du -sh internal/webui/dist | cut -f1) of frontend"

# The Dockerfile is the only hand written copy of the Go version: CI
# reads go.mod directly. A deploy that fails on a toolchain mismatch
# costs a round trip, so catch it here instead.
toolchain-check:
	@modver=$$(awk '/^go /{print $$2}' go.mod); \
	imgver=$$(awk -F'golang:' '/FROM golang:/{split($$2,a,"-"); print a[1]}' Dockerfile); \
	modmm=$$(echo $$modver | cut -d. -f1,2); \
	if [ "$$modmm" != "$$imgver" ]; then \
		echo "go.mod wants $$modver but the Dockerfile builds on golang:$$imgver"; exit 1; \
	fi; \
	echo "toolchain: go.mod $$modver, image golang:$$imgver"
