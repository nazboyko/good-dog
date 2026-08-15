.PHONY: fetch-photos

# Shelter photos stay out of the repo. This fills cache/photos/ from the
# photo_url of every real fixture dog, skipping ones already cached.
fetch-photos:
	go run ./cmd/fetchphotos
