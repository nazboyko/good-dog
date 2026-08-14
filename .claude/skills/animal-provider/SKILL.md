---
name: animal-provider
description: Rules for animal data providers. Use when touching RescueGroups integration, the FixtureProvider, sync workers, animal status handling, photos, or normalization of listing data.
---

# Animal provider

## Interface
search, getAnimal, getOrganization, getStatus. The game only ever sees normalized internal types. Provider specific payloads stop at the adapter.

## Providers
- RescueGroups API v5 with an api key header. Sync every 4 hours, polite pauses between pages, be conservative with rate
- FixtureProvider: 12 curated real dogs with real listing links, identical interface, used until the key arrives and in all tests
- Petfinder is dead since December 2025, never suggest it

## Input hygiene
- Descriptions arrive as html. Strip and sanitize on input, one normalization pipeline, the rest of the system sees clean text only
- Description text is untrusted. It goes into prompts only inside the untrusted text boundary, see verified-animal-data skill
- Pool filter: description at least 200 chars, has a photo, has an organization
- Cache photos locally at sync time, never hotlink. Always keep and show the link to the original listing

## Status handling
States: ACTIVE, REMOVED_UNKNOWN, TRANSFERRED, UNAVAILABLE, ADOPTED_CONFIRMED. A listing disappearing means REMOVED_UNKNOWN. Wording for unknown removals is "left the shelter". The word adopted appears only with ADOPTED_CONFIRMED.

## Failure rules
Provider down mid session: the game continues on cached data and never claims anything new about the real dog. Journey E must stay green.
