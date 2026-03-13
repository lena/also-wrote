package db

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection and provides app-specific helpers.
type DB struct {
	*sql.DB
}

// User represents a logged-in user (magic link auth).
type User struct {
	ID        int64
	Email     string
	CreatedAt time.Time
}

// Open connects to PostgreSQL using conn (e.g. from DATABASE_URL).
// For local dev use: postgres://localhost/also_wrote?sslmode=disable
func Open(conn string) (*DB, error) {
	if conn == "" {
		conn = "postgres://localhost/also_wrote?sslmode=disable"
	}
	sqlDB, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	// Users: magic-link auth by email
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         BIGSERIAL PRIMARY KEY,
			email      TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	// Magic link tokens: one-time use, short-lived
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS magic_link_tokens (
			token_hash TEXT PRIMARY KEY,
			email      TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	// User's hearted/favorite writers (TMDB person IDs)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_fave_writers (
			user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			person_id  INT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, person_id)
		);
		CREATE INDEX IF NOT EXISTS idx_user_fave_writers_user_id ON user_fave_writers(user_id);
	`)
	if err != nil {
		return err
	}

	return nil
}

// UserByID returns the user for the given id, or nil if not found.
func (db *DB) UserByID(id int64) (*User, error) {
	var u User
	err := db.QueryRow(
		`SELECT id, email, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UserByEmail returns the user for the given email, or nil if not found.
func (db *DB) UserByEmail(email string) (*User, error) {
	var u User
	err := db.QueryRow(
		`SELECT id, email, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a user by email and returns it. Errors if email already exists.
func (db *DB) CreateUser(email string) (*User, error) {
	var u User
	err := db.QueryRow(
		`INSERT INTO users (email) VALUES ($1) RETURNING id, email, created_at`,
		email,
	).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetOrCreateUser returns the user for email, creating one if needed.
func (db *DB) GetOrCreateUser(email string) (*User, error) {
	u, err := db.UserByEmail(email)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	return db.CreateUser(email)
}

// SaveMagicLinkToken stores a token hash for the given email and expiry.
func (db *DB) SaveMagicLinkToken(tokenHash, email string, expiresAt time.Time) error {
	_, err := db.Exec(
		`INSERT INTO magic_link_tokens (token_hash, email, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, email, expiresAt,
	)
	return err
}

// ConsumeMagicLinkToken finds a valid token, deletes it, and returns the email. Returns empty string if invalid/expired.
func (db *DB) ConsumeMagicLinkToken(tokenHash string) (email string, err error) {
	err = db.QueryRow(
		`DELETE FROM magic_link_tokens WHERE token_hash = $1 AND expires_at > NOW() RETURNING email`,
		tokenHash,
	).Scan(&email)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return email, err
}

// AddFavoriteWriter adds personID to the user's favorite writers. Idempotent.
func (db *DB) AddFavoriteWriter(userID int64, personID int) error {
	_, err := db.Exec(
		`INSERT INTO user_fave_writers (user_id, person_id) VALUES ($1, $2) ON CONFLICT (user_id, person_id) DO NOTHING`,
		userID, personID,
	)
	return err
}

// RemoveFavoriteWriter removes personID from the user's favorite writers.
func (db *DB) RemoveFavoriteWriter(userID int64, personID int) error {
	_, err := db.Exec(
		`DELETE FROM user_fave_writers WHERE user_id = $1 AND person_id = $2`,
		userID, personID,
	)
	return err
}

// FavoriteWriterPersonIDs returns the list of TMDB person IDs the user has favorited.
func (db *DB) FavoriteWriterPersonIDs(userID int64) ([]int, error) {
	rows, err := db.Query(
		`SELECT person_id FROM user_fave_writers WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IsFavoriteWriter returns whether the user has favorited the given person.
func (db *DB) IsFavoriteWriter(userID int64, personID int) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT 1 FROM user_fave_writers WHERE user_id = $1 AND person_id = $2`,
		userID, personID,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UserWithFavoriteCount is a user row with their favorite writers count (for admin list).
type UserWithFavoriteCount struct {
	ID                   int64
	Email                string
	FavoriteWritersCount int
}

// ListUsersWithFavoriteCount returns all users with their favorite writer count, ordered by user id.
func (db *DB) ListUsersWithFavoriteCount() ([]UserWithFavoriteCount, error) {
	rows, err := db.Query(`
		SELECT u.id, u.email, COUNT(f.person_id)::int AS favorite_writers_count
		FROM users u
		LEFT JOIN user_fave_writers f ON f.user_id = u.id
		GROUP BY u.id, u.email
		ORDER BY u.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []UserWithFavoriteCount
	for rows.Next() {
		var row UserWithFavoriteCount
		if err := rows.Scan(&row.ID, &row.Email, &row.FavoriteWritersCount); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}
