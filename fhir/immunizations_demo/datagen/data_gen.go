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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pborman/uuid"
)

const (
	travelGap  = 5
	year       = time.Hour * 24 * 365
	dateFormat = "2006-01-02"
)

type jObject map[string]interface{}
type patientData struct {
	patient    jObject
	trips      []jObject
	conditions []jObject
}

var (
	outputPath = flag.String("output_path", "", "The output path")
	patientNum = flag.Int("num", 5000, "Number of patients to generate")

	// 20 largest countries by population.
	countries = []string{
		"China",
		"India",
		"United States",
		"Indonesia",
		"Brazil",
		"Pakistan",
		"Bangladesh",
		"Nigeria",
		"Russia",
		"Japan",
		"Mexico",
		"Philippines",
		"Vietnam",
		"Ethiopia",
		"Germany",
		"Egypt",
		"Turkey",
		"Iran",
		"Democratic Republic of the Congo",
		"Thailand",
	}
	diseases = []string{
		"Hepatitis",
		"Measles",
		"Meningitis",
		"Yellow Fever",
	}
	riskFactors = generateRiskFactors()
	diseaseProb = generateDiseaseProb()
)

func generateRiskFactors() map[string]map[string]float32 {
	rf := map[string]map[string]float32{}
	for _, c := range countries {
		rfc := map[string]float32{}
		for _, d := range diseases {
			rfc[d] = 1.0 + rand.Float32()
		}
		rf[c] = rfc
	}
	return rf
}

func generateDiseaseProb() map[string]map[string]map[string]float32 {
	dp := map[string]map[string]map[string]float32{}
	for _, d := range diseases {
		dpg := map[string]map[string]float32{}
		for _, g := range []string{"M", "F"} {
			dpa := map[string]float32{}
			// Disease probability by genders by age group [0,15), [15, 65), [65, max)
			dpa["M"] = rand.Float32() / 4
			dpa["Y"] = dpa["M"] + 0.25
			dpa["O"] = dpa["M"] + 0.2
			dpg[g] = dpa
		}
		dp[d] = dpg
	}
	return dp
}

func generateCondition(disease, pID, tID string, date time.Time) jObject {
	return jObject{
		"resourceType":   "Condition",
		"id":             uuid.New(),
		"clinicalStatus": "resolved",
		"subject": jObject{
			"reference": fmt.Sprintf("Patient/%s", pID),
		},
		"onsetDateTime":     date.UTC().Format(time.RFC3339),
		"abatementDateTime": date.Add(time.Hour * 24 * 14).UTC().Format(time.RFC3339),
		"evidence": []jObject{
			{
				"detail": []jObject{
					jObject{"reference": fmt.Sprintf("QuestionnaireResponse/%s", tID)},
				},
			},
		},
		"code": jObject{
			"coding": []jObject{
				{
					"system": "http://hl7.org/fhir/v3/ConditionCode",
					"code":   disease,
				},
			},
		},
	}
}

func generateConditions(p jObject, trips []jObject) []jObject {
	conditions := []jObject{}
	birthday, _ := time.Parse(dateFormat, p["birthDate"].(string))
	for _, trip := range trips {
		var ag string
		departureDate, _ := time.Parse(dateFormat, trip["item"].([]jObject)[1]["item"].([]jObject)[0]["answer"].([]jObject)[0]["valueDate"].(string))
		returnDate, _ := time.Parse(dateFormat, trip["item"].([]jObject)[1]["item"].([]jObject)[1]["answer"].([]jObject)[0]["valueDate"].(string))
		age := departureDate.Sub(birthday) / year
		if age < 15 {
			ag = "Y"
		} else if age < 65 {
			ag = "M"
		} else {
			ag = "O"
		}
		country := trip["item"].([]jObject)[0]["answer"].([]jObject)[0]["valueString"].(string)
		var gender string
		if p["gender"].(string) == "male" {
			gender = "M"
		} else {
			gender = "F"
		}
		pID := p["id"].(string)
		tID := trip["id"].(string)

		for _, d := range diseases {
			r := riskFactors[country][d] * diseaseProb[d][gender][ag] * float32(returnDate.Sub(departureDate).Hours()/(5*24)) // time factor.
			if rand.Float32() <= r {
				onsetDateTime := returnDate.Add(time.Hour * 24 * time.Duration(rand.Intn(7)-3))
				conditions = append(conditions, generateCondition(d, pID, tID, onsetDateTime))
			}
		}
	}
	return conditions
}

func generateTravelPlans(p jObject, qID string) []jObject {
	var responses []jObject
	birthday, _ := time.Parse(dateFormat, p["birthDate"].(string))
	now := time.Now()
	pID := p["id"].(string)
	// Randomly add travel history in this person's lifetime.
	for cursor := birthday; cursor.Before(now); cursor = cursor.Add(year * time.Duration(rand.Intn(travelGap)+1)) {
		country := countries[rand.Intn(len(countries))]
		responses = append(responses, jObject{
			"resourceType": "QuestionnaireResponse",
			"id":           uuid.New(),
			"author": jObject{
				"reference": fmt.Sprintf("Patient/%s", pID),
			},
			"questionnaire": jObject{
				"reference": fmt.Sprintf("Questionnaire/%s", qID),
			},
			"status": "completed",
			"subject": jObject{
				"reference": fmt.Sprintf("Patient/%s", pID),
			},
			"item": []jObject{
				{
					"answer": []jObject{
						{"valueString": country},
					},
					"linkId": "1",
				},
				{
					"linkId": "2",
					"item": []jObject{
						{
							"answer": []jObject{
								{"valueDate": cursor.Format(dateFormat)},
							},
							"linkId": "2.1",
						},
						{
							"answer": []jObject{
								{"valueDate": cursor.Add(time.Hour * 24 * time.Duration(rand.Intn(14) + 1)).Format(dateFormat)},
							},
							"linkId": "2.2",
						},
					},
				},
			},
		})
	}

	return responses
}

func generatePatientData(qID string) patientData {
	var gender string
	if rand.Float32() > 0.5 {
		gender = "male"
	} else {
		gender = "female"
	}
	p := jObject{
		"resourceType": "Patient",
		"id":           uuid.New(),
		// Randomly set birthday to within 90 years old.
		"birthDate": time.Now().Add(-time.Duration(rand.Intn(90)) * year).Format(dateFormat),
		"gender":    gender,
	}

	// Add travel places.
	travelPlans := generateTravelPlans(p, qID)
	// Simulate getting sick.
	conditions := generateConditions(p, travelPlans)
	log.Printf("Patient %s traveled to %d places and got sick %d times", p["id"], len(travelPlans), len(conditions))

	return patientData{
		patient:    p,
		trips:      travelPlans,
		conditions: conditions,
	}
}

func generatePatientBundle(data patientData) []byte {
	// Put everything in a bundle.
	entries := make([]jObject, 0, 1+len(data.conditions)+len(data.trips))
	entries = append(entries, jObject{"resource": data.patient})
	for _, cd := range data.conditions {
		entries = append(entries, jObject{"resource": cd})
	}
	for _, tp := range data.trips {
		entries = append(entries, jObject{"resource": tp})
	}

	bundle := jObject{
		"resourceType": "Bundle",
		"entry":        entries,
	}

	jsonData, err := json.Marshal(bundle)
	if err != nil {
		log.Printf("Failed to convert resource to string; %v", err)
	}
	return jsonData
}

// Generates the Questionnaire that all the travel plans will refer to.
// Return the questionnaire ID and the json data.
func generateQuestionnaireBundle() (string, []byte) {
	qID := uuid.New()
	questionnaire := jObject{
		"resourceType": "Questionnaire",
		"id":           qID,
		"identifier": []jObject{
			jObject{
				"system": "internal",
				"value":  "travel-questionnaire",
			},
		},
		"title":  "Travel Intention",
		"status": "active",
		"subjectType": []string{
			"Patient",
		},
		"item": []jObject{
			{
				"linkId": "1",
				"text":   "What country will you be travelling to?",
				"type":   "string",
			},
			{
				"linkId": "2",
				"text":   "What dates will you be travelling on?",
				"type":   "group",
				"item": []jObject{
					{
						"linkId": "2.1",
						"text":   "Departure",
						"type":   "date",
					},
					{
						"linkId": "2.2",
						"text":   "Return",
						"type":   "date",
					},
				},
			},
		},
	}
	bundle := jObject{
		"resourceType": "Bundle",
		"entry": []jObject{
			{"resource": questionnaire},
		},
	}
	data, err := json.Marshal(bundle)
	if err != nil {
		log.Printf("Failed to convert resource to string; %v", err)
	}
	return qID, data
}

func generateDemoPatientBundle(qID string) []byte {
	data := generatePatientData(qID)
	data.patient["identifier"] = []jObject{
		{"value": "demo-patient"},
	}
	data.patient["name"] = []jObject{
		{
			"family": "Smith",
			"given": []string{
				"Taylor",
			},
			"use": "official",
		},
	}
	data.patient["telecom"] = []jObject{
		{
			"value": "(555) 555-5555",
		},
	}
	return generatePatientBundle(data)
}

func main() {
	flag.Parse()

	f, err := os.Create(*outputPath)
	defer f.Close()
	if err != nil {
		log.Fatalf("Failed to create output file %s; %v", *outputPath, err)
	}

	qID, questionnaireData := generateQuestionnaireBundle()
	if _, err := f.Write(questionnaireData); err != nil {
		log.Fatalf("Failed to write data; %v", err)
	}
	f.WriteString("\n")

	if _, err := f.Write(generateDemoPatientBundle(qID)); err != nil {
		log.Fatalf("Failed to write data; %v", err)
	}
	f.WriteString("\n")

	for i := 0; i < *patientNum-1; i++ {
		if _, err := f.Write(generatePatientBundle(generatePatientData(qID))); err != nil {
			log.Fatalf("Failed to write data; %v", err)
		}
		f.WriteString("\n")
	}
	log.Println("All Done!")
}
