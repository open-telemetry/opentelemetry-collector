// Copyright 2019, OpenTelemetry Authors
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

package builder

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/open-telemetry/opentelemetry-service/processor/attributekeyprocessor"
)

func TestGlobalProcessorCfg_InitFromViper(t *testing.T) {
	tests := []struct {
		name string
		file string
		want *AttributesCfg
	}{
		{
			name: "key_mapping",
			file: "./testdata/global_attributes_key_mapping.yaml",
			want: &AttributesCfg{
				KeyReplacements: []attributekeyprocessor.KeyReplacement{
					{
						Key:    "servertracer.http.responsecode",
						NewKey: "http.status_code",
					},
					{
						Key:          "servertracer.http.responsephrase",
						NewKey:       "http.message",
						Overwrite:    true,
						KeepOriginal: true,
					},
				},
			},
		},
		{
			name: "all_settings",
			file: "./testdata/global_attributes_all.yaml",
			want: &AttributesCfg{
				Overwrite: true,
				Values:    map[string]interface{}{"some_string": "hello world"},
				KeyReplacements: []attributekeyprocessor.KeyReplacement{
					{
						Key:    "servertracer.http.responsecode",
						NewKey: "http.status_code",
					},
					{
						Key:          "servertracer.http.responsephrase",
						NewKey:       "http.message",
						Overwrite:    true,
						KeepOriginal: true,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadViperFromFile(tt.file)
			if err != nil {
				t.Fatalf("Failed to load viper from test file: %v", err)
			}

			cfg := NewDefaultMultiSpanProcessorCfg().InitFromViper(v)

			got := cfg.Global.Attributes
			if got == nil {
				t.Fatalf("got nil, want non-nil")
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Mismatched global configuration\n-Got +Want:\n\t%s", diff)
			}
		})
	}
}
