package apply

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCreateCHCDataset(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return nil, nil
	}
	wantArgs := [][]string{{
		"gcloud", "alpha", "healthcare", "datasets", "create", "some_ds", "--location", "some_location", "--project", "some_project"}}
	err := CreateCHCDataset("some_ds", "some_location", "some_project")
	if err != nil {
		t.Fatalf("CreateCHCDataset error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("CreateCHCDataset commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestCreateCHCDatasetAlreadyExist(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return []byte("ERROR: (gcloud.alpha.healthcare.datasets.create) ALREADY_EXISTS: already exists"), fmt.Errorf("some error")
	}
	err := CreateCHCDataset("some_ds", "some_location", "some_project")
	if err != nil {
		t.Fatalf("CreateCHCDataset error: %v", err)
	}
}

func TestCreateCHCDatasetWithError(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return []byte("ERROR: SOME_ERROR"), fmt.Errorf("some error")
	}
	err := CreateCHCDataset("some_ds", "some_location", "some_project")
	if err == nil {
		t.Fatalf("CreateCHCDataset should return error, get nil.")
	}
}

func TestCreateCHCFhirStore(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return nil, nil
	}
	wantArgs := [][]string{{
		"gcloud", "alpha", "healthcare", "fhir-stores", "create", "some_fhir", "--dataset", "some_ds", "--location", "some_location", "--project", "some_project"}}
	err := CreateCHCFhirStore("some_fhir", "some_ds", "some_location", "some_project")
	if err != nil {
		t.Fatalf("CreateCHCFhirStore error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("CreateCHCFhirStore commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestCreateCHCHl7v2Store(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return nil, nil
	}
	wantArgs := [][]string{{
		"gcloud", "alpha", "healthcare", "hl7v2-stores", "create", "some_hl7v2", "--dataset", "some_ds", "--location", "some_location", "--project", "some_project"}}
	err := CreateCHCHl7v2Store("some_hl7v2", "some_ds", "some_location", "some_project")
	if err != nil {
		t.Fatalf("CreateCHCHl7v2Store error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("CreateCHCHl7v2Store commands differ: (-got, +want)\n:%v", diff)
	}
}

func TestCreateCHCDicomStore(t *testing.T) {
	var gotArgs [][]string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = append(gotArgs, cmd.Args)
		return nil, nil
	}
	wantArgs := [][]string{{
		"gcloud", "alpha", "healthcare", "dicom-stores", "create", "some_dicom", "--dataset", "some_ds", "--location", "some_location", "--project", "some_project"}}
	err := CreateCHCDicomStore("some_dicom", "some_ds", "some_location", "some_project")
	if err != nil {
		t.Fatalf("CreateCHCDicomStore error: %v", err)
	}
	if diff := cmp.Diff(gotArgs, wantArgs); len(diff) != 0 {
		t.Fatalf("CreateCHCDicomStore commands differ: (-got, +want)\n:%v", diff)
	}
}
