package apply

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// Forseti applies the forseti config, if it exists.
func Forseti(conf *config.Config) error {
	if conf.Forseti == nil {
		log.Println("no forseti config, nothing to do")
		return nil
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	runCmd := func(bin string, args ...string) error {
		cmd := exec.Command(bin, args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmdRun(cmd)
	}

	// TODO: use registry instead of cloning repo
	if err := runCmd("git", "clone", "https://github.com/forseti-security/terraform-google-forseti"); err != nil {
		return fmt.Errorf("failed to clone forseti module repo: %v", err)
	}

	runTerraformCommand := func(args ...string) error {
		return runCmd("terraform", args...)
	}

	tfConf := map[string]interface{}{
		"module": map[string]interface{}{
			"forseti": &config.TerraformModule{
				Source:     "./terraform-google-forseti",
				Properties: conf.Forseti.Properties,
			},
		},
	}

	b, err := json.MarshalIndent(tfConf, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal terraform config: %v", err)
	}

	log.Printf("forseti terraform config:\n%v", string(b))

	// drw-r--r--
	if err := ioutil.WriteFile(filepath.Join(dir, "main.tf.json"), b, 0644); err != nil {
		return fmt.Errorf("failed to write terraform config: %v", err)
	}

	if err := runTerraformCommand("init"); err != nil {
		return fmt.Errorf("failed to init terraform dir: %v", err)
	}
	if err := runTerraformCommand("plan", "--out", "tfplan"); err != nil {
		return fmt.Errorf("failed to create terraform plan: %v", err)
	}
	if err := runTerraformCommand("apply", "tfplan"); err != nil {
		return fmt.Errorf("failed to apply plan: %v", err)
	}
	return nil
}
