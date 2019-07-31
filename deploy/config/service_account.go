/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"encoding/json"
	"fmt"
)

// TODO Add service accounts into config.go

// ServiceAccount wraps a deployment manager service account.
type ServiceAccount struct {
	ServiceAccountProperties `json:"properties"`
	raw                      json.RawMessage
}

// ServiceAccountProperties represents a partial DM service account resource.
type ServiceAccountProperties struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

// Init initializes the instance.
func (sa *ServiceAccount) Init() error {
	return nil
}

// Name returns the name of this service account.
func (sa *ServiceAccount) Name() string {
	return sa.AccountID
}

// DeploymentManagerType returns the type to use for deployment manager.
func (*ServiceAccount) DeploymentManagerType() string {
	return "iam.v1.serviceAccount"
}

// aliasServiceAccount is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasServiceAccount ServiceAccount

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (sa *ServiceAccount) UnmarshalJSON(data []byte) error {
	var alias aliasServiceAccount
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*sa = ServiceAccount(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (sa *ServiceAccount) MarshalJSON() ([]byte, error) {
	return interfacePair{sa.raw, aliasServiceAccount(*sa)}.MarshalJSON()
}
