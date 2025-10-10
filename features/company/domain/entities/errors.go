package entities

import "errors"

var (
	// ErrCompanyNotFound is returned when a company is not found
	ErrCompanyNotFound = errors.New("company not found")

	// ErrCompanyAlreadyExists is returned when trying to create a company for an organization that already has one
	ErrCompanyAlreadyExists = errors.New("company already exists for this organization")

	// ErrUnauthorized is returned when a user tries to access a company they don't have permission to
	ErrUnauthorized = errors.New("unauthorized to access this company")
)
