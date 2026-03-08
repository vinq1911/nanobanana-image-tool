package models

// ImageResult represents the output of a successful image generation.
type ImageResult struct {
	ImagePath string `json:"image_path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Seed      int64  `json:"seed"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
}

// GenerateResponse wraps the API response returned by the HTTP server.
type GenerateResponse struct {
	ImagePath string      `json:"image_path"`
	Metadata  ImageResult `json:"metadata"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
