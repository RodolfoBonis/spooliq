package types

// Array represents a generic array type.
type Array []any

// Contains checks if the array contains a specific element.
func (a *Array) Contains(value any) bool {
	list := *a
	for i := 0; i < len(list); i++ {
		if list[i] == value {
			return true
		}
	}
	return false
}
