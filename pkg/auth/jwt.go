package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims holds the payload embedded in every JWT token.
// These are the fields available to any handler after the
// JWT middleware has validated and attached them to the context.
type Claims struct {
	// UserID is the UUID primary key from the users table.
	UserID uuid.UUID `json:"sub"`

	// Phone is the user's E.164 phone number — the primary login identifier.
	Phone string `json:"phone"`

	// Role is one of: shipper, driver, fleet_admin, owner_operator, admin.
	Role string `json:"role"`

	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation, verification and refresh.
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWTManager.
// secretKey must be at least 32 characters long.
func NewJWTManager(secretKey string, tokenDuration time.Duration) (*JWTManager, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("JWT secret key cannot be empty")
	}
	if len(secretKey) < 32 {
		return nil, fmt.Errorf("JWT secret key must be at least 32 characters long")
	}

	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}, nil
}

// Generate creates a signed JWT token for the given user.
// The token is valid for the duration configured in JWTManager.
func (m *JWTManager) Generate(userID uuid.UUID, phone, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Phone:  phone,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

// Verify validates the token's signature and expiry, then returns its claims.
// Returns an error if the token is malformed, expired, or has the wrong signing method.
func (m *JWTManager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.secretKey), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// Refresh issues a new token with a fresh expiry, preserving the same claims.
// Used by the /auth/refresh endpoint.
func (m *JWTManager) Refresh(oldToken string) (string, error) {
	claims, err := m.Verify(oldToken)
	if err != nil {
		return "", err
	}
	return m.Generate(claims.UserID, claims.Phone, claims.Role)
}
