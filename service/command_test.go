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

package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/featuregate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestNewCommandVersion(t *testing.T) {
	cmd := NewCommand(CollectorSettings{BuildInfo: component.BuildInfo{Version: "test_version"}})
	assert.Equal(t, "test_version", cmd.Version)
}

func TestNewCommandNoConfigURI(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cmd := NewCommand(CollectorSettings{Factories: factories})
	require.Error(t, cmd.Execute())
}

func TestNewCommandInvalidComponent(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-invalid.yaml")}))
	require.NoError(t, err)

	cmd := NewCommand(CollectorSettings{Factories: factories, ConfigProvider: cfgProvider})
	require.Error(t, cmd.Execute())
}

func TestBuildSubCommand(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"build-info"})

	ExpectedYamlStruct := componentsOutput{
		Version:    "latest",
		Receivers:  []config.Type{"nop"},
		Processors: []config.Type{"nop"},
		Exporters:  []config.Type{"nop"},
		Extensions: []config.Type{"nop"},
	}
	ExpectedOutput, err := yaml.Marshal(ExpectedYamlStruct)
	require.NoError(t, err)

	backup := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.Execute()
	require.NoError(t, err)

	bufChan := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)
		bufChan <- buf.String()
	}()

	err = w.Close()
	require.NoError(t, err)
	defer func() { os.Stdout = backup }()
	output := <-bufChan

	assert.Equal(t, strings.Trim(output, "\n"), strings.Trim(string(ExpectedOutput), "\n"))

}
