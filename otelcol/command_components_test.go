// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewBuildSubCommand(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
		ProviderModules: map[string]string{
			"nop": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
		},
		ConverterModules: []string{
			"go.opentelemetry.io/collector/converter/testconverter v1.2.3",
		},
	}
	cmd := NewCommand(set)
	cmd.SetArgs([]string{"components"})

	ExpectedOutput, err := os.ReadFile(filepath.Join("testdata", "components-output.yaml"))
	require.NoError(t, err)

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	err = cmd.Execute()
	require.NoError(t, err)

	// Trim new line at the end of the two strings to make a better comparison as string() adds an extra new
	// line that makes the test fail.
	assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(ExpectedOutput), "\n", ""), "\r", ""), strings.ReplaceAll(strings.ReplaceAll(b.String(), "\n", ""), "\r", ""))
}
