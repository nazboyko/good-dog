package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	// pure Go sqlite, no cgo, per the trap table
	_ "modernc.org/sqlite"
)

// DB is the durable truth for sessions. Every transition snapshots the
// state here, and a resume after a restart rebuilds from here. WAL mode
// so readers never block the writer, per the stack notes.
type DB struct {
	sql *sql.DB
}

// Row is one persisted session: enough to rebuild the Session without
// the model, plus the state itself.
type Row struct {
	ID        string
	DogID     string
	OrgID     string
	Rails     string
	State     State
	UpdatedAt time.Time
}

// ErrNotFound is the miss: an unknown, expired or purged id. Callers
// degrade to a clean new run, never an error screen.
var ErrNotFound = errors.New("session not found")

const schema = `
create table if not exists sessions (
	id         text primary key,
	dog_id     text not null,
	org_id     text not null,
	rails      text not null,
	state      text not null,
	updated_at text not null
);
create index if not exists sessions_updated on sessions(updated_at);
`

// OpenDB opens or creates the sqlite file. Tests use a temp file so
// the WAL path is what runs everywhere.
func OpenDB(path string) (*DB, error) {
	// WAL and a busy timeout so a burst of snapshots never errors
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sessions db: %w", err)
	}
	// modernc sqlite is safest with one connection per file for writes
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("sessions schema: %w", err)
	}
	return &DB{sql: db}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

// Save writes or replaces a session's snapshot.
func (d *DB) Save(ctx context.Context, row Row) error {
	raw, err := row.State.Marshal()
	if err != nil {
		return err
	}
	_, err = d.sql.ExecContext(ctx, `
		insert into sessions (id, dog_id, org_id, rails, state, updated_at)
		values (?, ?, ?, ?, ?, ?)
		on conflict(id) do update set state = excluded.state, updated_at = excluded.updated_at`,
		row.ID, row.DogID, row.OrgID, row.Rails, string(raw), stamp(row.UpdatedAt))
	return err
}

// stamp is a fixed width utc timestamp, so string order is time order
// and the purge cutoff compares correctly.
const stampLayout = "2006-01-02T15:04:05.000000000Z"

func stamp(t time.Time) string { return t.UTC().Format(stampLayout) }

// Load reads one session, ErrNotFound when it is not there.
func (d *DB) Load(ctx context.Context, id string) (Row, error) {
	var row Row
	var raw, updated string
	err := d.sql.QueryRowContext(ctx,
		`select id, dog_id, org_id, rails, state, updated_at from sessions where id = ?`, id,
	).Scan(&row.ID, &row.DogID, &row.OrgID, &row.Rails, &raw, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Row{}, ErrNotFound
	}
	if err != nil {
		return Row{}, err
	}
	state, err := UnmarshalState([]byte(raw))
	if err != nil {
		// a row this build cannot read is a miss, not a crash
		return Row{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	row.State = state
	row.UpdatedAt, _ = time.Parse(stampLayout, updated)
	return row, nil
}

// Purge deletes sessions untouched since the cutoff. A life left open
// for days is not resumed, the player gets a clean new one.
func (d *DB) Purge(ctx context.Context, before time.Time) (int64, error) {
	res, err := d.sql.ExecContext(ctx, `delete from sessions where updated_at < ?`, stamp(before))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RailsByName maps the persisted rails name back to the table. Unknown
// names are a miss so a resume never guesses.
func RailsByName(name string) (Rails, bool) {
	switch name {
	case "short":
		return ShortRun, true
	case "full":
		return FullRun, true
	}
	return Rails{}, false
}

// Name is the persisted identity of a rails table.
func (r Rails) Name() string {
	if r.Days == FullRun.Days && len(r.Beats) == len(FullRun.Beats) {
		return "full"
	}
	return "short"
}
