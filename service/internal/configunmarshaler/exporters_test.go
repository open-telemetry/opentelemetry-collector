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

package configunmarshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestExportersUnmarshal(t *testing.T) {
	factories, err := exporter.MakeFactoryMap(exportertest.NewNopFactory())
	require.NoError(t, err)

	exps := NewExporters(factories)
	conf := confmap.NewFromStringMap(map[string]interface{}{
		"nop":            nil,
		"nop/myexporter": nil,
	})
	require.NoError(t, exps.Unmarshal(conf))

	cfgWithName := factories["nop"].CreateDefaultConfig()
	cfgWithName.SetIDName("myexporter") //nolint:staticcheck
	assert.Equal(t, map[component.ID]component.Config{
		component.NewID("nop"):                       factories["nop"].CreateDefaultConfig(),
		component.NewIDWithName("nop", "myexporter"): cfgWithName,
	}, exps.GetExporters())
}

func TestExportersUnmarshalError(t *testing.T) {
	var testCases = []struct {
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectedError string
	}{
		{
			name: "invalid-exporter-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":     nil,
				"/custom": nil,
			}),
			expectedError: "the part before / should not be empty",
		},
		{
			name: "invalid-exporter-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":  nil,
				"nop/": nil,
			}),
			expectedError: "the part after / should not be empty",
		},
		{
			name: "unknown-exporter-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nosuchexporter": nil,
			}),
			expectedError: "unknown exporters type: \"nosuchexporter\"",
		},
		{
			name: "duplicate-exporter",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop /exp ": nil,
				" nop/ exp": nil,
			}),
			expectedError: "duplicate name",
		},
		{
			name: "invalid-exporter-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": map[string]interface{}{
					"unknown_section": "exporter",
				},
			}),
			expectedError: "error reading exporters configuration for \"nop\"",
		},
		{
			name: "invalid-exporter-sub-config",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": "tests",
			}),
			expectedError: "'[nop]' expected a map, got 'string'",
		},
	}

	factories, err := exporter.MakeFactoryMap(exportertest.NewNopFactory())
	assert.NoError(t, err)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			exps := NewExporters(factories)
			err = exps.Unmarshal(tt.conf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
