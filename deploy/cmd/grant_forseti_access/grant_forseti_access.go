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

// grant_forseti_access grants default permissions on project to forseti.
//
// Usage:
//   $ bazel run :grant_forseti_access -- --project_id=${PROJECT_ID?} --forseti_service_account=${FORSETI_SERVICE_ACCOUNT?}
package main

import (
	"log"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

var (
	projectID             = flag.String("project_id", "", "ID of project to grant forseti access to")
	forsetiServiceAccount = flag.String("forseti_service_account", "", "Service account of forseti")
)

func main() {
	flag.Parse()

	if *projectID == "" {
		log.Fatal("--project_id must be set")
	}

	if *forsetiServiceAccount == "" {
		log.Fatal("--forseti_service_account must be set")
	}

	if err := apply.GrantForsetiPermissions(*projectID, *forsetiServiceAccount, &runner.DefaultRunner{}); err != nil {
		log.Fatalf("failed to grant forseti permissions: %v", err)
	}
}
