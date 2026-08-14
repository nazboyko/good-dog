package audiocache

import (
	"os"
	"testing"
)

func TestKeyChangesWithEveryInput(t *testing.T) {
	base := Key("hello", "voice1", "s0.5")
	same := Key("hello", "voice1", "s0.5")
	if base != same {
		t.Error("same inputs must give the same key")
	}
	for name, other := range map[string]string{
		"text":     Key("bye", "voice1", "s0.5"),
		"voice":    Key("hello", "voice2", "s0.5"),
		"settings": Key("hello", "voice1", "s0.9"),
	} {
		if other == base {
			t.Errorf("changing %s must change the key", name)
		}
	}
	if len(base) != 20 || base[16:] != ".mp3" {
		t.Errorf("key %q has unexpected shape", base)
	}
}

func TestGetPutRoundTrip(t *testing.T) {
	c := New(t.TempDir() + "/nested/audio")
	key := Key("hello", "v", "s")
	if _, ok := c.Get(key); ok {
		t.Fatal("empty cache should miss")
	}
	path, err := c.Put(key, []byte("mp3bytes"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get(key)
	if !ok || got != path {
		t.Fatalf("Get = %q, %v after Put %q", got, ok, path)
	}
	data, err := os.ReadFile(got)
	if err != nil || string(data) != "mp3bytes" {
		t.Fatalf("read back %q, %v", data, err)
	}
}
