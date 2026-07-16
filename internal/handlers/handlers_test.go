package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"ticket-system/internal/middleware"
	"ticket-system/internal/models"
	"ticket-system/internal/store"
)

func setupTestRouter(t *testing.T) (http.Handler, *store.Store, string) {
	jwtSecret := "testsecretjwt"
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	h := NewHandlers(s, jwtSecret)

	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)

	r.Route("/tickets", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/", h.CreateTicket)
		r.Get("/", h.ListTickets)
		r.Get("/{id}", h.GetTicketByID)
		r.Patch("/{id}/status", h.UpdateTicketStatus)
	})

	return r, s, jwtSecret
}

func TestHealthEndpoint(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

func TestAuthFlowAndTicketAccess(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// 1. Register User 1
	regPayload1 := models.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(regPayload1)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected registration to succeed (201), got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Register Duplicate User 1 -> should fail with 409
	req, _ = http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate registration, got %d", rr.Code)
	}

	// 3. Register User 2
	regPayload2 := models.RegisterRequest{
		Username: "charlie",
		Email:    "charlie@example.com",
		Password: "password456",
	}
	body, _ = json.Marshal(regPayload2)
	req, _ = http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected registration 2 to succeed (201), got %d", rr.Code)
	}

	// 4. Login User 1 -> Get Token
	loginPayload1 := models.LoginRequest{
		Username: "bob",
		Password: "password123",
	}
	body, _ = json.Marshal(loginPayload1)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected login to succeed (200), got %d", rr.Code)
	}

	var loginResp1 map[string]string
	json.NewDecoder(rr.Body).Decode(&loginResp1)
	token1 := loginResp1["token"]

	// 5. Login User 2 -> Get Token
	loginPayload2 := models.LoginRequest{
		Username: "charlie",
		Password: "password456",
	}
	body, _ = json.Marshal(loginPayload2)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var loginResp2 map[string]string
	json.NewDecoder(rr.Body).Decode(&loginResp2)
	token2 := loginResp2["token"]

	// 6. Login with invalid credentials -> should fail with 401
	loginPayloadErr := models.LoginRequest{
		Username: "bob",
		Password: "wrongpassword",
	}
	body, _ = json.Marshal(loginPayloadErr)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad login credentials, got %d", rr.Code)
	}

	// 7. Create ticket as User 1
	ticketPayload1 := models.CreateTicketRequest{
		Title:       "Setup dev server",
		Description: "Configure server on AWS instance",
	}
	body, _ = json.Marshal(ticketPayload1)
	req, _ = http.NewRequest("POST", "/tickets", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected ticket creation to succeed (201), got %d: %s", rr.Code, rr.Body.String())
	}

	var ticket1 models.Ticket
	json.NewDecoder(rr.Body).Decode(&ticket1)
	if ticket1.Status != "open" {
		t.Errorf("expected initial status to be 'open', got '%s'", ticket1.Status)
	}

	// Create a second ticket for User 1
	ticketPayload2 := models.CreateTicketRequest{
		Title:       "Write API docs",
		Description: "Draft Swagger definition",
	}
	body, _ = json.Marshal(ticketPayload2)
	req, _ = http.NewRequest("POST", "/tickets", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var ticket2 models.Ticket
	json.NewDecoder(rr.Body).Decode(&ticket2)

	// 8. List tickets as User 1 -> should return 2 tickets
	req, _ = http.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected list tickets success (200), got %d", rr.Code)
	}

	var ticketsList1 []models.Ticket
	json.NewDecoder(rr.Body).Decode(&ticketsList1)
	if len(ticketsList1) != 2 {
		t.Errorf("expected Bob to have 2 tickets, got %d", len(ticketsList1))
	}

	// 9. List tickets as User 2 -> should return empty array
	req, _ = http.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var ticketsList2 []models.Ticket
	json.NewDecoder(rr.Body).Decode(&ticketsList2)
	if len(ticketsList2) != 0 {
		t.Errorf("expected Charlie to have 0 tickets, got %d", len(ticketsList2))
	}

	// 10. Access own ticket (User 1 access Ticket 1) -> 200 OK
	req, _ = http.NewRequest("GET", fmt.Sprintf("/tickets/%d", ticket1.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected user to access own ticket (200), got %d", rr.Code)
	}

	// 11. Access another user's ticket (User 2 access Ticket 1) -> 403 Forbidden
	req, _ = http.NewRequest("GET", fmt.Sprintf("/tickets/%d", ticket1.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected access to other user's ticket to return 403 Forbidden, got %d", rr.Code)
	}

	// 12. Non-existent ticket -> 404
	req, _ = http.NewRequest("GET", "/tickets/9999", nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent ticket, got %d", rr.Code)
	}
}

func TestTicketStatusTransitions(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Register & Login Bob
	regPayload := models.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	loginPayload := models.LoginRequest{
		Username: "bob",
		Password: "password123",
	}
	body, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var loginResp map[string]string
	json.NewDecoder(rr.Body).Decode(&loginResp)
	token := loginResp["token"]

	// Create Ticket (default 'open')
	ticketPayload := models.CreateTicketRequest{
		Title:       "Test transitions",
		Description: "Description",
	}
	body, _ = json.Marshal(ticketPayload)
	req, _ = http.NewRequest("POST", "/tickets", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var ticket models.Ticket
	json.NewDecoder(rr.Body).Decode(&ticket)

	// Try invalid transition directly open -> closed (skipped transition) -> should fail 400
	updatePayload := models.UpdateStatusRequest{Status: "closed"}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/tickets/%d/status", ticket.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected open -> closed (skipped) to fail with 400, got %d", rr.Code)
	}

	// Try valid transition open -> in_progress -> should succeed 200
	updatePayload = models.UpdateStatusRequest{Status: "in_progress"}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/tickets/%d/status", ticket.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected open -> in_progress to succeed (200), got %d", rr.Code)
	}

	// Try transition back in_progress -> open -> should fail 400 (reopening)
	updatePayload = models.UpdateStatusRequest{Status: "open"}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/tickets/%d/status", ticket.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected in_progress -> open to fail with 400, got %d", rr.Code)
	}

	// Try valid transition in_progress -> closed -> should succeed 200
	updatePayload = models.UpdateStatusRequest{Status: "closed"}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/tickets/%d/status", ticket.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected in_progress -> closed to succeed (200), got %d", rr.Code)
	}

	// Try reopening closed ticket closed -> in_progress -> should fail 400
	updatePayload = models.UpdateStatusRequest{Status: "in_progress"}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/tickets/%d/status", ticket.ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected closed -> in_progress to fail with 400, got %d", rr.Code)
	}
}
