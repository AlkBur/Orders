package ui

// Access defines visibility restrictions for UI elements.
// The zero value means the element is visible to all authenticated users.
type Access struct {
	Admin bool
}
