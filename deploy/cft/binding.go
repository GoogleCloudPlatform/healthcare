package cft

type binding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

// mergeBindings merges bindings together. It is typically used to merge default bindings with user specified bindings.
// Roles will be de-duplicated and merged into a single binding. Members are de-duplicated by deployment manager.
func mergeBindings(bs ...binding) []binding {
	roleToMembers := make(map[string][]string)
	var roles []string // preserve ordering

	for _, b := range bs {
		if _, ok := roleToMembers[b.Role]; !ok {
			roles = append(roles, b.Role)
		}
		roleToMembers[b.Role] = append(roleToMembers[b.Role], b.Members...)
	}

	var merged []binding
	for _, role := range roles {
		merged = append(merged, binding{Role: role, Members: roleToMembers[role]})
	}
	return merged
}
