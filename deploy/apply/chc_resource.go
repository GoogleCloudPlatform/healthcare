package apply

import (
	"fmt"
	"os/exec"
	"strings"

	"google3/base/go/log"
)

// must enable healthcare.googleapis.com TODO @ruihuang

// ExpectedCHCResourceName create a name of prefix for a CHC resource
func ExpectedCHCResourceName(datasetID, location, projectID string) string {
	return "projects/" + projectID + "/locations/" + location + "/datasets/" + datasetID
}

// CHCStoresCreationCmd create an command that creates a CHC resource
func CHCStoresCreationCmd(datasetID, location, projectID, resourceType, resourceID string) *exec.Cmd {
	return exec.Command("gcloud", "alpha", "healthcare", resourceType, "create", resourceID, "--dataset", datasetID, "--location", location, "--project", projectID)
}

// CreateCHCDataset create a CHC dataset.
func CreateCHCDataset(datasetID, location, projectID string) error {
	expectedDatasetName := ExpectedCHCResourceName(datasetID, location, projectID)
	resourceType := "datasets"
	cmd := exec.Command("gcloud", "alpha", "healthcare", resourceType, "create", datasetID, "--location", location, "--project", projectID)
	return CreateCHCResource(expectedDatasetName, resourceType, cmd)
}

// CreateCHCFhirStore create a CHC FHIR store.
func CreateCHCFhirStore(fhirStoreID, datasetID, location, projectID string) error {
	expectedFhirStoreName := ExpectedCHCResourceName(datasetID, location, projectID) + "/fhirStores/" + fhirStoreID
	resourceType := "fhir-stores"
	cmd := CHCStoresCreationCmd(datasetID, location, projectID, resourceType, fhirStoreID)
	return CreateCHCResource(expectedFhirStoreName, resourceType, cmd)
}

// CreateCHCHl7v2Store create a CHC HL7v2 store.
func CreateCHCHl7v2Store(Hl7v2StoreID, datasetID, location, projectID string) error {
	expectedHl7v2StoreName := ExpectedCHCResourceName(datasetID, location, projectID) + "/hl7V2Stores/" + Hl7v2StoreID
	resourceType := "hl7v2-stores"
	cmd := CHCStoresCreationCmd(datasetID, location, projectID, resourceType, Hl7v2StoreID)
	return CreateCHCResource(expectedHl7v2StoreName, resourceType, cmd)
}

// CreateCHCDicomStore create a CHC Dicom store.
func CreateCHCDicomStore(DicomStoreID, datasetID, location, projectID string) error {
	expectedDicomStoreName := ExpectedCHCResourceName(datasetID, location, projectID) + "/dicomStores/" + DicomStoreID
	resourceType := "dicom-stores"
	cmd := CHCStoresCreationCmd(datasetID, location, projectID, resourceType, DicomStoreID)
	return CreateCHCResource(expectedDicomStoreName, resourceType, cmd)
}

// CreateCHCResource create a CHC resource.
func CreateCHCResource(expectedResourceName, resourceType string, createCmd *exec.Cmd) error {
	cmdOutput, err := cmdCombinedOutput(createCmd)
	strOutput := string(cmdOutput)
	if err != nil {
		splitOutput := strings.Split(strOutput, "\n")
		strAlreadyExist := fmt.Sprintf("ERROR: (gcloud.alpha.healthcare.%v.create) ALREADY_EXISTS: already exists", resourceType)
		if splitOutput[0] == strAlreadyExist {
			log.Warningf("%q already exists.", expectedResourceName)
			err = nil
		} else {
			log.Errorf("Failed to create %q.\n%v", expectedResourceName, strOutput)
		}
	} else {
		log.Infof("Create %q successfully.", expectedResourceName)
	}
	return err
}
