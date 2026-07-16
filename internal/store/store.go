package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"ticket-system/internal/models"
)

var (
	ErrDuplicateUsernameOrEmail = errors.New("username or email already exists")
	ErrUserNotFound             = errors.New("user not found")
	ErrTicketNotFound           = errors.New("ticket not found")
)

type Store struct {
	db *sql.DB
}

// NewStore initializes the SQLite database connection, enables foreign keys, and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign key support
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('open', 'in_progress', 'closed')),
		owner_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(owner_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func parseTime(val interface{}) (time.Time, error) {
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case string:
		layouts := []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.999999999-07:00",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse time string: %s", v)
	default:
		return time.Time{}, fmt.Errorf("unexpected time type: %T", val)
	}
}

// CreateUser inserts a new user record. Returns ErrDuplicateUsernameOrEmail if constraint is violated.
func (s *Store) CreateUser(username, email, passwordHash string) (*models.User, error) {
	query := `INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`
	res, err := s.db.Exec(query, username, email, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrDuplicateUsernameOrEmail
		}
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(id)
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(id int64) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var u models.User
	var createdAtVal interface{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &createdAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt, err = parseTime(createdAtVal)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetUserByUsernameOrEmail retrieves a user matching username or email.
func (s *Store) GetUserByUsernameOrEmail(identifier string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE username = ? OR email = ?`
	row := s.db.QueryRow(query, identifier, identifier)

	var u models.User
	var createdAtVal interface{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &createdAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt, err = parseTime(createdAtVal)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// CreateTicket inserts a new ticket.
func (s *Store) CreateTicket(title, description string, ownerID int64) (*models.Ticket, error) {
	query := `INSERT INTO tickets (title, description, status, owner_id) VALUES (?, ?, 'open', ?)`
	res, err := s.db.Exec(query, title, description, ownerID)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetTicketByID(id)
}

// GetTicketByID retrieves a ticket by ID.
func (s *Store) GetTicketByID(id int64) (*models.Ticket, error) {
	query := `SELECT id, title, description, status, owner_id, created_at, updated_at FROM tickets WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var t models.Ticket
	var createdAtVal, updatedAtVal interface{}
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.OwnerID, &createdAtVal, &updatedAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}

	t.UserID = t.OwnerID

	t.CreatedAt, err = parseTime(createdAtVal)
	if err != nil {
		return nil, err
	}

	t.UpdatedAt, err = parseTime(updatedAtVal)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// GetTicketsByOwner retrieves all tickets belonging to a specific owner.
func (s *Store) GetTicketsByOwner(ownerID int64) ([]models.Ticket, error) {
	query := `SELECT id, title, description, status, owner_id, created_at, updated_at FROM tickets WHERE owner_id = ? ORDER BY created_at DESC`
	rows, err := s.db.Query(query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := []models.Ticket{} // ensure empty slice is returned rather than nil to produce [] in JSON
	for rows.Next() {
		var t models.Ticket
		var createdAtVal, updatedAtVal interface{}
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.OwnerID, &createdAtVal, &updatedAtVal)
		if err != nil {
			return nil, err
		}

		t.UserID = t.OwnerID

		t.CreatedAt, err = parseTime(createdAtVal)
		if err != nil {
			return nil, err
		}

		t.UpdatedAt, err = parseTime(updatedAtVal)
		if err != nil {
			return nil, err
		}

		tickets = append(tickets, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

// UpdateTicketStatus updates the status of a ticket and updates its updated_at column.
func (s *Store) UpdateTicketStatus(id int64, status string) (*models.Ticket, error) {
	query := `UPDATE tickets SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.Exec(query, status, id)
	if err != nil {
		return nil, err
	}

	return s.GetTicketByID(id)
}
