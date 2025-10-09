package entities

import "errors"

var (
	// ErrPresetNameRequired error message when preset name is not passed
	ErrPresetNameRequired = errors.New("preset name is required")
	// ErrInvalidPresetType error message when preset type is not valid
	ErrInvalidPresetType = errors.New("invalid preset type")
)
