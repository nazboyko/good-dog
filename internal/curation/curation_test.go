package curation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuplicatesNamesBothDogs(t *testing.T) {
	problems := Duplicates(map[string]string{
		"ahs-1": "aaaaaaaaaaaaaaaa",
		"ahs-2": "aaaaaaaaaaaaaaaa",
		"ahs-3": "bbbbbbbbbbbbbbbb",
	})
	if len(problems) != 1 {
		t.Fatalf("problems = %v, want one duplicate", problems)
	}
	for _, id := range []string{"ahs-1", "ahs-2"} {
		if !strings.Contains(problems[0], id) {
			t.Errorf("%q does not name %s, the fix has to be obvious", problems[0], id)
		}
	}
	if strings.Contains(problems[0], "ahs-3") {
		t.Errorf("%q blames a dog with its own photo", problems[0])
	}
}

func TestDuplicatesQuietWhenEveryDogIsItself(t *testing.T) {
	problems := Duplicates(map[string]string{"ahs-1": "aaaa11112222", "ahs-2": "bbbb11112222"})
	if len(problems) != 0 {
		t.Errorf("distinct photos must not report a problem, got %v", problems)
	}
}

func TestChangedReportsDriftAndAcceptsNewDogs(t *testing.T) {
	m := Manifest{Version: 1, Photos: map[string]string{"ahs-1": "aaaaaaaaaaaaaaaa"}}
	problems := Changed(m, map[string]string{
		"ahs-1": "cccccccccccccccc",
		"ahs-2": "bbbbbbbbbbbbbbbb",
	})
	if len(problems) != 1 || !strings.Contains(problems[0], "ahs-1") {
		t.Fatalf("problems = %v, want only the changed photo reported", problems)
	}
}

func TestManifestRoundTripKeepsMismatchReportable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photos", "manifest.json")

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Photos) != 0 {
		t.Fatalf("a missing manifest must start empty, got %v", m.Photos)
	}
	if err := SaveManifest(path, m, map[string]string{"ahs-1": "original"}); err != nil {
		t.Fatal(err)
	}

	m, err = LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Photos["ahs-1"] != "original" {
		t.Fatalf("manifest lost the hash, got %q", m.Photos["ahs-1"])
	}
	// a replaced file must not overwrite the record that catches it
	if err := SaveManifest(path, m, map[string]string{"ahs-1": "replaced"}); err != nil {
		t.Fatal(err)
	}
	m, err = LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Photos["ahs-1"] != "original" {
		t.Errorf("manifest hash = %q, a mismatch that erases itself never reports twice", m.Photos["ahs-1"])
	}
}

func TestHashFileSeesContentNotName(t *testing.T) {
	dir := t.TempDir()
	same := filepath.Join(dir, "a.jpg")
	copy := filepath.Join(dir, "b.jpg")
	other := filepath.Join(dir, "c.jpg")
	for path, content := range map[string]string{same: "dog", copy: "dog", other: "different dog"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sumA, err := HashFile(same)
	if err != nil {
		t.Fatal(err)
	}
	sumB, err := HashFile(copy)
	if err != nil {
		t.Fatal(err)
	}
	sumC, err := HashFile(other)
	if err != nil {
		t.Fatal(err)
	}
	if sumA != sumB {
		t.Error("the same image under two names must hash the same, that is the crossed photo case")
	}
	if sumA == sumC {
		t.Error("different images must not collide")
	}
}
