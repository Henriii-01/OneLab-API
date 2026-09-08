package auth

import (
	"errors"
	"log/slog"
	"time"

	"github.com/HTMLuke/OneLab-API/secretProvider"
	libjwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ClientID string `json:"clientID"`
	libjwt.RegisteredClaims
}

type jwtService struct {
	clients            map[string]jwtClient
	expirationDuration time.Duration
	jwtSecret          []byte
	logger             *slog.Logger
}

type jwtClient struct {
	clientSecret string
}

func init() {
	RegisterAuth("jwt", NewJwtService)
}

func NewJwtService(duration time.Duration, secretService secretProvider.SecretService, logger *slog.Logger) (AuthService, error) {
	jwtSecret, err := secretService.GetSecret("ONELAB_JWT_SECRET")
	if err != nil {
		return nil, err
	}
	clientID, err := secretService.GetSecret("ONELAB_AUTH_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	clientSecret, err := secretService.GetSecret("ONELAB_AUTH_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	return &jwtService{
		clients:            map[string]jwtClient{clientID: {clientSecret: clientSecret}},
		expirationDuration: duration,
		jwtSecret:          []byte(jwtSecret),
		logger:             logger,
	}, nil

}
func (s *jwtService) ValidateCredentials(clientID string, clientSecret string) bool {
	c, exists := s.clients[clientID]
	return exists && c.clientSecret == clientSecret
}

func (s *jwtService) GenerateToken(clientID, clientSecret string) (string, int, error) {
	c, exists := s.clients[clientID]
	if !exists || c.clientSecret != clientSecret {
		if s.logger != nil {
			s.logger.Debug("jwt credentials rejected")
		}
		return "", 0, errInvalidCredentials
	}

	now := time.Now()
	expirationTime := now.Add(s.expirationDuration)
	claims := &Claims{
		ClientID: clientID,
		RegisteredClaims: libjwt.RegisteredClaims{
			Subject:   clientID,
			ExpiresAt: libjwt.NewNumericDate(expirationTime),
			IssuedAt:  libjwt.NewNumericDate(now),
		},
	}
	signedString, err := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, err
	}

	if s.logger != nil {
		s.logger.Debug("jwt token generated", "expires_in_seconds", int(s.expirationDuration.Seconds()))
	}
	return signedString, int(s.expirationDuration.Seconds()), nil
}

func (s *jwtService) ValidateToken(tokenString string) (string, error) {
	parsedToken, err := libjwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *libjwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*libjwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			claims, ok := token.Claims.(*Claims)
			if !ok {
				return nil, errors.New("invalid claims")
			}
			if _, exists := s.clients[claims.ClientID]; !exists {
				return nil, errors.New("unknown client")
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
