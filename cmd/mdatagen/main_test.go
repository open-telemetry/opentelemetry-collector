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
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	validMetadata = `
name: metricreceiver
metrics:
  system.cpu.time:
    description: Total CPU seconds broken down by different states.
    unit: s
    data:
      type: double sum
      aggregation: cumulative
    labels: []
`
	allDataTypes = `
name: metricreceiver
metrics:
  int.gauge:
    description: int gauge type
    unit: s
    data:
      type: int gauge
  int.sum:
    description: int sum type
    unit: s
    data:
      type: int sum
      monotonic: true
      aggregation: delta
  int.histogram:
    description: int histogram type
    unit: s
    data:
      type: int histogram
      aggregation: cumulative
  double.gauge:
    description: double gauge type
    unit: s
    data:
      type: double gauge
  double.sum:
    description: double sum type
    unit: s
    data:
      type: double sum
      monotonic: true
      aggregation: delta
  double.histogram:
    description: double histogram type
    unit: s
    data:
      type: double histogram
      aggregation: cumulative
`
)

func Test_runContents(t *testing.T) {
	type args struct {
		yml string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr string
	}{
		{
			name: "valid metadata",
			args: args{validMetadata},
			want: "",
		},
		{
			name:    "invalid yaml",
			args:    args{"invalid"},
			want:    "",
			wantErr: "cannot unmarshal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir, err := ioutil.TempDir("", "metadata-test-*")
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, os.RemoveAll(tmpdir))
			})

			metadataFile := path.Join(tmpdir, "metadata.yaml")
			require.NoError(t, ioutil.WriteFile(metadataFile, []byte(tt.args.yml), 0666))

			err = run(metadataFile)

			if tt.wantErr != "" {
				require.Regexp(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
				require.FileExists(t, path.Join(tmpdir, "internal/metadata/generated_metrics.go"))
			}
		})
	}
}

func Test_run(t *testing.T) {
	type args struct {
		ymlPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "no argument",
			args:    args{""},
			wantErr: true,
		},
		{
			name:    "no such file",
			args:    args{"/no/such/file"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(tt.args.ymlPath); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 		{
//			name: "all data types",
//			yml:  allDataTypes,
//			want: metadata{Name: "metricreceiver",
//				Metrics: map[metricName]metric{
//					"double.gauge": {Description: "double gauge type", Unit: "s",
//						Data: &doubleGauge{}},
//					"double.histogram": {Description: "double histogram type", Unit: "s",
//						Data: &doubleHistogram{Aggregated{Aggregation: "cumulative"}}},
//					"double.sum": {Description: "double sum type", Unit: "s",
//						Data: &doubleSum{
//							Aggregated: Aggregated{Aggregation: "delta"},
//							Mono:       Mono{Monotonic: true},
//						}},
//					"int.gauge": {Description: "int gauge type", Unit: "s",
//						Data: &intGauge{}},
//					"int.histogram": {Description: "int histogram type", Unit: "s",
//						Data: &intHistogram{Aggregated{Aggregation: "cumulative"}}},
//					"int.sum": {Description: "int sum type", Unit: "s",
//						Data: &intSum{
//							Aggregated: Aggregated{Aggregation: "delta"},
//							Mono:       Mono{Monotonic: true},
//						}}}},
//		},
