package entities

import "errors"

var (
	// ErrBudgetNotFound is returned when a budget is not found
	ErrBudgetNotFound = errors.New("budget not found")

	// ErrInvalidStatus is returned when a budget status is invalid
	ErrInvalidStatus = errors.New("invalid budget status")

	// ErrInvalidTransition is returned when trying to perform an invalid status transition
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrBudgetStatusConflict is returned when a status update finds no matching row
	// because the budget's current status changed concurrently (optimistic guard).
	ErrBudgetStatusConflict = errors.New("budget status changed concurrently")

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

	// ErrInvalidPresetReference is returned when a referenced preset does not belong
	// to the caller's organization or is of the wrong type. It is safe to surface as
	// a 400 (bad request) with the stable code invalid_preset_reference.
	ErrInvalidPresetReference = errors.New("referenced preset does not belong to your organization")

	// ErrInvalidStatusFilter is returned when a list request carries a status filter
	// that is not one of the known budget statuses. Surfaced as a 400.
	ErrInvalidStatusFilter = errors.New("invalid status filter")

	// ErrProfileNotFound is returned when a budget references a print profile that
	// does not exist within the caller's organization (another tenant's profile, or
	// a soft-deleted one). It is safe to surface as a 400 (bad request).
	ErrProfileNotFound = errors.New("requested print profile does not belong to your organization")

	// ErrInvalidModel3DReference is returned when a budget item references a 3D model
	// that does not belong to the caller's organization. Safe to surface as a 400.
	ErrInvalidModel3DReference = errors.New("referenced 3D model does not belong to your organization")

	// ErrUnauthorizedAccess is returned when user tries to access a budget they don't own
	ErrUnauthorizedAccess = errors.New("unauthorized access to budget")
)
