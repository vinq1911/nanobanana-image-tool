package models

import "errors"

var (
	ErrEmptyPrompt   = errors.New("prompt must not be empty")
	ErrInvalidFormat = errors.New("image_format must be 'png' or 'jpg'")
	ErrGeneration    = errors.New("image generation failed")
)
