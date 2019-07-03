package config

import (
	"encoding/json"
	"errors"
)

// IAMCustomRole wraps a CFT IAM custom role.
type IAMCustomRole struct {
	IAMCustomRoleProperties `json:"properties"`
	raw                     json.RawMessage
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

// aliasIAMCustomRole is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasIAMCustomRole IAMCustomRole

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *IAMCustomRole) UnmarshalJSON(data []byte) error {
	var alias aliasIAMCustomRole
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return err
	}
	*i = IAMCustomRole(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *IAMCustomRole) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasIAMCustomRole(*i)}.MarshalJSON()
}

// IAMPolicy wraps a CFT IAM policy.
type IAMPolicy struct {
	IAMPolicyProperties `json:"properties"`
	IAMPolicyName       string `json:"name"`
	raw                 json.RawMessage
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

// aliasIAMPolicy is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasIAMPolicy IAMPolicy

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *IAMPolicy) UnmarshalJSON(data []byte) error {
	var alias aliasIAMPolicy
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return err
	}
	*i = IAMPolicy(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *IAMPolicy) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasIAMPolicy(*i)}.MarshalJSON()
}
