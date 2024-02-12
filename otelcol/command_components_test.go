// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
)

var nopType = component.MustNewType("nop")

func TestNewBuildSubCommand(t *testing.T) {
	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      nopFactories,
		ConfigProvider: cfgProvider,
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"components"})

	ExpectedYamlStruct := componentsOutput{
		BuildInfo: component.NewDefaultBuildInfo(),
		Receivers: []componentWithStability{{
			Name: nopType,
			Stability: map[string]string{
				"logs":    "Stable",
				"metrics": "Stable",
				"traces":  "Stable",
			},
		}},
		Processors: []componentWithStability{{
			Name: nopType,
			Stability: map[string]string{
				"logs":    "Stable",
				"metrics": "Stable",
				"traces":  "Stable",
			},
		}},
		Exporters: []componentWithStability{{
			Name: nopType,
			Stability: map[string]string{
				"logs":    "Stable",
				"metrics": "Stable",
				"traces":  "Stable",
			},
		}},
		Connectors: []componentWithStability{{
			Name: nopType,
			Stability: map[string]string{
				"logs-to-logs":    "Development",
				"logs-to-metrics": "Development",
				"logs-to-traces":  "Development",

				"metrics-to-logs":    "Development",
				"metrics-to-metrics": "Development",
				"metrics-to-traces":  "Development",

				"traces-to-logs":    "Development",
				"traces-to-metrics": "Development",
				"traces-to-traces":  "Development",
			},
		}},
		Extensions: []componentWithStability{{
			Name: nopType,
			Stability: map[string]string{
				"extension": "Stable",
			},
		}},
	}
	ExpectedOutput, err := yaml.Marshal(ExpectedYamlStruct)
	require.NoError(t, err)

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err = cmd.Execute()
	require.NoError(t, err)

	// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
	// line that makes the test fail.
	assert.Equal(t, strings.Trim(string(ExpectedOutput), "\n"), strings.Trim(b.String(), "\n"))
}
