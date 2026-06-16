package secretProvider

import (
	"fmt"
	"os"
)

type SecretService interface {
	// GetSecret returns the value for key and an error if it is empty
	// required state can be set via ignoring or handling the error
	GetSecret(key string) (string, error)
}
type envSecretService struct{}

func NewSecretService() SecretService {

	return &envSecretService{}
}

func (s *envSecretService) GetSecret(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required secret %q is not set", key)
	}
	return value, nil
}
