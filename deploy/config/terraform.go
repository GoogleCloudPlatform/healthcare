// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
)

func (p *Project) initTerraform(auditProject *Project) error {
	if err := p.initTerraformAuditResources(auditProject); err != nil {
		return fmt.Errorf("failed to init audit resources: %v", err)
	}

	for _, r := range p.TerraformResources() {
		if err := r.Init(p.ID); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) initTerraformAuditResources(auditProject *Project) error {
	d := p.Audit.LogsBigqueryDataset
	if d == nil {
		return errors.New("audit.logs_bigquery_dataset must be set")
	}

	if err := d.Init(auditProject.ID); err != nil {
		return fmt.Errorf("failed to init logs bq dataset: %v", err)
	}

	accesses := []*tfconfig.Access{
		{Role: "OWNER", GroupByEmail: auditProject.OwnersGroup},
		{Role: "READER", GroupByEmail: p.AuditorsGroup},
	}
	d.Accesses = accesses

	p.BQLogSinkTF = &tfconfig.LoggingSink{
		Name:                 "audit-logs-to-bigquery",
		Destination:          fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/%s", auditProject.ID, d.DatasetID),
		Filter:               `logName:"logs/cloudaudit.googleapis.com"`,
		UniqueWriterIdentity: true,
	}
	if err := p.BQLogSinkTF.Init(p.ID); err != nil {
		return fmt.Errorf("failed to init bigquery log sink: %v", err)
	}

	b := p.Audit.LogsStorageBucket
	if b == nil {
		return nil
	}

	if err := b.Init(auditProject.ID); err != nil {
		return fmt.Errorf("failed to init logs gcs bucket: %v", err)
	}

	b.IAMMembers = append(b.IAMMembers,
		&tfconfig.StorageIAMMember{Role: "roles/storage.admin", Member: "group:" + auditProject.OwnersGroup},
		&tfconfig.StorageIAMMember{Role: "roles/storage.objectCreator", Member: accessLogsWriter},
		&tfconfig.StorageIAMMember{Role: "roles/storage.objectViewer", Member: "group:" + p.AuditorsGroup})
	return nil
}

// TerraformResources gets all terraform data resources in this project.
func (p *Project) TerraformResources() []tfconfig.Resource {
	var rs []tfconfig.Resource
	for _, r := range p.StorageBuckets {
		rs = append(rs, r)
	}
	for _, r := range p.BigqueryDatasets {
		rs = append(rs, r)
	}
	return rs
}
