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

package main

import (
	"reflect"
	"strings"
	"testing"
)

type testCase struct {
	name, input string
	want        []map[string]string
}

func execute(t *testing.T, tcs []testCase) {
	t.Helper()

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out, err := detectSchema(strings.NewReader(tc.input))
			if err != nil || !reflect.DeepEqual(tc.want, out) {
				t.Errorf("detectSchema() => %v, %v; want %v, nil", out, err, tc.want)
			}
		})
	}
}

func TestDetectSchema_SingleValue(t *testing.T) {
	tcs := []testCase{
		{
			name:  "integer",
			input: "a\n42",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "boolean",
			input: "b\ntrue",
			want: []map[string]string{
				{
					"name": "b",
					"type": "BOOLEAN",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "float",
			input: "c\n3.5",
			want: []map[string]string{
				{
					"name": "c",
					"type": "FLOAT",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "date",
			input: "d\n2015-05-06",
			want: []map[string]string{
				{
					"name": "d",
					"type": "DATE",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "datetime",
			input: "e\n2015-05-06 05:01:01",
			want: []map[string]string{
				{
					"name": "e",
					"type": "DATETIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "datetime-24",
			input: "e\n2015-05-06 19:01:01",
			want: []map[string]string{
				{
					"name": "e",
					"type": "DATETIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "time",
			input: "f\n05:01:01",
			want: []map[string]string{
				{
					"name": "f",
					"type": "TIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "time-24",
			input: "f\n16:01:30",
			want: []map[string]string{
				{
					"name": "f",
					"type": "TIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "string",
			input: "g\ntext",
			want: []map[string]string{
				{
					"name": "g",
					"type": "STRING",
					"mode": "NULLABLE",
				},
			},
		},
	}
	execute(t, tcs)
}

func TestDetectSchema_SingleFieldUpgrade(t *testing.T) {
	tcs := []testCase{
		{
			name:  "integer",
			input: "a\n42\n52",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "boolean",
			input: "b\ntrue\nfalse",
			want: []map[string]string{
				{
					"name": "b",
					"type": "BOOLEAN",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "float",
			input: "c\n3\n4.5\n6.78\n2",
			want: []map[string]string{
				{
					"name": "c",
					"type": "FLOAT",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "date to datetime",
			input: "d\n2015-05-06\n2015-05-06 04:00:00",
			want: []map[string]string{
				{
					"name": "d",
					"type": "DATETIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "time to datetime",
			input: "d\n04:00:00\n2015-05-06 04:00:00",
			want: []map[string]string{
				{
					"name": "d",
					"type": "DATETIME",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "string",
			input: "g\n2015-05-06\nrandom text",
			want: []map[string]string{
				{
					"name": "g",
					"type": "STRING",
					"mode": "NULLABLE",
				},
			},
		},
	}
	execute(t, tcs)
}

func TestDetectSchema_MultipleFields(t *testing.T) {
	tcs := []testCase{
		{
			name:  "single line same type",
			input: "a,b,c\n42,12,2",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "b",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "c",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "multiple lines same types",
			input: "a,b,c\n42,12,2\n52,100,23",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "b",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "c",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "multiple lines multiple types",
			input: "a,b,c\n42,true,2.5\n52,false,23.1",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "b",
					"type": "BOOLEAN",
					"mode": "NULLABLE",
				},
				{
					"name": "c",
					"type": "FLOAT",
					"mode": "NULLABLE",
				},
			},
		},
	}
	execute(t, tcs)
}

func TestDetectSchema_CornerCase(t *testing.T) {
	tcs := []testCase{
		{
			name:  "partially empty field",
			input: "a,b\n,false\n1,true",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
				{
					"name": "b",
					"type": "BOOLEAN",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "empty quotes ignored",
			input: "a\n\"\"\n2",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "empty becomes string",
			input: "a,b\n,1\n,2",
			want: []map[string]string{
				{
					"name": "a",
					"type": "STRING",
					"mode": "NULLABLE",
				},
				{
					"name": "b",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "string with and without quotes",
			input: "a\ntext\n\"\"\n''",
			want: []map[string]string{
				{
					"name": "a",
					"type": "STRING",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "multiple line breaks",
			input: "a\n\n0\n1\n\n",
			want: []map[string]string{
				{
					"name": "a",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "field name has spaces",
			input: "a b c\n1",
			want: []map[string]string{
				{
					"name": "a b c",
					"type": "INTEGER",
					"mode": "NULLABLE",
				},
			},
		},
		{
			name:  "value has spaces",
			input: "a\ntext and text",
			want: []map[string]string{
				{
					"name": "a",
					"type": "STRING",
					"mode": "NULLABLE",
				},
			},
		},
	}
	execute(t, tcs)
}
