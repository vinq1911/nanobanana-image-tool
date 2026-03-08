package generator

import (
	"fmt"
	"log/slog"

	"github.com/vinq1911/nanobanana-image-tool/internal/config"
)

// New creates the appropriate ImageGenerator based on config.Provider.
func New(cfg *config.Config, logger *slog.Logger) (ImageGenerator, error) {
	switch cfg.Provider {
	case "gemini":
		if cfg.GeminiAPIKey == "" {
			return nil, fmt.Errorf("NANOBANANA_GEMINI_API_KEY (or GEMINI_API_KEY / GOOGLE_API_KEY) is required for gemini provider")
		}
		logger.Info("using Gemini provider", "model", cfg.GeminiModel)
		return NewGeminiGenerator(cfg, logger), nil

	case "falai":
		if cfg.FalAIKey == "" {
			return nil, fmt.Errorf("NANOBANANA_FALAI_KEY (or FAL_KEY) is required for falai provider")
		}
		logger.Info("using fal.ai provider", "model", cfg.FalAIModel)
		return NewFalAIGenerator(cfg, logger), nil

	default:
		return nil, fmt.Errorf("unknown provider %q: must be \"gemini\" or \"falai\"", cfg.Provider)
	}
}
