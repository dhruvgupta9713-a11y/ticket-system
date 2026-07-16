package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	password := "my-secure-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if hash == password {
		t.Fatal("expected hash to be different from plaintext password")
	}

	if !CheckPasswordHash(password, hash) {
		t.Fatal("expected password to match its hash")
	}

	if CheckPasswordHash("wrong-password", hash) {
		t.Fatal("expected comparison to fail for incorrect password")
	}
}

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key"
	userID := int64(42)
	username := "testuser"

	token, err := GenerateToken(userID, username, secret, 5*time.Second)
	if err != nil {
		t.Fatalf("expected no error generating token, got %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("expected token to validate successfully, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, claims.UserID)
	}

	if claims.Username != username {
		t.Errorf("expected username %s, got %s", username, claims.Username)
	}

	// Validate validation fails with wrong secret
	_, err = ValidateToken(token, "wrong-secret-key")
	if err == nil {
		t.Fatal("expected token validation to fail with wrong secret")
	}
}
