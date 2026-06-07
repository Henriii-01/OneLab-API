package fileService

import (
	"context"
	"mime/multipart"
)

// IntegrationService defines how the controller communicates with different target software.
// Implementing this interface makes it easy to add new services in the future.
type IntegrationService interface {
	TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error
	CheckStatus(ctx context.Context) error
}
