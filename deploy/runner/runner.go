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

// Package runner provides utilities to execute commands.
package runner

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Runner is the interface to execute commands.
type Runner interface {
	CmdRun(cmd *exec.Cmd) error
	CmdOutput(cmd *exec.Cmd) ([]byte, error)
	CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error)
}

// Default is the Runner that executes the command by default.
type Default struct{}

// CmdRun executes the command.
func (*Default) CmdRun(cmd *exec.Cmd) error {
	log.Printf("Running: %v", cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

// CmdOutput executes the command and returns the command stdout.
func (*Default) CmdOutput(cmd *exec.Cmd) ([]byte, error) {
	log.Printf("Running: %v", cmd.Args)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return b, fmt.Errorf("%v: %s", err, stderr.String())
	}
	return b, nil
}

// CmdCombinedOutput executes the command and returns the command stdout and stderr.
func (*Default) CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	log.Printf("Running: %v", cmd.Args)
	return cmd.CombinedOutput()
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// Fake is the Runner that only prints command information and does not run it.
type Fake struct{}

// CmdRun prints the command information.
func (*Fake) CmdRun(cmd *exec.Cmd) error {
	log.Printf("Dry run call: %s", strings.Join(cmd.Args, " "))
	return nil
}

// CmdOutput prints the command information.
func (*Fake) CmdOutput(cmd *exec.Cmd) ([]byte, error) {
	log.Printf("Dry run call: %s", strings.Join(cmd.Args, " "))
	switch cmdStr := strings.Join(cmd.Args, " "); {
	case contains(cmdStr, "projects describe"):
		return nil, errors.New("")
	case contains(cmdStr, "logging sinks describe audit-logs-to-bigquery", "--format json"):
		return []byte("{}"), nil
	case contains(cmdStr, "config get-value account", "--format json"):
		return []byte(`"dryrun@dryrun.com"`), nil
	case contains(cmdStr, "projects get-iam-policy"):
		return []byte("{}"), nil
	case contains(cmdStr, "service-accounts list", "--filter email:forseti-server-gcp-*", "--format json"):
		return []byte(`[{"email": "forseti-server-gcp-dryrun@dryrun.iam.gserviceaccount.com"}]`), nil
	case contains(cmdStr, "gsutil ls"):
		return []byte("gs://forseti-server-dryrun"), nil
	case contains(cmdStr, "iam service-accounts list --format json --filter email:forseti-server-gcp-*"):
		return []byte(`[{"email": forseti-sa@@forseti-project.iam.gserviceaccount.com"}]`), nil
	case contains(cmdStr, "monitoring channels list"):
		return []byte("[]"), nil
	case contains(cmdStr, "monitoring channels create"):
		return []byte(`{"name": "projects/dryrun/notificationChannels/dryrun"}`), nil
	case contains(cmdStr, "monitoring policies list"):
		return []byte("[]"), nil
	case contains(cmdStr, "compute instances list"):
		return []byte("[]"), nil
	default:
		return nil, nil
	}
}

// CmdCombinedOutput prints the command information.
func (*Fake) CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	log.Printf("Dry run call: %s", strings.Join(cmd.Args, " "))
	switch cmdStr := strings.Join(cmd.Args, " "); {
	case contains(cmdStr, "deployment-manager deployments list", "--format json"):
		return []byte("[]"), nil
	case contains(cmdStr, "monitoring policies list"):
		return []byte(""), nil
	default:
		return nil, nil
	}
}
