package jwt

import (
	"time"

	"github.com/Abraxas-365/opd/internal/user"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey       []byte
	tokenDuration   time.Duration
	refreshDuration time.Duration
}

type Claims struct {
	UserID           string                `json:"user_id"`
	Email            string                `json:"email"`
	IsAdmin          bool                  `json:"is_admin"`
	Role             user.Role             `json:"role"`
	SubscriptionType user.SubscriptionType `json:"subscription_type,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTService creates a new JWT service with the given secret key and durations
func NewJWTService(secretKey string, tokenDuration, refreshDuration time.Duration) *JWTService {
	return &JWTService{
		secretKey:       []byte(secretKey),
		tokenDuration:   tokenDuration,
		refreshDuration: refreshDuration,
	}
}

// GenerateToken creates a new JWT token for a user
func (s *JWTService) GenerateToken(user *user.User) (string, error) {
	expirationTime := time.Now().Add(s.tokenDuration)

	claims := &Claims{
		UserID:           user.ID,
		Email:            user.Email,
		IsAdmin:          user.IsAdmin,
		Role:             user.Role,
		SubscriptionType: user.SubscriptionType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken creates a refresh token with longer expiration
func (s *JWTService) GenerateRefreshToken(userID string) (string, error) {
	expirationTime := time.Now().Add(s.refreshDuration)

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken validates the provided token string
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return s.secretKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *JWTService) ValidateRefreshToken(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			return s.secretKey, nil
		},
	)

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	return claims.Subject, nil
}
