// Copyright 2018 Google LLC
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

// Package util provides parsing and formatting for the paths used by the REST API.
package util

import (
	"fmt"
	"strings"
)

const (
	// Common URL path components.
	projectsPathComponent  = "projects"
	locationsPathComponent = "locations"
	datasetsPathComponent  = "datasets"
	hl7StoresPathComponent = "hl7Stores"
	messagesPathComponent  = "messages"
)

// GenerateHL7StoreName puts together the components to form the name of a REST HL7 store resource.
func GenerateHL7StoreName(projectReference, locationID, datasetID, hl7StoreID string) string {
	return strings.Join([]string{
		projectsPathComponent,
		projectReference,
		locationsPathComponent,
		locationID,
		datasetsPathComponent,
		datasetID,
		hl7StoresPathComponent,
		hl7StoreID,
	}, "/")
}

// GenerateHL7MessageName puts together the components to form the name of a REST Message resource.
func GenerateHL7MessageName(projectReference, locationID, datasetID, hl7StoreID, messageID string) string {
	return strings.Join([]string{
		GenerateHL7StoreName(projectReference, locationID, datasetID, hl7StoreID),
		messagesPathComponent,
		messageID,
	}, "/")
}

// ParseHL7MessageName parses the project reference, location id, dataset id,
// HL7 store id, and message id from the given resource name.
func ParseHL7MessageName(name string) (string, string, string, string, string, error) {
	parts := strings.Split(name, "/")
	ids, i := []string{}, 0
	allComponents := []string{projectsPathComponent, locationsPathComponent, datasetsPathComponent, hl7StoresPathComponent, messagesPathComponent}
	for _, component := range allComponents {
		if len(component) != 0 {
			if len(parts) <= i || parts[i] != component {
				return "", "", "", "", "", fmt.Errorf("expected component %v at position %v in %v", component, i, parts)
			}
			i++
		}
		if len(parts) <= i {
			return "", "", "", "", "", fmt.Errorf("expected a component at position %v in %v", i, parts)
		}
		ids = append(ids, parts[i])
		i++
	}
	if len(parts[i:]) > 0 {
		return "", "", "", "", "", fmt.Errorf("unexpected tokens %v in %v", parts[i:], name)
	}

	return ids[0], ids[1], ids[2], ids[3], ids[4], nil
}
