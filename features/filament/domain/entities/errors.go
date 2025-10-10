package entities

import "errors"

// Domain errors for filament feature
var (
	// ErrFilamentNotFound indicates that a filament was not found
	ErrFilamentNotFound = errors.New("filament not found")

	// ErrFilamentNameRequired indicates that filament name is required
	ErrFilamentNameRequired = errors.New("filament name is required")

	// ErrFilamentBrandRequired indicates that brand ID is required
	ErrFilamentBrandRequired = errors.New("brand ID is required")

	// ErrFilamentMaterialRequired indicates that material ID is required
	ErrFilamentMaterialRequired = errors.New("material ID is required")

	// ErrFilamentDiameterInvalid indicates that diameter is invalid
	ErrFilamentDiameterInvalid = errors.New("filament diameter must be greater than 0")

	// ErrFilamentPriceInvalid indicates that price is invalid
	ErrFilamentPriceInvalid = errors.New("filament price cannot be negative")

	// ErrFilamentColorInvalid indicates that color data is invalid
	ErrFilamentColorInvalid = errors.New("invalid color data")

	// ErrFilamentAccessDenied indicates that user cannot access the filament
	ErrFilamentAccessDenied = errors.New("access denied to filament")

	// ErrFilamentAlreadyExists indicates that a filament with the same name already exists
	ErrFilamentAlreadyExists = errors.New("filament with this name already exists")

	// ErrInvalidColorType indicates that the color type is not supported
	ErrInvalidColorType = errors.New("invalid color type")

	// ErrInvalidFilamentData indicates that filament data is invalid
	ErrInvalidFilamentData = errors.New("invalid filament data")
)
