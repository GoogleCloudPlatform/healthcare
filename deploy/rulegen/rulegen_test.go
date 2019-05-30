package rulegen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestRunOutputPath(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir = %v", err)
	}

	if err := Run(conf, tmpDir); err != nil {
		t.Fatalf("Run = %v", err)
	}

	checkRulesDir(t, tmpDir)
}

func TestRunServerBucket(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var gotArgs []string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		if len(gotArgs) != 0 {
			return nil, errors.New("fake CombinedOutput: unexpectedly called more than once")
		}

		gotArgs = cmd.Args
		if len(gotArgs) != 4 {
			return nil, fmt.Errorf("fake CombinedOutput: unexpected number of args: got %d, want 4, %v ", len(cmd.Args), cmd.Args)
		}
		checkRulesDir(t, filepath.Dir(gotArgs[2]))
		return nil, nil
	}
	if err := Run(conf, ""); err != nil {
		t.Fatalf("Run = %v", err)
	}

	wantRE, err := regexp.Compile(`gsutil cp .*\*\.yaml gs://my-forseti-project-server/rules`)
	if err != nil {
		t.Fatalf("regexp.Compile = %v", err)
	}
	got := strings.Join(gotArgs, " ")
	if !wantRE.MatchString(got) {
		t.Fatalf("rules upload command does not match: got %q, want match of %q", got, wantRE)
	}
}

func checkRulesDir(t *testing.T, rulesDir string) {
	t.Helper()

	// check one rules file
	b, err := ioutil.ReadFile(filepath.Join(rulesDir, "audit_logging_rules.yaml"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile = %v", err)
	}

	wantYAML := `
rules:
- name: Require all Cloud Audit logs.
  resource:
  - type: project
    resource_ids:
    - '*'
  service: allServices
  log_types:
  - ADMIN_READ
  - DATA_READ
  - DATA_WRITE
`
	got := make(map[string]interface{})
	want := make(map[string]interface{})

	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("audit logging rules differ (-got, +want):\n%v", diff)
	}
}
