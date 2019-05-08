package cft

import "errors"

// IAMCustomRole wraps a CFT IAM custom role.
type IAMCustomRole struct {
	IAMCustomRoleProperties `json:"properties"`
}

// IAMCustomRoleProperties represents a partial IAM custom role implementation.
type IAMCustomRoleProperties struct {
	RoleID string `json:"roleId"`
}

// Init initializes a new custom role with the given project.
func (i *IAMCustomRole) Init(*Project) error {
	if i.Name() == "" {
		return errors.New("roleId must be set")
	}
	return nil
}

// Name returns the name of this custom role.
func (i *IAMCustomRole) Name() string {
	return i.RoleID
}

// TemplatePath returns the template to use for this custom role.
func (i *IAMCustomRole) TemplatePath() string {
	return "deploy/cft/templates/iam_custom_role/project_custom_role.py"
}
