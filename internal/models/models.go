package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Ticket struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	OwnerID     int64     `json:"owner_id"`
	UserID      int64     `json:"user_id"` // Included to satisfy both owner_id and user_id requirements
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}
