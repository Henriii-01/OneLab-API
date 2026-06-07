package fileService

import (
	"context"
	"mime/multipart"
)

// IntegrationService is the base interface.
// Every registered integration MUST at least support checking its connectivity status.
type IntegrationService interface {
	CheckStatus(ctx context.Context) error
}

// FileTransferer is an optional interface.
// Only implemented by services that support uploading/transferring files.
type FileTransferer interface {
	TransferFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) error
}

// FileLookuper is an optional interface.
// Only implemented by services that support searching/looking up files.
type FileLookuper interface {
	LookupFile(ctx context.Context, filename string) (any, error)
}
