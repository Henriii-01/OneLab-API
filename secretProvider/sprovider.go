package secretProvider

import (
	"fmt"
	"log/slog"
	"os"
)

type SecretService interface {
	// GetSecret returns the value for key and an error if it is empty
	// required state can be set via ignoring or handling the error
	GetSecret(key string) (string, error)
}
type envSecretService struct {
	logger *slog.Logger
}

func NewSecretService(logger *slog.Logger) SecretService {
	return &envSecretService{logger: logger}
}

func (s *envSecretService) GetSecret(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		if s.logger != nil {
			s.logger.Warn("required secret is missing", "secret_key", key)
		}
		return "", fmt.Errorf("required secret %q is not set", key)
	}
	if s.logger != nil {
		s.logger.Debug("secret loaded", "secret_key", key)
	}
	return value, nil
}
