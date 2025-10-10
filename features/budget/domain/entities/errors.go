package entities

import "errors"

var (
	// ErrBudgetNotFound is returned when a budget is not found
	ErrBudgetNotFound = errors.New("budget not found")

	// ErrInvalidStatus is returned when a budget status is invalid
	ErrInvalidStatus = errors.New("invalid budget status")

	// ErrInvalidTransition is returned when trying to perform an invalid status transition
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrCustomerNotFound is returned when the customer for a budget is not found
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrFilamentNotFound is returned when a filament item is not found
	ErrFilamentNotFound = errors.New("filament not found")

	// ErrBudgetNotEditable is returned when trying to edit a non-draft budget
	ErrBudgetNotEditable = errors.New("only draft budgets can be edited")

	// ErrBudgetNotDeletable is returned when trying to delete a printing/completed budget
	ErrBudgetNotDeletable = errors.New("cannot delete printing or completed budgets")

	// ErrInvalidBudgetData is returned when budget data is invalid
	ErrInvalidBudgetData = errors.New("invalid budget data")

	// ErrNoItems is returned when trying to create a budget without items
	ErrNoItems = errors.New("budget must have at least one filament item")

	// ErrInvalidPrintTime is returned when print time is invalid
	ErrInvalidPrintTime = errors.New("print time must be greater than zero")

	// ErrPresetNotFound is returned when a preset is not found
	ErrPresetNotFound = errors.New("preset not found")

	// ErrUnauthorizedAccess is returned when user tries to access a budget they don't own
	ErrUnauthorizedAccess = errors.New("unauthorized access to budget")
)
