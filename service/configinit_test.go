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
	"fmt"
	"os"
	"testing"

	"github.com/ActiveState/vt10x"
	"github.com/AlecAivazis/survey/v2/terminal"
	expect "github.com/Netflix/go-expect"
	"github.com/kr/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension/ballastextension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

func testcomponents() component.Factories {
	factories := component.Factories{}

	factories.Extensions, _ = component.MakeExtensionFactoryMap(
		ballastextension.NewFactory(),
		zpagesextension.NewFactory(),
	)

	factories.Receivers, _ = component.MakeReceiverFactoryMap(
		otlpreceiver.NewFactory(),
	)

	factories.Exporters, _ = component.MakeExporterFactoryMap(
		loggingexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	)

	factories.Processors, _ = component.MakeProcessorFactoryMap(
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
	)

	return factories
}

func cleanup() {
	opts = &optionsSelected{}
}

func TestNewConfigurationInitCommand(t *testing.T) {
	t.Cleanup(cleanup)
	factories := testcomponents()
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "valid-logs",
			args: []string{"-r", "otlp", "-e", "otlp", "-l", "logs", "-p", "batch", "-d"},
		},
		{
			name: "valid-traces",
			args: []string{"-r", "otlp", "-e", "otlp", "-l", "traces", "-p", "batch", "-d"},
		},
		{
			name: "valid-metrics",
			args: []string{"-r", "otlp", "-e", "otlp", "-l", "metrics", "-p", "batch", "-d"},
		},
		{
			name: "valid-with-extension",
			args: []string{"-r", "otlp", "-e", "otlp", "-l", "logs", "-x", "zpages", "-p", "batch", "-d"},
		},
		{
			name:    "no-receiver",
			args:    []string{"-l", "metrics", "-e", "otlp", "-d"},
			wantErr: true,
		},
		{
			name:    "no-exporter",
			args:    []string{"-l", "metrics", "-r", "otlp", "-d"},
			wantErr: true,
		},
		{
			name:    "no-pipeline",
			args:    []string{"-r", "otlp", "-e", "otlp", "-d"},
			wantErr: true,
		},
		{
			name:    "invalid-pipeline-name",
			args:    []string{"-l", "invalid", "-r", "otlp", "-e", "otlp", "-d"},
			wantErr: true,
		},
		{
			name:    "duplicate-component",
			args:    []string{"-l", "metrics", "-r", "otlp", "-r", "otlp", "-e", "otlp", "-d"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewConfigurationInitCommand(CollectorSettings{Factories: factories})
			cmd.SilenceUsage = true
			origCfgCmd := cfgCmd
			cfgCmd.exiter = func(args ...error) error {
				return fmt.Errorf("%v", args)
			}
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if !tt.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			cfgCmd = origCfgCmd
		})
	}
}

func TestGetComponentNodeErrors(t *testing.T) {
	t.Cleanup(cleanup)
	factories := testcomponents()
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "invalid-exporter",
			args: []string{"-e", "noexp"},
		},
		{
			name: "invalid-processor",
			args: []string{"-p", "nopro"},
		},
		{
			name: "invalid-receiver",
			args: []string{"-r", "norec", "-d"},
		},
		{
			name: "invalid-extension",
			args: []string{"-x", "noext", "-d"},
		},
		{
			name: "invalid-service",
			args: []string{"-l", "pipeline-fail"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewConfigurationInitCommand(CollectorSettings{Factories: factories})
			cmd.SilenceUsage = true
			origCfgCmd := cfgCmd
			cfgCmd.exiter = func(args ...error) error {
				return fmt.Errorf("%v", args)
			}
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			assert.Error(t, err)
			cfgCmd = origCfgCmd
		})
	}
}

func TestSampleConfigFileCreation(t *testing.T) {
	t.Cleanup(cleanup)
	dir := t.TempDir()
	file := fmt.Sprintf("%v/%v", dir, "otelcol_temp_config.yaml")
	factories := testcomponents()
	cmd := NewConfigurationInitCommand(CollectorSettings{Factories: factories})
	cmd.SetArgs([]string{"-p", "batch", "-r", "otlp", "-e", "logging", "-l", "metrics", "-f", file})
	_ = cmd.Execute()

	_, err := os.Stat(file)
	assert.NoError(t, err, "file %v was expected", file)
}

func TestSampleConfigNotPresent(t *testing.T) {
	t.Cleanup(cleanup)
	dir := t.TempDir()
	file := fmt.Sprintf("%v/%v", dir, "otelcol_temp_config.yaml")
	// nop components dont have WithSampleConfig
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	cmd := NewConfigurationInitCommand(CollectorSettings{Factories: factories})
	cmd.SetArgs([]string{"-p", "nop", "-r", "nop", "-e", "nop", "-l", "metrics", "-f", file})
	err = cmd.Execute()
	assert.NoError(t, err)

	node := yaml.Node{}
	bytes, err := os.ReadFile(file)
	require.NoError(t, err)

	err = yaml.Unmarshal(bytes, &node)
	require.NoError(t, err)

	// get exporter.nop node and check the head comment
	assert.Equal(t, fmt.Sprintf("# Sample config for %v %v is not implemented.", "nop", "exporter"), node.Content[0].Content[1].Content[0].HeadComment)
}

// https://github.com/AlecAivazis/survey/blob/master/survey_posix_test.go
func RunCmdTest(t *testing.T, procedure func(*testing.T, *expect.Console, *vt10x.State), test func(*expect.Console)) {
	// Multiplex output to a buffer as well for the raw bytes.
	c, state, err := NewVTConsole(t)
	require.Nil(t, err)
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		procedure(t, c, state)
	}()

	test(c)

	// Close the client end of the pty, and read the remaining bytes from the provider's end.
	c.Tty().Close()
	<-donec

	// For debugging: Dump the terminal's screen.
	// t.Logf("\n%s", expect.StripTrailingEmptyLines(state.String()))
}

func NewVTConsole(t *testing.T, opts ...expect.ConsoleOpt) (*expect.Console, *vt10x.State, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}

	var screen vt10x.State
	term, err := vt10x.Create(&screen, pts)
	if err != nil {
		return nil, nil, err
	}

	opts = append([]expect.ConsoleOpt{
		expect.WithStdin(ptm),
		expect.WithStdout(term),
		expect.WithCloser(pts, ptm, term),
	}, opts...)

	c, err := expect.NewConsole(opts...)
	if err != nil {
		return nil, nil, err
	}

	return c, &screen, nil
}

func TestPrompts(t *testing.T) {
	t.Cleanup(cleanup)
	factories := testcomponents()
	set := &CollectorSettings{Factories: factories}
	available := getAvailableComponentsForPrompts(set)
	file := "otelcol_temp_config.yaml"
	RunCmdTest(t,
		func(t *testing.T, c *expect.Console, s *vt10x.State) {
			// select exporters at index 0, 1
			_, _ = c.ExpectString("Select exporters")
			_, _ = c.Send(" ")
			_, _ = c.Send(string(terminal.KeyArrowDown))
			_, _ = c.Send(" ")
			_, _ = c.SendLine(string(terminal.KeyEnter))

			// select processor at index 1
			_, _ = c.ExpectString("Select processors")
			_, _ = c.Send(string(terminal.KeyArrowDown))
			_, _ = c.Send(" ")
			_, _ = c.SendLine(string(terminal.KeyEnter))

			// select receiver at index 0
			_, _ = c.ExpectString("Select receivers")
			_, _ = c.Send(" ")
			_, _ = c.SendLine(string(terminal.KeyEnter))

			// select no extensions
			_, _ = c.ExpectString("Select extensions")
			_, _ = c.SendLine(string(terminal.KeyEnter))

			// select pipelines at index 1, 2
			_, _ = c.ExpectString("Select pipelines")
			_, _ = c.Send(string(terminal.KeyArrowDown))
			_, _ = c.Send(" ")
			_, _ = c.Send(string(terminal.KeyArrowDown))
			_, _ = c.Send(" ")
			_, _ = c.SendLine(string(terminal.KeyEnter))

			// enter file name
			_, _ = c.ExpectString("Name of the config file")
			_, _ = c.SendLine(file)

			_, _ = c.ExpectEOF()

		},
		func(c *expect.Console) {
			err := showPrompt(available, terminal.Stdio{In: c.Tty(), Out: c.Tty(), Err: c.Tty()})
			assert.NoError(t, err)

			assert.Equal(t, 2, len(opts.Exporters))
			assert.Equal(t, available.exporters[0:2], opts.Exporters)

			assert.Equal(t, 1, len(opts.Processors))
			assert.Equal(t, available.processors[1:], opts.Processors)

			assert.Equal(t, 1, len(opts.Receivers))
			assert.Equal(t, available.receivers[0:1], opts.Receivers)

			assert.Equal(t, 0, len(opts.Extensions))

			assert.Equal(t, 2, len(opts.Pipelines))
			assert.Equal(t, available.pipelines[1:], opts.Pipelines)

			assert.Equal(t, file, opts.ConfigFileName)
		})
}
