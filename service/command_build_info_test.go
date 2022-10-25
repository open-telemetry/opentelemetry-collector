package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/featuregate"
)

func TestNewBuildSubCommand(t *testing.T) {
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
		BuildInfo:  component.NewDefaultBuildInfo(),
		Receivers:  []config.Type{"nop"},
		Processors: []config.Type{"nop"},
		Exporters:  []config.Type{"nop"},
		Extensions: []config.Type{"nop"},
	}
	ExpectedOutput, err := yaml.Marshal(ExpectedYamlStruct)
	require.NoError(t, err)

	// Obtaining StdOutput of cmd.Execute()
	oldStdOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w // Write to os.StdOut

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
	defer func() { os.Stdout = oldStdOut }() // Restore os.Stdout to old value after test
	output := <-bufChan
	// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
	// line that makes the test fail.
	assert.Equal(t, strings.Trim(output, "\n"), strings.Trim(string(ExpectedOutput), "\n"))
}
