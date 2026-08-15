package animal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// FixtureProvider serves the curated dogs from fixtures/dogs.json,
// identical interface to the live provider, used until the key flow
// lands and in all tests.
type FixtureProvider struct {
	animals map[string]Animal
	order   []string
	orgs    map[string]Organization
}

// fixtureFile mirrors fixtures/dogs.json. Descriptions arrive as raw
// listing html and go through the one normalization pipeline on load.
type fixtureFile struct {
	Version       int            `json:"version"`
	Note          string         `json:"note"`
	Organizations []Organization `json:"organizations"`
	Dogs          []struct {
		Animal
		DescriptionHTML string `json:"description_html"`
	} `json:"dogs"`
}

func NewFixtureProvider(path string) (*FixtureProvider, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixtures: %w", err)
	}
	var file fixtureFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse fixtures: %w", err)
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("fixtures version %d, this build reads version 1", file.Version)
	}
	p := &FixtureProvider{
		animals: map[string]Animal{},
		orgs:    map[string]Organization{},
	}
	for _, org := range file.Organizations {
		if _, dup := p.orgs[org.ID]; dup {
			return nil, fmt.Errorf("duplicate org id %q", org.ID)
		}
		p.orgs[org.ID] = org
	}
	for _, dog := range file.Dogs {
		a := dog.Animal
		a.Description = SanitizeDescription(dog.DescriptionHTML)
		if _, dup := p.animals[a.ID]; dup {
			return nil, fmt.Errorf("duplicate dog id %q", a.ID)
		}
		if !ValidStatus(a.Status) {
			return nil, fmt.Errorf("dog %s has unknown status %q", a.ID, a.Status)
		}
		if _, ok := p.orgs[a.OrgID]; !ok {
			return nil, fmt.Errorf("dog %s references unknown org %q", a.ID, a.OrgID)
		}
		if err := ValidLongStay(a); err != nil {
			return nil, fmt.Errorf("dog %s %w", a.ID, err)
		}
		// no invented timestamps, layer one never hallucinates
		if a.RetrievedAt.IsZero() {
			return nil, fmt.Errorf("dog %s has no retrieved_at, set the curation moment", a.ID)
		}
		p.animals[a.ID] = a
		p.order = append(p.order, a.ID)
	}
	return p, nil
}

func (p *FixtureProvider) Search(ctx context.Context) ([]Animal, error) {
	var pool []Animal
	for _, id := range p.order {
		a := p.animals[id]
		// synthetic shape examples must never reach a player
		if a.Synthetic || a.Status != StatusActive {
			continue
		}
		if ok, _ := PoolFilter(a); ok {
			pool = append(pool, a)
		}
	}
	return pool, nil
}

func (p *FixtureProvider) GetAnimal(ctx context.Context, id string) (*Animal, error) {
	a, ok := p.animals[id]
	if !ok {
		return nil, fmt.Errorf("animal %s not in fixtures", id)
	}
	return &a, nil
}

func (p *FixtureProvider) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	org, ok := p.orgs[id]
	if !ok {
		return nil, fmt.Errorf("organization %s not in fixtures", id)
	}
	return &org, nil
}

// GetStatus never errors for unknown ids: a listing the provider cannot
// see anymore means REMOVED_UNKNOWN, worded as left the shelter.
func (p *FixtureProvider) GetStatus(ctx context.Context, id string) (Status, error) {
	a, ok := p.animals[id]
	if !ok {
		return StatusRemovedUnknown, nil
	}
	return a.Status, nil
}
