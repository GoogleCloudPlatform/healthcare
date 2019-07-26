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

// Standard (built in) roles required by the Forseti service account on projects to be monitored.
// This list includes project level roles from
// https://github.com/forseti-security/terraform-google-forseti/blob/master/modules/server/main.tf#L63
// In the future, have a deeper integration with Forseti module and reuse the role list.
var forsetiStandardRoles = [...]string{
	"appengine.appViewer",
	"bigquery.metadataViewer",
	"browser",
	"cloudasset.viewer",
	"cloudsql.viewer",
	"compute.networkViewer",
	"iam.securityReviewer",
	"servicemanagement.quotaViewer",
	"serviceusage.serviceUsageConsumer",
}

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

// GrantForsetiPermissions grants all necessary permissions to the given Forseti service account in the project.
// TODO Use Terraform to deploy these.
func GrantForsetiPermissions(projectID, serviceAccount string) error {
	for _, r := range forsetiStandardRoles {
		if err := addBinding(projectID, serviceAccount, fmt.Sprintf("roles/%s", r)); err != nil {
			return fmt.Errorf("failed to grant all necessary permissions to Forseti service account %q in project %q: %v", serviceAccount, projectID, err)
		}
	}
	return nil
}

// addBinding adds an IAM policy binding for the given service account for the given role.
func addBinding(projectID, serviceAccount, role string) error {
	cmd := exec.Command(
		"gcloud", "projects", "add-iam-policy-binding",
		projectID,
		"--member", fmt.Sprintf("serviceAccount:%s", serviceAccount),
		"--role", role,
	)
	if err := cmdRun(cmd); err != nil {
		return fmt.Errorf("failed to add iam policy binding for service account %q for role %q: %v", serviceAccount, role, err)
	}
	return nil
}
