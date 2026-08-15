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
