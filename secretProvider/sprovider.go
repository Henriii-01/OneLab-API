package secretProvider

import (
	"os"
)

type SecretService interface {
	GetNextcloudUser() string
	GetNextcloudPassword() string
	GetPaperlessToken() string
}
type envSecretService struct{}

func NewSecretService() SecretService {

	return &envSecretService{}
}

func (s *envSecretService) GetNextcloudUser() string {
	return os.Getenv("ONELAB_NEXTCLOUD_USER")
}

func (s *envSecretService) GetNextcloudPassword() string {
	return os.Getenv("ONELAB_NEXTCLOUD_PASSWORD")
}

func (s *envSecretService) GetPaperlessToken() string {
	return os.Getenv("ONELAB_PAPERLESS_TOKEN")
}
