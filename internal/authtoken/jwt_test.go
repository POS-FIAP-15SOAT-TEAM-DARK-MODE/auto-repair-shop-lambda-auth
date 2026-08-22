package authtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken_ClaimsRoundtrip(t *testing.T) {
	// Arrange
	secret := []byte("test-secret")
	expiresAt := time.Now().Add(time.Hour)

	// Act
	tokenStr, err := GenerateToken(secret, "user-123", []string{"CUSTOMER"}, expiresAt)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims := &UserClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})

	// Assert
	if err != nil || !parsed.Valid {
		t.Fatalf("token did not parse/validate: %v", err)
	}
	if claims.UserId != "user-123" {
		t.Errorf("UserId = %q, want %q", claims.UserId, "user-123")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "CUSTOMER" {
		t.Errorf("Roles = %v, want [CUSTOMER]", claims.Roles)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(expiresAt.Truncate(time.Second)) {
		t.Errorf("ExpiresAt = %v, want ~%v", claims.ExpiresAt, expiresAt)
	}
}

func TestGenerateToken_WrongSecretFailsValidation(t *testing.T) {
	// Arrange
	tokenStr, err := GenerateToken([]byte("secret-a"), "user-123", []string{"CUSTOMER"}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	// Act
	claims := &UserClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return []byte("secret-b"), nil
	})

	// Assert
	if err == nil {
		t.Fatal("expected validation error with mismatched secret, got nil")
	}
}

func TestGenerateToken_UsesHS256(t *testing.T) {
	// Arrange
	tokenStr, err := GenerateToken([]byte("secret"), "user-123", []string{"CUSTOMER"}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	// Act
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenStr, &UserClaims{})
	if err != nil {
		t.Fatalf("ParseUnverified error: %v", err)
	}

	// Assert — must match the app's middleware expectation
	if token.Method.Alg() != "HS256" {
		t.Errorf("signing alg = %q, want HS256", token.Method.Alg())
	}
}
