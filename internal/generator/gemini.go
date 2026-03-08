package generator

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"google.golang.org/genai"

	"github.com/vinq1911/nanobanana-image-tool/internal/config"
	"github.com/vinq1911/nanobanana-image-tool/internal/models"
)

// GeminiGenerator implements ImageGenerator using the Google Gemini API.
// This is the canonical provider for Nano Banana 2 (gemini-2.5-flash-preview-image-generation).
type GeminiGenerator struct {
	cfg    *config.Config
	logger *slog.Logger
}

// NewGeminiGenerator creates a new Gemini-based image generator.
func NewGeminiGenerator(cfg *config.Config, logger *slog.Logger) *GeminiGenerator {
	return &GeminiGenerator{cfg: cfg, logger: logger}
}

func (g *GeminiGenerator) Generate(ctx context.Context, req models.GenerateRequest) (*models.ImageResult, []byte, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	seed := resolveSeed(req.Seed)

	g.logger.Info("generating image via Gemini API",
		"model", g.cfg.GeminiModel,
		"prompt", req.Prompt,
		"width", req.Width,
		"height", req.Height,
		"seed", seed,
	)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating Gemini client: %w", err)
	}

	prompt := req.Prompt
	if req.NegativePrompt != "" {
		prompt += ". Avoid: " + req.NegativePrompt
	}
	if req.Style != "" {
		prompt += ". Style: " + req.Style
	}

	seed32 := int32(seed & 0x7FFFFFFF)
	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{string(genai.ModalityImage), string(genai.ModalityText)},
		Seed:               &seed32,
		ImageConfig: &genai.ImageConfig{
			ImageSize: sizeToGemini(req.Width, req.Height),
		},
	}

	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	resp, err := client.Models.GenerateContent(ctx, g.cfg.GeminiModel, contents, config)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: gemini API call: %v", models.ErrGeneration, err)
	}

	imgData, mimeType, err := extractImage(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", models.ErrGeneration, err)
	}

	// Override format based on what the API actually returned.
	format := req.ImageFormat
	if mimeType == "image/jpeg" {
		format = "jpg"
	} else if mimeType == "image/png" {
		format = "png"
	}
	req.ImageFormat = format

	result := &models.ImageResult{
		Width:  req.Width,
		Height: req.Height,
		Seed:   seed,
		Prompt: req.Prompt,
		Model:  "nanobanana-2",
	}

	return result, imgData, nil
}

// extractImage finds the first image part in the Gemini response.
func extractImage(resp *genai.GenerateContentResponse) ([]byte, string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, "", fmt.Errorf("empty response from Gemini API")
	}

	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return part.InlineData.Data, part.InlineData.MIMEType, nil
			}
		}
	}

	return nil, "", fmt.Errorf("no image data in Gemini response")
}

// sizeToGemini maps pixel dimensions to Gemini's size presets.
func sizeToGemini(width, height int) string {
	maxDim := width
	if height > maxDim {
		maxDim = height
	}
	switch {
	case maxDim <= 512:
		return "" // let the API default
	case maxDim <= 1024:
		return "1K"
	case maxDim <= 2048:
		return "2K"
	default:
		return "4K"
	}
}

func resolveSeed(seed *int64) int64 {
	if seed != nil {
		return *seed
	}
	return rand.Int64N(1<<32) + 1
}
