package entities

import "errors"

var (
	// ErrPresetNameRequired error message when preset name is not passed
	ErrPresetNameRequired = errors.New("preset name is required")
	// ErrInvalidPresetType error message when preset type is not valid
	ErrInvalidPresetType = errors.New("invalid preset type")
	// ErrCannotDeleteDefaultPreset error message when attempting to delete a default preset
	ErrCannotDeleteDefaultPreset = errors.New("default presets cannot be deleted")
	// ErrPresetNotFound error message when a preset does not exist within the organization scope
	ErrPresetNotFound = errors.New("preset not found")
	// ErrTemplateNotFound error message when a preset template key is unknown
	ErrTemplateNotFound = errors.New("preset template not found")
)
