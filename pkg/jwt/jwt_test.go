package jwt

import (
	"errors"
	"testing"
	"time"

	"chatapp/internal/model"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "a-test-secret-with-at-least-32-characters"

func TestGenerateAndValidateAccessToken(t *testing.T) {
	user := &model.User{ID: uuid.New(), FullName: "Ada Lovelace"}

	raw, err := GenerateToken(user, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ValidateToken(raw, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != user.ID.String() || claims.Username != user.FullName || claims.TokenType != AccessToken {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Fatal("access token must have a future expiry")
	}
}

func TestRefreshToken(t *testing.T) {
	userID := uuid.New().String()
	raw, err := GenerateRefreshToken(userID, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	access, err := RefreshToken(raw, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	claims, err := ValidateToken(access, testSecret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != userID || claims.TokenType != AccessToken {
		t.Fatalf("unexpected refreshed claims: %+v", claims)
	}
}

func TestRefreshTokenRejectsAccessToken(t *testing.T) {
	raw, err := GenerateToken(&model.User{ID: uuid.New()}, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := RefreshToken(raw, testSecret, time.Hour); !errors.Is(err, ErrInvalidTokenType) {
		t.Fatalf("RefreshToken() error = %v, want ErrInvalidTokenType", err)
	}
}

func TestValidateTokenRejectsWrongSecretAndExpiredToken(t *testing.T) {
	raw, err := GenerateToken(&model.User{ID: uuid.New()}, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := ValidateToken(raw, "another-secret-with-at-least-32-characters"); err == nil {
		t.Fatal("ValidateToken() accepted a token signed with another secret")
	}

	expired := Claims{UserID: uuid.New().String(), TokenType: AccessToken, RegisteredClaims: jwtlib.RegisteredClaims{
		ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(-time.Hour)),
		IssuedAt:  jwtlib.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}}
	expiredRaw, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, expired).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, err := ValidateToken(expiredRaw, testSecret); err == nil {
		t.Fatal("ValidateToken() accepted an expired token")
	}
}
