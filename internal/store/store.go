// Package store persists what must survive a relaunch: settings, guests,
// invitations, and the queue. SQLite via modernc.org/sqlite (pure Go).
//
// Nothing that grants access is stored in the clear: guest cookie IDs and
// invitation codes are kept as SHA-256 hashes. They are high-entropy random
// tokens, so a fast hash is the right tool (bcrypt exists to slow down
// guessing of low-entropy passwords). The Spotify token is not in this file;
// see config.SaveToken.
package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS guests (
	id         TEXT PRIMARY KEY,           -- sha256(cookie id)
	name       TEXT NOT NULL DEFAULT '',
	blocked    INTEGER NOT NULL DEFAULT 0,
	admitted   INTEGER NOT NULL DEFAULT 0, -- passed an invitation (or admin admitted)
	first_seen TEXT NOT NULL,
	last_seen  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS invitations (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	code_hash  TEXT NOT NULL UNIQUE,        -- sha256(code)
	label      TEXT NOT NULL DEFAULT '',   -- last 4 chars, for the admin list
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	revoked    INTEGER NOT NULL DEFAULT 0,
	uses       INTEGER NOT NULL DEFAULT 0,
	used_by    TEXT NOT NULL DEFAULT ''     -- guest id that redeemed it; a link works once
);
CREATE TABLE IF NOT EXISTS local_index (
	path    TEXT PRIMARY KEY,
	size    INTEGER NOT NULL,
	mtime   TEXT NOT NULL,
	data    TEXT NOT NULL,                   -- local.indexed JSON (tags, duration, artwork flag)
	seen_at TEXT NOT NULL                    -- last scan that found the file
);
CREATE TABLE IF NOT EXISTS plays (
	source      TEXT NOT NULL,
	track_id    TEXT NOT NULL,
	track       TEXT NOT NULL,               -- latest source.Track JSON
	count       INTEGER NOT NULL DEFAULT 0,
	last_played TEXT NOT NULL,
	PRIMARY KEY (source, track_id)
);
CREATE TABLE IF NOT EXISTS queue (
	position          INTEGER PRIMARY KEY,
	id                TEXT NOT NULL,
	track             TEXT NOT NULL,        -- source.Track as JSON
	requested_by      TEXT NOT NULL,        -- guest id (hash)
	requested_by_name TEXT NOT NULL,
	requested_at      TEXT NOT NULL,
	rank              INTEGER NOT NULL DEFAULT 0,
	votes             TEXT NOT NULL          -- JSON array of guest ids
);
`

type Store struct{ db *sql.DB }

// Open creates the file (0600) and schema if needed.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // sqlite: one writer, and we are tiny
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: schema: %w", err)
	}
	// migration for databases created before invitations became single-use
	if _, err := db.Exec(`ALTER TABLE invitations ADD COLUMN used_by TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column") {
		db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}
	_ = os.Chmod(path, 0o600)
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Hash is how cookie IDs and invitation codes are keyed.
func Hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// --- settings (JSON blobs by key; config uses one key) ---

var ErrNotFound = errors.New("store: not found")

func (s *Store) Get(key string) ([]byte, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return []byte(v), err
}

func (s *Store) Set(key string, v []byte) error {
	_, err := s.db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, string(v))
	return err
}

// --- guests ---

type Guest struct {
	ID        string    `json:"id"` // hash
	Name      string    `json:"name"`
	Blocked   bool      `json:"blocked"`
	Admitted  bool      `json:"admitted"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// Seen records a visit, creating the row on first sight. Returns the guest.
func (s *Store) Seen(id string, at time.Time) (Guest, error) {
	ts := at.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`INSERT INTO guests(id, first_seen, last_seen) VALUES(?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_seen = excluded.last_seen`, id, ts, ts); err != nil {
		return Guest{}, err
	}
	return s.guest(id)
}

func (s *Store) guest(id string) (Guest, error) {
	row := s.db.QueryRow(`SELECT id, name, blocked, admitted, first_seen, last_seen FROM guests WHERE id = ?`, id)
	return scanGuest(row)
}

type scanner interface{ Scan(dest ...any) error }

func scanGuest(r scanner) (Guest, error) {
	var g Guest
	var first, last string
	if err := r.Scan(&g.ID, &g.Name, &g.Blocked, &g.Admitted, &first, &last); err != nil {
		return Guest{}, err
	}
	g.FirstSeen, _ = time.Parse(time.RFC3339Nano, first)
	g.LastSeen, _ = time.Parse(time.RFC3339Nano, last)
	return g, nil
}

func (s *Store) Guests() ([]Guest, error) {
	rows, err := s.db.Query(`SELECT id, name, blocked, admitted, first_seen, last_seen FROM guests ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) SetGuestName(id, name string) error {
	_, err := s.db.Exec(`UPDATE guests SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *Store) SetGuestBlocked(id string, blocked bool) error {
	// blocking an unseen id still needs a row so the block survives
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`INSERT INTO guests(id, blocked, first_seen, last_seen) VALUES(?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET blocked = excluded.blocked`, id, blocked, now, now)
	return err
}

func (s *Store) SetGuestAdmitted(id string, admitted bool) error {
	_, err := s.db.Exec(`UPDATE guests SET admitted = ? WHERE id = ?`, admitted, id)
	return err
}

// AdmitAll marks every known guest admitted (used when invite-only is switched on).
func (s *Store) AdmitAll() error {
	_, err := s.db.Exec(`UPDATE guests SET admitted = 1`)
	return err
}

func (s *Store) DeleteGuest(id string) error {
	_, err := s.db.Exec(`DELETE FROM guests WHERE id = ?`, id)
	return err
}

// --- invitations ---

type Invitation struct {
	ID        int64     `json:"id"`
	Label     string    `json:"label"` // last 4 chars of the code
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Revoked   bool      `json:"revoked"`
	Uses      int       `json:"uses"`
}

func (s *Store) CreateInvitation(code string, expires time.Time) (Invitation, error) {
	label := code
	if len(label) > 4 {
		label = label[len(label)-4:]
	}
	now := time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO invitations(code_hash, label, created_at, expires_at) VALUES(?, ?, ?, ?)`,
		Hash(code), label, now.Format(time.RFC3339Nano), expires.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Invitation{}, err
	}
	id, _ := res.LastInsertId()
	return Invitation{ID: id, Label: label, CreatedAt: now, ExpiresAt: expires.UTC()}, nil
}

func (s *Store) Invitations() ([]Invitation, error) {
	rows, err := s.db.Query(`SELECT id, label, created_at, expires_at, revoked, uses FROM invitations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invitation
	for rows.Next() {
		var in Invitation
		var c, e string
		if err := rows.Scan(&in.ID, &in.Label, &c, &e, &in.Revoked, &in.Uses); err != nil {
			return nil, err
		}
		in.CreatedAt, _ = time.Parse(time.RFC3339Nano, c)
		in.ExpiresAt, _ = time.Parse(time.RFC3339Nano, e)
		out = append(out, in)
	}
	return out, rows.Err()
}

func (s *Store) RevokeInvitation(id int64) error {
	_, err := s.db.Exec(`UPDATE invitations SET revoked = 1 WHERE id = ?`, id)
	return err
}

// ErrInvalidInvitation covers unknown, expired, revoked, and used codes alike.
var ErrInvalidInvitation = errors.New("invalid invitation")

// RedeemInvitation validates the code at `now` and marks it used by guestID.
// A link works once; the guest who used it may open it again (idempotent),
// anyone else is refused.
func (s *Store) RedeemInvitation(code, guestID string, now time.Time) error {
	res, err := s.db.Exec(`UPDATE invitations SET uses = 1, used_by = ?
		WHERE code_hash = ? AND revoked = 0 AND expires_at > ? AND (used_by = '' OR used_by = ?)`,
		guestID, Hash(code), now.UTC().Format(time.RFC3339Nano), guestID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrInvalidInvitation
	}
	return nil
}

// --- queue ---

type QueueRow struct {
	ID              string          `json:"id"`
	Track           json.RawMessage `json:"track"`
	RequestedBy     string          `json:"requestedBy"`
	RequestedByName string          `json:"requestedByName"`
	RequestedAt     time.Time       `json:"requestedAt"`
	Rank            int             `json:"rank"`
	Votes           []string        `json:"votes"`
}

// SaveQueue replaces the stored queue with items, in order.
func (s *Store) SaveQueue(items []QueueRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM queue`); err != nil {
		return err
	}
	for i, it := range items {
		votes, _ := json.Marshal(it.Votes)
		if _, err := tx.Exec(`INSERT INTO queue(position, id, track, requested_by, requested_by_name, requested_at, rank, votes)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			i, it.ID, string(it.Track), it.RequestedBy, it.RequestedByName, it.RequestedAt.UTC().Format(time.RFC3339Nano), it.Rank, string(votes)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) LoadQueue() ([]QueueRow, error) {
	rows, err := s.db.Query(`SELECT id, track, requested_by, requested_by_name, requested_at, rank, votes FROM queue ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QueueRow
	for rows.Next() {
		var it QueueRow
		var track, at, votes string
		if err := rows.Scan(&it.ID, &track, &it.RequestedBy, &it.RequestedByName, &at, &it.Rank, &votes); err != nil {
			return nil, err
		}
		it.Track = json.RawMessage(track)
		it.RequestedAt, _ = time.Parse(time.RFC3339Nano, at)
		_ = json.Unmarshal([]byte(votes), &it.Votes)
		out = append(out, it)
	}
	return out, rows.Err()
}

// --- play history ---

// RecordPlay counts one play of a track on a source, keeping the latest metadata.
func (s *Store) RecordPlay(sourceID, trackID string, track json.RawMessage, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO plays(source, track_id, track, count, last_played) VALUES(?, ?, ?, 1, ?)
		ON CONFLICT(source, track_id) DO UPDATE SET count = count + 1, track = excluded.track, last_played = excluded.last_played`,
		sourceID, trackID, string(track), at.UTC().Format(time.RFC3339Nano))
	return err
}

// ClearPlays forgets the play history (all sources).
func (s *Store) ClearPlays() error {
	_, err := s.db.Exec(`DELETE FROM plays`)
	return err
}

// TopTracks returns the most played tracks of a source, ties broken by recency.
func (s *Store) TopTracks(sourceID string, limit int) ([]json.RawMessage, error) {
	rows, err := s.db.Query(`SELECT track FROM plays WHERE source = ? ORDER BY count DESC, last_played DESC LIMIT ?`, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(t))
	}
	return out, rows.Err()
}

// --- local file index cache (see local.Cache) ---

// LocalIndexGet returns the cached record if path is unchanged, marking it seen.
func (s *Store) LocalIndexGet(path string, size int64, mtime time.Time) ([]byte, bool) {
	var data string
	err := s.db.QueryRow(`SELECT data FROM local_index WHERE path = ? AND size = ? AND mtime = ?`,
		path, size, mtime.UTC().Format(time.RFC3339Nano)).Scan(&data)
	if err != nil {
		return nil, false
	}
	_, _ = s.db.Exec(`UPDATE local_index SET seen_at = ? WHERE path = ?`, time.Now().UTC().Format(time.RFC3339Nano), path)
	return []byte(data), true
}

// LocalIndexPut stores the record for path at this size and mtime.
func (s *Store) LocalIndexPut(path string, size int64, mtime time.Time, data []byte) {
	_, _ = s.db.Exec(`INSERT INTO local_index(path, size, mtime, data, seen_at) VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET size = excluded.size, mtime = excluded.mtime, data = excluded.data, seen_at = excluded.seen_at`,
		path, size, mtime.UTC().Format(time.RFC3339Nano), string(data), time.Now().UTC().Format(time.RFC3339Nano))
}

// LocalIndexPrune forgets files no scan has seen since before.
func (s *Store) LocalIndexPrune(before time.Time) {
	_, _ = s.db.Exec(`DELETE FROM local_index WHERE seen_at < ?`, before.UTC().Format(time.RFC3339Nano))
}
