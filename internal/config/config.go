package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the nanobanana-image-tool.
type Config struct {
	// Provider: "gemini" (Google Gemini API) or "falai" (fal.ai API).
	Provider string

	// Google Gemini API key. Used when Provider is "gemini".
	// Can also be set via GOOGLE_API_KEY or GEMINI_API_KEY.
	GeminiAPIKey string

	// Gemini model ID for image generation.
	GeminiModel string

	// fal.ai API key. Used when Provider is "falai".
	FalAIKey string

	// fal.ai model ID.
	FalAIModel string

	// Default output directory for generated images.
	OutputDir string

	// HTTP server listen address.
	ListenAddr string

	// HTTP server port.
	Port int

	// Default image format (png or jpg).
	DefaultFormat string

	// Default image width.
	DefaultWidth int

	// Default image height.
	DefaultHeight int
}

// Load reads configuration from a .env file (if present) and environment
// variables. Real environment variables take precedence over .env values.
func Load() *Config {
	loadDotEnv(".env")

	return &Config{
		Provider:      getEnv("NANOBANANA_PROVIDER", "gemini"),
		GeminiAPIKey:  getEnv("NANOBANANA_GEMINI_API_KEY", getEnv("GEMINI_API_KEY", getEnv("GOOGLE_API_KEY", ""))),
		GeminiModel:   getEnv("NANOBANANA_GEMINI_MODEL", "gemini-2.5-flash-image"),
		FalAIKey:      getEnv("NANOBANANA_FALAI_KEY", getEnv("FAL_KEY", "")),
		FalAIModel:    getEnv("NANOBANANA_FALAI_MODEL", "fal-ai/nano-banana-2"),
		OutputDir:     getEnv("NANOBANANA_OUTPUT_DIR", "./output"),
		ListenAddr:    getEnv("NANOBANANA_LISTEN_ADDR", "0.0.0.0"),
		Port:          getEnvInt("NANOBANANA_PORT", 8080),
		DefaultFormat: getEnv("NANOBANANA_DEFAULT_FORMAT", "png"),
		DefaultWidth:  getEnvInt("NANOBANANA_DEFAULT_WIDTH", 1024),
		DefaultHeight: getEnvInt("NANOBANANA_DEFAULT_HEIGHT", 1024),
	}
}

// loadDotEnv reads a .env file and sets environment variables for any keys
// that are not already set. Existing env vars always take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Strip surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Don't overwrite existing env vars.
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
