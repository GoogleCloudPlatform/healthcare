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

package processors

import (
	"errors"
	"strings"
	"time"
)

// AuditConfig represents an audit config.
type AuditConfig struct {
	projectConfigText string
	createTime        *time.Time
	md5Hash           string
}

// AuditAPIClient presents a prototype for the future Audit API.
// Currently, this client fakes API calls by reading and writing resources in
// memory. Once the API is launched, this client will be replaced with one backed
// by real API calls.
type AuditAPIClient struct {
	auditConfigs map[string]AuditConfig
	// TODO: adds other resources.
}

// NewAuditAPIClient creates an audit api client.
func NewAuditAPIClient() *AuditAPIClient {
	return &AuditAPIClient{
		auditConfigs: make(map[string]AuditConfig),
	}
}

// CreateAuditConfig creats a config in memory.
func (ac *AuditAPIClient) CreateAuditConfig(auditConfig *AuditConfig) error {
	if auditConfig.md5Hash == "" {
		return errors.New("no md5 hash in AuditConfig to create")
	}
	ac.auditConfigs[auditConfig.md5Hash] = *auditConfig
	return nil
}

// FetchLatestAuditConfig returns the latest AuditConfig. If there is no stored audit config, it return nil, nil.
func (ac *AuditAPIClient) FetchLatestAuditConfig() (*AuditConfig, error) {
	var latestTime *time.Time
	var latestConfig *AuditConfig
	for _, v := range ac.auditConfigs {
		if v.createTime == nil {
			return nil, errors.New("contains AuditConfig without createTime")
		}
		if latestTime == nil || latestTime.Before(*v.createTime) {
			latestTime = v.createTime
			latestConfig = &v
		}
	}
	return latestConfig, nil
}

// DebugDumpAuditConfigs returns a string that contains all configs stored.
func (ac *AuditAPIClient) DebugDumpAuditConfigs() string {
	text := make([]string, 0)
	for _, v := range ac.auditConfigs {
		text = append(text, v.projectConfigText)
	}
	return strings.Join(text, "|")
}

// DebugClearAuditConfigs clears all stored AuditConfig.
func (ac *AuditAPIClient) DebugClearAuditConfigs() {
	ac.auditConfigs = make(map[string]AuditConfig)
}

// DebugDumpAll returns a string that contains all audit data stored.
func (ac *AuditAPIClient) DebugDumpAll() string {
	return strings.Join([]string{ac.DebugDumpAuditConfigs()}, "||||")
}

// DebugClearAll clears all stored audit data.
func (ac *AuditAPIClient) DebugClearAll() {
	ac.DebugClearAuditConfigs()
}
