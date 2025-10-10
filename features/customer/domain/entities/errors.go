package entities

import "errors"

var (
	// ErrCustomerNotFound is returned when a customer is not found
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrCustomerAlreadyExists is returned when trying to create a customer that already exists
	ErrCustomerAlreadyExists = errors.New("customer already exists")

	// ErrCustomerInUse is returned when trying to delete a customer that has associated budgets
	ErrCustomerInUse = errors.New("customer has associated budgets and cannot be deleted")

	// ErrInvalidCustomerData is returned when customer data is invalid
	ErrInvalidCustomerData = errors.New("invalid customer data")

	// ErrUnauthorizedAccess is returned when user tries to access a customer they don't own
	ErrUnauthorizedAccess = errors.New("unauthorized access to customer")
)
