package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	ValidateCredentials(clientID string, clientSecret string) bool //check if client is known
	GenerateToken(clientID string) (string, error)
	ValidateToken(token string) (string, error)
}

type Claims struct {
	ClientID string `json:"clientID"`
	jwt.RegisteredClaims
}

type jwtService struct {
	clients     map[string]string
	jwtSecret   []byte
	tokenExpiry time.Duration
}

func NewJwtService(clients map[string]string, jwtSecret string, tokenExpiry time.Duration) AuthService {

	return &jwtService{
		clients:     clients,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: tokenExpiry,
	}
}

func (s *jwtService) ValidateCredentials(clientID string, clientSecret string) bool {

	storedSecret, exists := s.clients[clientID]
	if !exists || storedSecret != clientSecret {
		return false
	}

	return true
}

func (s *jwtService) GenerateToken(clientID string) (string, error) {
	expiry := time.Now().Add(s.tokenExpiry)
	claims := &Claims{
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   clientID,
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *jwtService) ValidateToken(token string) (string, error) {

	parsedToken, err := jwt.ParseWithClaims(
		token,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return s.jwtSecret, nil
		},
	)
	if err != nil {
		return "", err
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return "", errors.New("invalid token")
	}

	return claims.ClientID, nil
}
