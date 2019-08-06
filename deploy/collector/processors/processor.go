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

// Package processors provides processors that collect data and save them via AuditAPIClient.
package processors

import (
	"context"
	"time"
)

// UpdateProjectConfig updates the project config at the specified path.
// It reads the latest config version from the API and specified GCS path. If the
// md5_hash has changed, then create a new project config version via AuditAPIClient.
func UpdateProjectConfig(ctx context.Context, filePath string, apiClient *AuditAPIClient, gcsClient *GCSClient) error {
	latestConfig, err := apiClient.FetchLatestAuditConfig()
	if err != nil {
		return err
	}

	md5Hash, err := gcsClient.ObjectHash(ctx, filePath)
	if err != nil {
		return err
	}

	// lastestConfig could be nil if there is no stored AuditConfig in AuditAPIClient.
	if latestConfig != nil {
		// There is at least one stored config.
		// Do nothing if no change in config.
		if string(md5Hash) == latestConfig.md5Hash {
			return nil
		}
	}

	content, err := gcsClient.ObjectContent(ctx, filePath)
	if err != nil {
		return err
	}
	createTime := time.Now()
	err = apiClient.CreateAuditConfig(&AuditConfig{
		projectConfigText: string(content),
		createTime:        &createTime,
		md5Hash:           string(md5Hash),
	})

	return err
}
