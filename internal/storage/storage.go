package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Storage defines the interface for persisting generated images.
type Storage interface {
	Save(ctx context.Context, data []byte, format string, outputDir string) (string, error)
}

// LocalStorage saves images to the local filesystem.
type LocalStorage struct {
	logger *slog.Logger
}

// NewLocalStorage creates a new local storage backend.
func NewLocalStorage(logger *slog.Logger) *LocalStorage {
	return &LocalStorage{logger: logger}
}

// Save writes the image data to the output directory and returns the file path.
func (s *LocalStorage) Save(_ context.Context, data []byte, format string, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	filename := fmt.Sprintf("nanobanana_%s.%s", time.Now().UTC().Format("20060102_150405_000"), format)
	fullPath := filepath.Join(outputDir, filename)

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("writing image file: %w", err)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		absPath = fullPath
	}

	s.logger.Info("image saved", "path", absPath, "size_bytes", len(data))
	return absPath, nil
}
