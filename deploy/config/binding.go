package config

// Binding represents a GCP policy binding.
type Binding struct {
	Role    string   `json:"role" yaml:"role"`
	Members []string `json:"members" yaml:"members"`
}

// MergeBindings merges bindings together. It is typically used to merge default bindings with user specified bindings.
// Roles will be de-duplicated and merged into a single binding. Members are de-duplicated by deployment manager.
func MergeBindings(bs ...Binding) []Binding {
	roleToMembers := make(map[string][]string)
	var roles []string // preserve ordering

	for _, b := range bs {
		if _, ok := roleToMembers[b.Role]; !ok {
			roles = append(roles, b.Role)
		}
		roleToMembers[b.Role] = append(roleToMembers[b.Role], b.Members...)
	}

	var merged []Binding
	for _, role := range roles {
		merged = append(merged, Binding{Role: role, Members: roleToMembers[role]})
	}
	return merged
}
