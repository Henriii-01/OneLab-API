package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	ValidateCredentials(clientID string, clientSecret string) bool
	GenerateToken(clientID string) (string, error)
	ValidateToken(token string) (string, error)
	NewJwtClient(clientID string, clientSecret string)
	tokenDuration() time.Duration
}

type Claims struct {
	ClientID string `json:"clientID"`
	jwt.RegisteredClaims
}

type jwtService struct {
	clients            map[string]jwtClient
	tokens             map[string][]issuedToken
	expirationDuration time.Duration
}
type jwtClient struct {
	clientSecret string
	jwtSecret    []byte
}
type issuedToken struct {
	token     string
	issuedAt  time.Time
	expiresAt time.Time
}

func NewJwtService(duration time.Duration) AuthService {
	return &jwtService{
		clients:            make(map[string]jwtClient),
		tokens:             make(map[string][]issuedToken),
		expirationDuration: duration,
	}
}

func (s *jwtService) NewJwtClient(clientID string, clientSecret string) {
	s.clients[clientID] = jwtClient{
		clientSecret: clientSecret,
		jwtSecret:    []byte(clientSecret),
	}
}

func (s *jwtService) tokenDuration() time.Duration { return s.expirationDuration }

func (s *jwtService) ValidateCredentials(clientID string, clientSecret string) bool {
	c, exists := s.clients[clientID]
	return exists && c.clientSecret == clientSecret
}

func (s *jwtService) GenerateToken(clientID string) (string, error) {
	c, exists := s.clients[clientID]
	if !exists {
		return "", errors.New("unknown client")
	}

	now := time.Now()
	expirationTime := now.Add(s.expirationDuration)
	claims := &Claims{
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   clientID,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	signedString, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(c.jwtSecret)
	s.tokens[clientID] = append(s.tokens[clientID], issuedToken{
		token:     signedString,
		issuedAt:  now,
		expiresAt: expirationTime,
	})
	return signedString, nil
}

func (s *jwtService) ValidateToken(tokenString string) (string, error) {
	parsedToken, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			claims, ok := token.Claims.(*Claims)
			if !ok {
				return nil, errors.New("invalid claims")
			}
			c, exists := s.clients[claims.ClientID]
			if !exists {
				return nil, errors.New("unknown client")
			}
			return c.jwtSecret, nil
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
