package lenses

// User represents the logged user, it contains the name, e-mail and the given roles.
type User struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}
