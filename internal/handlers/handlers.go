package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"ticket-system/internal/auth"
	"ticket-system/internal/middleware"
	"ticket-system/internal/models"
	"ticket-system/internal/store"
)

type Handlers struct {
	store     *store.Store
	jwtSecret string
}

func NewHandlers(store *store.Store, jwtSecret string) *Handlers {
	return &Handlers{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

// Health checks the health of the service.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	h.respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Register handles user registration.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.respondWithError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Email, hashedPassword)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateUsernameOrEmail) {
			h.respondWithError(w, http.StatusConflict, "Username or email already exists")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, user)
}

// Login authenticates a user and returns a JWT.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	identifier := strings.TrimSpace(req.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	password := strings.TrimSpace(req.Password)

	if identifier == "" || password == "" {
		h.respondWithError(w, http.StatusBadRequest, "Username/Email and password are required")
		return
	}

	user, err := h.store.GetUserByUsernameOrEmail(identifier)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Login error")
		return
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Token expires in 24 hours
	token, err := auth.GenerateToken(user.ID, user.Username, h.jwtSecret, 24*time.Hour)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"token": token,
		"jwt":   token, // Support both token and jwt keys in response to be safe
	})
}

// CreateTicket creates a new ticket for the logged-in user.
func (h *Handlers) CreateTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" || req.Description == "" {
		h.respondWithError(w, http.StatusBadRequest, "Title and description are required")
		return
	}

	ticket, err := h.store.CreateTicket(req.Title, req.Description, userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create ticket")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, ticket)
}

// ListTickets returns all tickets belonging to the logged-in user.
func (h *Handlers) ListTickets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tickets, err := h.store.GetTicketsByOwner(userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to fetch tickets")
		return
	}

	h.respondWithJSON(w, http.StatusOK, tickets)
}

// GetTicketByID returns a single ticket by ID after ownership checks.
func (h *Handlers) GetTicketByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	ticketID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid ticket ID format")
		return
	}

	ticket, err := h.store.GetTicketByID(ticketID)
	if err != nil {
		if errors.Is(err, store.ErrTicketNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Ticket not found")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}

	if ticket.OwnerID != userID {
		h.respondWithError(w, http.StatusForbidden, "You do not own this ticket")
		return
	}

	h.respondWithJSON(w, http.StatusOK, ticket)
}

// UpdateTicketStatus updates the ticket status enforcing state transition rules.
func (h *Handlers) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	ticketID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid ticket ID format")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	nextStatus := strings.ToLower(strings.TrimSpace(req.Status))
	if nextStatus != "open" && nextStatus != "in_progress" && nextStatus != "closed" {
		h.respondWithError(w, http.StatusBadRequest, "Invalid status. Must be 'open', 'in_progress', or 'closed'")
		return
	}

	ticket, err := h.store.GetTicketByID(ticketID)
	if err != nil {
		if errors.Is(err, store.ErrTicketNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Ticket not found")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}

	if ticket.OwnerID != userID {
		h.respondWithError(w, http.StatusForbidden, "You do not own this ticket")
		return
	}

	if !isValidTransition(ticket.Status, nextStatus) {
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid status transition from '%s' to '%s'", ticket.Status, nextStatus))
		return
	}

	// No-op if transition is to the same status
	if ticket.Status == nextStatus {
		h.respondWithJSON(w, http.StatusOK, ticket)
		return
	}

	updatedTicket, err := h.store.UpdateTicketStatus(ticketID, nextStatus)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update ticket status")
		return
	}

	h.respondWithJSON(w, http.StatusOK, updatedTicket)
}

func isValidTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "open":
		return next == "in_progress"
	case "in_progress":
		return next == "closed"
	case "closed":
		return false
	default:
		return false
	}
}

func (h *Handlers) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	h.respondWithJSON(w, statusCode, map[string]string{"error": message})
}

func (h *Handlers) respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
