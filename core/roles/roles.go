package roles

import "github.com/RodolfoBonis/spooliq/core/entities"

// System role constants
const (
	UserRole          = "User"
	OrgAdmin          = "OrgAdmin"
	PlatformAdminRole = "PlatformAdmin"
)

// ExampleRoles provides example user roles for the system.
var ExampleRoles = entities.Roles{
	Search: "search-examples",
	Insert: "add-example",
	Detail: "example-details",
	Update: "update-example",
	Delete: "delete-example",
}
