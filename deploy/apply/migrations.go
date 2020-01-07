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

package apply

import (
	"log"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// RemoveDeprecatedBigqueryAPI removes the deprecated BigqueryAPI if it was deployed.
// https://www.terraform.io/docs/providers/google/guides/version_3_upgrade.html#resource-google_project_service.
func RemoveDeprecatedBigqueryAPI(dir string, rn runner.Runner) error {
	cmd := exec.Command("terraform", "state", "list")
	cmd.Dir = dir
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return err
	}

	resources := strings.Split(string(out), "\n")
	for _, res := range resources {
		if strings.Contains(res, "google_project_service") && strings.Contains(res, "bigquery-json.googleapis.com") {
			log.Println("Found deprecated bigquery-json project service resource: ", res)
			cmd := exec.Command("terraform", "state", "rm", res)
			cmd.Dir = dir
			return rn.CmdRun(cmd)
		}
	}
	return nil
}
