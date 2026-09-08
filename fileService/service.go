package fileService

import (
	"context"
	"mime/multipart"
)

// IntegrationService is the base interface.
// Every registered integration MUST at least support checking its connectivity status.
type IntegrationService interface {
	CheckStatus(ctx context.Context) error
	GetFile(ctx context.Context, fileID string) ([]byte, error)
	AddFile(ctx context.Context, file []byte, filename string) error
}

// PaperlessBackupService is an optional interface for services that can prepare and
// persist a local backup export created inside the target application container.
type PaperlessBackupService interface {
	Backup(ctx context.Context) (string, error)
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
