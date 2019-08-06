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
	"strings"
	"testing"
	"time"

	"google3/third_party/golang/godebug/pretty/pretty"
)

var (
	configText1 = "config#1"
	configText2 = "config#2"
	createTime1 = time.Now()
	createTime2 = createTime1.Add(time.Hour)
	md5Hash1    = "hash1"
	md5Hash2    = "hash2"

	auditConfig1 = &AuditConfig{
		projectConfigText: configText1,
		createTime:        &createTime1,
		md5Hash:           md5Hash1,
	}

	auditConfig2 = &AuditConfig{
		projectConfigText: configText2,
		createTime:        &createTime2,
		md5Hash:           md5Hash2,
	}
)

func TestAuditConfigs(t *testing.T) {
	ac := NewAuditAPIClient()

	ac.CreateAuditConfig(auditConfig1)
	ac.CreateAuditConfig(auditConfig2)

	config, err := ac.FetchLatestAuditConfig()
	if err != nil {
		t.Errorf("FetchLatestAuditConfig return error: %v", err)
	}
	if diff := pretty.Compare(config.projectConfigText, configText2); diff != "" {
		t.Errorf("FetchLatestAuditConfig() returned if (-want +got): \n%s", diff)
	}
	if diff := pretty.Compare(ac.DebugDumpAuditConfigs(), strings.Join([]string{configText1, configText2}, "|")); diff != "" {
		t.Errorf("DebugDumpAuditConfigs() returned if (-want +got): \n%s", diff)
	}

	ac.DebugClearAuditConfigs()
	if diff := pretty.Compare(ac.DebugDumpAuditConfigs(), ""); diff != "" {
		t.Errorf("DebugDumpAuditConfigs() returned if (-want +got): \n%s", diff)
	}
}
