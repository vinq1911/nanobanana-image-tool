package generator

import (
	"context"

	"github.com/vinq1911/nanobanana-image-tool/internal/models"
)

// ImageGenerator defines the interface for image generation backends.
type ImageGenerator interface {
	// Generate produces an image from the given request.
	Generate(ctx context.Context, req models.GenerateRequest) (*models.ImageResult, []byte, error)
}
