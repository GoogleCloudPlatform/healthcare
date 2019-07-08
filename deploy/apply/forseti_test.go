package apply

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"path/filepath"
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
