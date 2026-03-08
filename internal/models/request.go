package models

// ReferenceImage is a named character reference image passed to the generator.
type ReferenceImage struct {
	Name string `json:"name"`
	Data []byte `json:"-"` // image bytes, not serialized over JSON
}

// GenerateRequest represents an image generation request.
type GenerateRequest struct {
	Prompt          string           `json:"prompt"`
	NegativePrompt  string           `json:"negative_prompt,omitempty"`
	Width           int              `json:"width"`
	Height          int              `json:"height"`
	Seed            *int64           `json:"seed,omitempty"`
	Style           string           `json:"style,omitempty"`
	OutputDir       string           `json:"output_dir,omitempty"`
	ImageFormat     string           `json:"image_format,omitempty"`
	ReferenceNames  []string         `json:"reference_names,omitempty"`
	ReferenceImages []ReferenceImage `json:"-"` // populated by caller after loading from store
}

// Validate checks that the request has the minimum required fields
// and applies sensible defaults.
func (r *GenerateRequest) Validate() error {
	if r.Prompt == "" {
		return ErrEmptyPrompt
	}
	if r.Width <= 0 {
		r.Width = 1024
	}
	if r.Height <= 0 {
		r.Height = 1024
	}
	if r.ImageFormat == "" {
		r.ImageFormat = "png"
	}
	if r.ImageFormat != "png" && r.ImageFormat != "jpg" {
		return ErrInvalidFormat
	}
	return nil
}
