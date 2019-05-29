package config

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
	return "deploy/config/templates/iam_custom_role/project_custom_role.py"
}

// IAMPolicy wraps a CFT IAM policy.
type IAMPolicy struct {
	IAMPolicyProperties `json:"properties"`
	IAMPolicyName       string `json:"name"`
}

// IAMPolicyProperties represents a partial IAM policy implementation.
type IAMPolicyProperties struct {
	Bindings []Binding `json:"roles"`
}

// Init initializes a new custom role with the given project.
func (i *IAMPolicy) Init(*Project) error {
	if i.Name() == "" {
		return errors.New("name must be set")
	}
	return nil
}

// Name returns the name of this custom role.
func (i *IAMPolicy) Name() string {
	return i.IAMPolicyName
}

// TemplatePath returns the template to use for this custom role.
func (i *IAMPolicy) TemplatePath() string {
	return "deploy/config/templates/iam_member/iam_member.py"
}
