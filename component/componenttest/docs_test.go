// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componenttest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsComponentImport(t *testing.T) {
	type args struct {
		importStr             string
		importPrefixesToCheck []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match",
			args: args{
				importStr: "matching/prefix",
				importPrefixesToCheck: []string{
					"some/prefix",
					"matching/prefix",
				},
			},
			want: true,
		},
		{
			name: "No match",
			args: args{
				importStr: "some/prefix",
				importPrefixesToCheck: []string{
					"expecting/prefix",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isComponentImport(tt.args.importStr, tt.args.importPrefixesToCheck); got != tt.want {
				t.Errorf("isComponentImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetImportPrefixesToCheck(t *testing.T) {
	tests := []struct {
		name   string
		module string
		want   []string
	}{
		{
			name:   "Get import prefixes - 1",
			module: "test",
			want: []string{
				"test/extension",
				"test/receiver",
				"test/processor",
				"test/exporter",
			},
		},
		{
			name:   "Get import prefixes - 2",
			module: "test/",
			want: []string{
				"test/extension",
				"test/receiver",
				"test/processor",
				"test/exporter",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getImportPrefixesToCheck(tt.module); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getImportPrefixesToCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckDocs(t *testing.T) {
	type args struct {
		projectPath                   string
		relativeDefaultComponentsPath string
		projectGoModule               string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid project path",
			args: args{
				projectPath:                   "invalid/project",
				relativeDefaultComponentsPath: "invalid/file",
				projectGoModule:               "go.opentelemetry.io/collector",
			},
			wantErr: true,
		},
		{
			name: "Valid files",
			args: args{
				projectPath:                   getProjectPath(t),
				relativeDefaultComponentsPath: "service/defaultcomponents/defaults.go",
				projectGoModule:               "go.opentelemetry.io/collector",
			},
			wantErr: false,
		},
		{
			name: "Invalid files",
			args: args{
				projectPath:                   getProjectPath(t),
				relativeDefaultComponentsPath: "service/defaultcomponents/invalid.go",
				projectGoModule:               "go.opentelemetry.io/collector",
			},
			wantErr: true,
		},
		{
			name: "Invalid imports",
			args: args{
				projectPath:                   getProjectPath(t),
				relativeDefaultComponentsPath: "component/componenttest/testdata/invalid_go.txt",
				projectGoModule:               "go.opentelemetry.io/collector",
			},
			wantErr: true,
		},
		{
			name: "README does not exist",
			args: args{
				projectPath:                   getProjectPath(t),
				relativeDefaultComponentsPath: "component/componenttest/testdata/valid_go.txt",
				projectGoModule:               "go.opentelemetry.io/collector",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckDocs(tt.args.projectPath, tt.args.relativeDefaultComponentsPath, tt.args.projectGoModule); (err != nil) != tt.wantErr {
				t.Errorf("CheckDocs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getProjectPath(t *testing.T) string {
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory: %v")

	// Absolute path to the project root directory
	projectPath := filepath.Join(wd, "../../")

	return projectPath
}
