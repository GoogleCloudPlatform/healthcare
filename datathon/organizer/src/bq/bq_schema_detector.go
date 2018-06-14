// Copyright 2018 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// bq_schema_detector scans all data in the given csv file and outputs the corresponding BigQuery
// schema.
// Usage:
//     go run bq_schema_detector.go -input_csv=${file} [-output_file=${file}]
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	attrName     = "name"
	attrType     = "type"
	attrMode     = "mode"
	typeInteger  = "INTEGER"
	typeBoolean  = "BOOLEAN"
	typeFloat    = "FLOAT"
	typeString   = "STRING"
	typeDate     = "DATE"
	typeDatetime = "DATETIME"
	typeTime     = "TIME"
	modeNullable = "NULLABLE"
)

var (
	// The following layouts are known patterns in the MIMIC3 and eICU datasets. To support more
	// formats, please add patterns to the corresponding layout list.
	dateLayouts = []string{
		"2006-01-02",
	}
	timeLayouts = []string{
		"15:04:05",
	}
	datetimeLayouts = []string{
		"2006-01-02 15:04:05",
	}
	inputCSV   = flag.String("input_csv", "", "The input csv file.")
	outputFile = flag.String("output_file", "", "The output file.")
)

func inferType(value string) string {
	if value == "" {
		// Return empty type for empty value.
		return ""
	}
	// Try to convert to integer.
	if _, err := strconv.Atoi(value); err == nil {
		return typeInteger
	}
	// Try to convert to float.
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return typeFloat
	}
	// Try to convert to boolean.
	if _, err := strconv.ParseBool(value); err == nil {
		return typeBoolean
	}
	// Try to convert to date, datetime, and timestamp.
	for _, layout := range dateLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return typeDate
		}
	}
	for _, layout := range timeLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return typeTime
		}
	}
	for _, layout := range datetimeLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return typeDatetime
		}
	}
	return typeString
}

func upgradeType(t1, t2 string) string {
	switch {
	case t1 == t2:
		return t1
	case t1 == "":
		return t2
	case t2 == "":
		return t1
	case t1 == typeInteger && t2 == typeFloat || t1 == typeFloat && t2 == typeInteger:
		return typeFloat
	case t1 == typeDate && t2 == typeDatetime || t1 == typeDatetime && t2 == typeDate:
		return typeDatetime
	case t1 == typeTime && t2 == typeDatetime || t1 == typeDatetime && t2 == typeTime:
		return typeDatetime
	default:
		return typeString
	}
}

func buildSchema(fields []string, schema []map[string]string) ([]map[string]string, error) {
	if len(fields) != len(schema) {
		return nil, fmt.Errorf("field size and schema size don't match; field size equals %d while schema size equals %d", len(fields), len(schema))
	}
	for i, field := range fields {
		if schema[i][attrType] == typeString {
			// STRING is already the most generic type. No need to
			// parse the field data.
			continue
		}
		ft := inferType(field)
		schema[i][attrType] = upgradeType(ft, schema[i][attrType])
	}
	return schema, nil
}

func detectSchema(r io.Reader) ([]map[string]string, error) {
	var schema []map[string]string
	reader := csv.NewReader(r)
	// The first line is for field names.
	records, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for _, name := range records {
		schema = append(schema, map[string]string{attrName: name, attrMode: modeNullable})
	}
	for {
		records, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		schema, err = buildSchema(records, schema)
		if err != nil {
			return nil, err
		}
	}
	for _, e := range schema {
		if e[attrType] == "" {
			// No value for this field in the entire file.
			e[attrType] = typeString
		}
	}
	return schema, nil
}

func main() {
	flag.Parse()
	if *inputCSV == "" {
		log.Fatal("Must specify input csv file; usage: go run bq_schema_detector.go -input_csv=${file} [-output_file=${file}]")
	}
	file, err := os.Open(*inputCSV)
	if err != nil {
		log.Fatalf("Failed to open input csv file due to error: %v", err)
	}
	defer file.Close()
	schema, err := detectSchema(file)
	if err != nil {
		log.Fatalf("Failed to detect schema due to error: %v", err)
	}
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize the schema due to error: %v", err)
	}
	if *outputFile == "" {
		// No output file specified, just print the schema on console.
		log.Print(string(output))
		return
	}
	if err := ioutil.WriteFile(*outputFile, output, 0644); err != nil {
		log.Fatalf("Failed to write schema to file due to error: %v", err)
	}
}
