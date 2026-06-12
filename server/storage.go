package server

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DefaultTTL = 72 * time.Hour
	MaxTTL     = 168 * time.Hour
)

type Drop struct {
	ID        string
	Hint      string
	Payload   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Store struct{ db *sql.DB }

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.Init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) DB() *sql.DB     { return s.db }
func (s *Store) Close() error    { return s.db.Close() }

func (s *Store) Init(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`CREATE TABLE IF NOT EXISTS drops (
			id TEXT PRIMARY KEY,
			hint TEXT NOT NULL,
			payload BLOB NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_drops_hint ON drops(hint);`,
		`CREATE INDEX IF NOT EXISTS idx_drops_expires_at ON drops(expires_at);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateDrop(ctx context.Context, hint, payload string, ttl time.Duration) (Drop, error) {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		ttl = MaxTTL
	}
	now := time.Now().UTC().Truncate(time.Second)
	d := Drop{ID: uuid.NewString(), Hint: hint, Payload: payload, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	_, err := s.db.ExecContext(ctx, `INSERT INTO drops(id,hint,payload,created_at,expires_at) VALUES(?,?,?,?,?)`, d.ID, d.Hint, d.Payload, d.CreatedAt.Unix(), d.ExpiresAt.Unix())
	return d, err
}

func (s *Store) ListDrops(ctx context.Context, hint string, now time.Time) ([]Drop, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,hint,payload,created_at,expires_at FROM drops WHERE hint=? AND expires_at>=? ORDER BY created_at ASC`, hint, now.UTC().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Drop
	for rows.Next() {
		var d Drop
		var ca, ea int64
		if err := rows.Scan(&d.ID, &d.Hint, &d.Payload, &ca, &ea); err != nil {
			return nil, err
		}
		d.CreatedAt, d.ExpiresAt = time.Unix(ca, 0).UTC(), time.Unix(ea, 0).UTC()
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DeleteDrop(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("empty id")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM drops WHERE id=?`, id)
	return err
}

func (s *Store) ExpireDrops(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM drops WHERE expires_at < ?`, now.UTC().Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) Vacuum(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE);`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `VACUUM;`)
	return err
}
