package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPassword(t *testing.T) {
	password := "password123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Expected hash to not be empty")
	}

	if hash == password {
		t.Fatal("Expected hash to be different from password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "password123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Fatal("Expected password check to succeed")
	}

	if CheckPasswordHash("wrongpassword", hash) {
		t.Fatal("Expected password check to fail")
	}
}

func TestGenerateAndValidateJWT(t *testing.T) {
	userID := 1
	jwtSecret := "test-secret"

	tokenString, err := GenerateJWT(userID, jwtSecret)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if tokenString == "" {
		t.Fatal("Expected token to not be empty")
	}

	token, err := ValidateJWT(tokenString, jwtSecret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if !token.Valid {
		t.Fatal("Expected token to be valid")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.UserID != userID {
			t.Fatalf("Expected user ID to be %d, got %d", userID, claims.UserID)
		}
	} else {
		t.Fatal("Failed to parse claims")
	}
}

func TestExpiredJWT(t *testing.T) {
	jwtSecret := "test-secret"
	expirationTime := time.Now().Add(-1 * time.Hour)
	claims := &Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	_, err = ValidateJWT(tokenString, jwtSecret)
	if err == nil {
		t.Fatal("Expected token validation to fail for expired token")
	}
}
