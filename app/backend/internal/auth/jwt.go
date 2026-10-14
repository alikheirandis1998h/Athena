package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrWrongType    = errors.New("unexpected token type")
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email,omitempty"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *TokenManager) IssueAccessToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.accessSecret)
}

func (m *TokenManager) IssueRefreshToken(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTTL)
	claims := Claims{
		UserID: userID,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*Claims, error) {
	return m.parseToken(tokenString, m.accessSecret, TokenTypeAccess)
}

func (m *TokenManager) ParseRefreshToken(tokenString string) (*Claims, error) {
	return m.parseToken(tokenString, m.refreshSecret, TokenTypeRefresh)
}

func (m *TokenManager) parseToken(tokenString string, secret []byte, expectedType TokenType) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Type != expectedType {
		return nil, ErrWrongType
	}
	return claims, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
