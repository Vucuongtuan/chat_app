package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"chatapp/internal/model"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	AccessToken       = "access"
	RefreshTokenType  = "refresh"
	TempTwoFactorType = "temp_2fa"
)

var (
	ErrEmptySecret       = errors.New("jwt secret must not be empty")
	ErrInvalidExpiration = errors.New("jwt expiration must be greater than zero")
	ErrInvalidTokenType  = errors.New("invalid jwt token type")
)

type Claims struct {
	UserID    string `json:"user_id"`
	AccountID string `json:"account_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	IsPrimary bool   `json:"is_primary,omitempty"`
	TokenType string `json:"token_type"`
	jwtlib.RegisteredClaims
}

func GenerateSessionToken(user *model.User, accountID, sessionID, deviceID string, isPrimary bool, secret string, expiration time.Duration) (string, error) {
	if user == nil {
		return "", errors.New("user must not be nil")
	}

	return sign(Claims{
		UserID:    user.ID.String(),
		AccountID: accountID,
		Username:  user.FullName,
		SessionID: sessionID,
		DeviceID:  deviceID,
		IsPrimary: isPrimary,
		TokenType: AccessToken,
	}, secret, expiration)
}

func GenerateSessionRefreshToken(user *model.User, accountID, sessionID, deviceID string, isPrimary bool, secret string, expiration time.Duration) (string, error) {
	if user == nil {
		return "", errors.New("user must not be nil")
	}

	return sign(Claims{
		UserID:    user.ID.String(),
		AccountID: accountID,
		Username:  user.FullName,
		SessionID: sessionID,
		DeviceID:  deviceID,
		IsPrimary: isPrimary,
		TokenType: RefreshTokenType,
	}, secret, expiration)
}

func GenerateTempTwoFactorToken(accountID, email string, secret string, expiration time.Duration) (string, error) {
	if strings.TrimSpace(accountID) == "" {
		return "", errors.New("account ID must not be empty")
	}

	return sign(Claims{
		UserID:    accountID,
		AccountID: accountID,
		Email:     email,
		TokenType: TempTwoFactorType,
	}, secret, expiration)
}

func GenerateToken(user *model.User, secret string, expiration time.Duration) (string, error) {
	if user == nil {
		return "", errors.New("user must not be nil")
	}

	return sign(Claims{
		UserID:    user.ID.String(),
		Username:  user.FullName,
		TokenType: AccessToken,
	}, secret, expiration)
}

func GenerateRefreshToken(userID string, secret string, expiration time.Duration) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", errors.New("user ID must not be empty")
	}

	return sign(Claims{
		UserID:    userID,
		TokenType: RefreshTokenType,
	}, secret, expiration)
}

func ValidateToken(tokenString, secret string) (*Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptySecret
	}

	claims := new(Claims)
	//
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(token *jwtlib.Token) (any, error) {
		if token.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
		}
		return []byte(secret), nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, errors.New("invalid jwt token")
	}
	if strings.TrimSpace(claims.UserID) == "" {
		return nil, errors.New("jwt user ID is required")
	}
	if claims.TokenType != AccessToken && claims.TokenType != RefreshTokenType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}

func ValidateTempTwoFactorToken(tokenString, secret string) (*Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptySecret
	}

	claims := new(Claims)
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(token *jwtlib.Token) (any, error) {
		if token.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
		}
		return []byte(secret), nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, errors.New("invalid jwt token")
	}
	if strings.TrimSpace(claims.AccountID) == "" && strings.TrimSpace(claims.UserID) == "" {
		return nil, errors.New("jwt user/account ID is required")
	}
	if claims.TokenType != TempTwoFactorType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}

func RefreshToken(refreshToken, secret string, newExpiration time.Duration) (string, error) {
	claims, err := ValidateToken(refreshToken, secret)
	if err != nil {
		return "", err
	}
	if claims.TokenType != RefreshTokenType {
		return "", ErrInvalidTokenType
	}

	return sign(Claims{
		UserID:    claims.UserID,
		TokenType: AccessToken,
	}, secret, newExpiration)
}

func sign(claims Claims, secret string, expiration time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrEmptySecret
	}
	if expiration <= 0 {
		return "", ErrInvalidExpiration
	}

	now := time.Now()
	claims.RegisteredClaims = jwtlib.RegisteredClaims{
		Subject:   claims.UserID,
		IssuedAt:  jwtlib.NewNumericDate(now),
		NotBefore: jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(expiration)),
	}

	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(secret))
}
