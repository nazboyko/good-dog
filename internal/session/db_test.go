package session

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDBSaveLoadRoundTrip(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	state := run(t, FullRun, NewState(FullRun, t0), adv, adv, say(Whine), adv, say(Howl), adv, adv)
	row := Row{ID: "abc", DogID: "rsmn-a-9548", OrgID: "ruff-start-rescue", Rails: FullRun.Name(), State: state, UpdatedAt: t0}
	if err := db.Save(ctx, row); err != nil {
		t.Fatal(err)
	}
	got, err := db.Load(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := state.Marshal()
	b, _ := got.State.Marshal()
	if string(a) != string(b) {
		t.Errorf("state changed through the database:\n%s\n%s", a, b)
	}
	if got.DogID != "rsmn-a-9548" || got.OrgID != "ruff-start-rescue" || got.Rails != "full" {
		t.Errorf("row identity lost: %+v", got)
	}
	// a second save is a replace, not a duplicate: day 2 wake, scent, visitor, answer
	state2 := run(t, FullRun, state, adv, adv, say(Silence))
	row.State = state2
	row.UpdatedAt = t0.Add(time.Minute)
	if err := db.Save(ctx, row); err != nil {
		t.Fatal(err)
	}
	got, _ = db.Load(ctx, "abc")
	if got.State.Signal != Silence || !got.UpdatedAt.Equal(t0.Add(time.Minute)) {
		t.Errorf("replace did not land: %+v", got.State)
	}
}

func TestDBLoadMissIsNotFound(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Load(context.Background(), "never"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id must be ErrNotFound, got %v", err)
	}
}

func TestDBUnreadableRowIsAMissNotACrash(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	// a row written by some future build with a version this one cannot read
	if _, err := db.sql.ExecContext(ctx, `insert into sessions values ('old', 'd', 'o', 'short', '{"version":99,"day":1,"beat":"wake","bond":[],"started_at":"2026-08-15T09:00:00Z"}', '2026-08-15T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Load(ctx, "old"); !errors.Is(err, ErrNotFound) {
		t.Errorf("an unreadable row must degrade to a miss, got %v", err)
	}
}

func TestDBSurvivesRestart(t *testing.T) {
	// the same file opened twice stands in for a server restart
	path := filepath.Join(t.TempDir(), "sessions.db")
	ctx := context.Background()
	state := run(t, ShortRun, NewState(ShortRun, t0), adv, adv, say(Whine))

	first, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Save(ctx, Row{ID: "life", DogID: "d", OrgID: "o", Rails: "short", State: state, UpdatedAt: t0}); err != nil {
		t.Fatal(err)
	}
	first.Close()

	second, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, err := second.Load(ctx, "life")
	if err != nil {
		t.Fatalf("snapshot must survive the restart: %v", err)
	}
	a, _ := state.Marshal()
	b, _ := got.State.Marshal()
	if string(a) != string(b) {
		t.Errorf("state after restart differs:\n%s\n%s", a, b)
	}
	// and it keeps playing from exactly there
	next := run(t, ShortRun, got.State, adv)
	if next.Beat != BeatNight || len(next.Bond) != 1 {
		t.Errorf("resumed state did not continue: %+v", next)
	}
}

func TestDBPurgeDropsStaleSessions(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	old := Row{ID: "old", DogID: "d", OrgID: "o", Rails: "short", State: NewState(ShortRun, t0), UpdatedAt: t0}
	fresh := old
	fresh.ID = "fresh"
	fresh.UpdatedAt = t0.Add(48 * time.Hour)
	db.Save(ctx, old)
	db.Save(ctx, fresh)
	n, err := db.Purge(ctx, t0.Add(24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("purge dropped %d (%v), want 1", n, err)
	}
	if _, err := db.Load(ctx, "old"); !errors.Is(err, ErrNotFound) {
		t.Error("stale session must be gone")
	}
	if _, err := db.Load(ctx, "fresh"); err != nil {
		t.Error("fresh session must stay")
	}
}

func TestRailsNameRoundTrip(t *testing.T) {
	for _, r := range []Rails{ShortRun, FullRun} {
		back, ok := RailsByName(r.Name())
		if !ok || back.Days != r.Days || len(back.Beats) != len(r.Beats) {
			t.Errorf("rails %q did not round trip", r.Name())
		}
	}
	if _, ok := RailsByName("dreams"); ok {
		t.Error("unknown rails must be a miss")
	}
}
