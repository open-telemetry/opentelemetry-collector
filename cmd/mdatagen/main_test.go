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
      type: sum
      aggregation: cumulative
    labels: []
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
			require.NoError(t, ioutil.WriteFile(metadataFile, []byte(tt.args.yml), 0600))

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
