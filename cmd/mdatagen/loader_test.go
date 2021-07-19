// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	allOptions = `
name: metricreceiver
labels:
  freeFormLabel:
    description: Label that can take on any value.

  freeFormLabelWithValue:
    value: state
    description: Label that has alternate value set.

  enumLabel:
    description: Label with a known set of values.
    enum: [red, green, blue]

metrics:
  system.cpu.time:
    description: Total CPU seconds broken down by different states.
    unit: s
    data:
      type: sum
      monotonic: true
      aggregation: cumulative
    labels: [freeFormLabel, freeFormLabelWithValue, enumLabel]
`

	unknownMetricLabel = `
name: metricreceiver
metrics:
  system.cpu.time:
    description: Total CPU seconds broken down by different states.
    unit: s
    data:
      type: sum
      monotonic: true
      aggregation: cumulative
    labels: [missing]
`
	unknownMetricType = `
name: metricreceiver
metrics:
  system.cpu.time:
    description: Total CPU seconds broken down by different states.
    unit: s
    data:
      type: invalid
    labels:
`
)

func Test_loadMetadata(t *testing.T) {
	tests := []struct {
		name    string
		yml     string
		want    metadata
		wantErr string
	}{
		{
			name: "all options",
			yml:  allOptions,
			want: metadata{
				Name: "metricreceiver",
				Labels: map[labelName]label{
					"enumLabel": {
						Description: "Label with a known set of values.",
						Value:       "",
						Enum:        []string{"red", "green", "blue"}},
					"freeFormLabel": {
						Description: "Label that can take on any value.",
						Value:       ""},
					"freeFormLabelWithValue": {
						Description: "Label that has alternate value set.",
						Value:       "state"}},
				Metrics: map[metricName]metric{
					"system.cpu.time": {
						Description: "Total CPU seconds broken down by different states.",
						Unit:        "s",
						Data: &sum{
							Aggregated: Aggregated{Aggregation: "cumulative"},
							Mono:       Mono{Monotonic: true},
						},
						// YmlData: nil,
						Labels: []labelName{"freeFormLabel", "freeFormLabelWithValue", "enumLabel"}}},
			},
		},
		{
			name:    "unknown metric label",
			yml:     unknownMetricLabel,
			want:    metadata{},
			wantErr: "error validating struct:\n\tmetadata.Metrics[system.cpu.time].Labels[missing]: unknown label value\n",
		},
		{
			name:    "unknownMetricType",
			yml:     unknownMetricType,
			want:    metadata{},
			wantErr: `unable to unmarshal yaml: metric data "invalid" type invalid`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadMetadata([]byte(tt.yml))
			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
