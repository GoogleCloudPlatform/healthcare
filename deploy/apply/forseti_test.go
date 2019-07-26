package apply

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
)

func TestForseti(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var b []byte
	cmdRun = func(cmd *exec.Cmd) error {
		if cmd.Args[0] == "terraform" && len(b) == 0 {
			var err error
			b, err = ioutil.ReadFile(filepath.Join(cmd.Dir, "main.tf.json"))
			if err != nil {
				t.Fatalf("ioutil.ReadFile = %v", err)
			}
		}
		return nil
	}

	if err := Forseti(conf); err != nil {
		t.Fatalf("Forseti = %v", err)
	}

	wantConfig := `{
	"module": {
		"forseti": {
			"composite_root_resources": [
			  "organizations/12345678",
				"folders/98765321"
			 ],
			 "domain": "my-domain.com",
			 "project_id": "my-project",
			 "source": "./terraform-google-forseti"
		}
	}
}`

	var got, want interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	if err := json.Unmarshal([]byte(wantConfig), &want); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
}

func TestGrantForsetiPermissions(t *testing.T) {
	wantCmdCnt := 9
	wantCmdPrefix := "gcloud projects add-iam-policy-binding project1 --member serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com --role roles/"
	var got []string
	cmdRun = func(cmd *exec.Cmd) error {
		got = append(got, strings.Join(cmd.Args, " "))
		return nil
	}
	if err := GrantForsetiPermissions("project1", "forseti-sa@@forseti-project.iam.gserviceaccount.com"); err != nil {
		t.Fatalf("GrantForsetiPermissions = %v", err)
	}
	if len(got) != wantCmdCnt {
		t.Fatalf("number of permissions granted differ: got %d, want %d", len(got), wantCmdCnt)
	}
	for _, cmd := range got {
		if !strings.HasPrefix(cmd, wantCmdPrefix) {
			t.Fatalf("command %q does not contain expected prefix %q", cmd, wantCmdPrefix)
		}
	}
}
