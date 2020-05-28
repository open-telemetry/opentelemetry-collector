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

// Package collector handles the command-line, configuration, and runs the OC collector.
package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service/defaultcomponents"
	"go.opentelemetry.io/collector/testutils"
)

func TestApplication_Start(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	app, err := New(Parameters{Factories: factories, ApplicationStartInfo: ApplicationStartInfo{}})
	require.NoError(t, err)
	assert.Equal(t, app.rootCmd, app.Command())

	const testPrefix = "a_test"
	metricsPort := testutils.GetAvailablePort(t)
	app.rootCmd.SetArgs([]string{
		"--config=testdata/otelcol-config.yaml",
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
		"--add-instance-id=true",
	})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.NoError(t, app.Start())
	}()

	<-app.readyChan

	require.True(t, isAppAvailable(t, "http://localhost:13133"))
	assert.Equal(t, app.logger, app.GetLogger())

	assertMetricsPrefix(t, testPrefix, metricsPort)

	close(app.stopTestChan)
	<-appDone
}

// isAppAvailable checks if the healthcheck server at the given endpoint is
// returning `available`.
func isAppAvailable(t *testing.T, healthCheckEndPoint string) bool {
	client := &http.Client{}
	resp, err := client.Get(healthCheckEndPoint)
	require.NoError(t, err)

	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func assertMetricsPrefix(t *testing.T, prefix string, metricsPort uint16) {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	require.NoError(t, err)

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	for {
		s, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		if len(s) == 0 || s[0] == '#' {
			// Skip this line since it is not a metric.
			continue
		}

		// require is used here so test fails with a single message.
		require.True(
			t,
			strings.HasPrefix(s, prefix),
			"expected prefix %q but string starts with %q",
			prefix,
			s[:len(prefix)+1]+"...")
	}
}

func TestApplication_setupExtensions(t *testing.T) {
	exampleExtensionFactory := &config.ExampleExtensionFactory{FailCreation: true}
	exampleExtensionConfig := &config.ExampleExtensionCfg{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: exampleExtensionFactory.Type(),
			NameVal: string(exampleExtensionFactory.Type()),
		},
	}

	badExtensionFactory := &badExtensionFactory{}
	badExtensionFactoryConfig := &configmodels.ExtensionSettings{
		TypeVal: "bf",
		NameVal: "bf",
	}

	tests := []struct {
		name       string
		factories  config.Factories
		config     *configmodels.Config
		wantErrMsg string
	}{
		{
			name: "extension_not_configured",
			config: &configmodels.Config{
				Service: configmodels.Service{
					Extensions: []string{
						"myextension",
					},
				},
			},
			wantErrMsg: "cannot build builtExtensions: extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(exampleExtensionFactory.Type()): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(exampleExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "cannot build builtExtensions: extension factory for type \"exampleextension\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: config.Factories{
				Extensions: map[configmodels.Type]component.ExtensionFactory{
					exampleExtensionFactory.Type(): exampleExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(exampleExtensionFactory.Type()): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(exampleExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "cannot build builtExtensions: failed to create extension \"exampleextension\": cannot create \"exampleextension\" extension type",
		},
		{
			name: "bad_factory",
			factories: config.Factories{
				Extensions: map[configmodels.Type]component.ExtensionFactory{
					badExtensionFactory.Type(): badExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(badExtensionFactory.Type()): badExtensionFactoryConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(badExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "cannot build builtExtensions: factory for \"bf\" produced a nil extension",
		},
	}

	nopLogger := zap.NewNop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Application{
				logger:    nopLogger,
				factories: tt.factories,
				config:    tt.config,
			}

			err := app.setupExtensions(context.Background())

			if tt.wantErrMsg == "" {
				assert.NoError(t, err)
				assert.Equal(t, 1, len(app.builtExtensions))
				for _, ext := range app.builtExtensions {
					assert.NotNil(t, ext)
				}
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
				assert.Equal(t, 0, len(app.builtExtensions))
			}
		})
	}
}

// badExtensionFactory is a factory that returns no error but returns a nil object.
type badExtensionFactory struct{}

func (b badExtensionFactory) Type() configmodels.Type {
	return "bf"
}

func (b badExtensionFactory) CreateDefaultConfig() configmodels.Extension {
	return &configmodels.ExtensionSettings{}
}

func (b badExtensionFactory) CreateExtension(_ context.Context, _ component.ExtensionCreateParams, _ configmodels.Extension) (component.ServiceExtension, error) {
	return nil, nil
}

func TestApplication_GetFactory(t *testing.T) {
	// Create some factories.
	exampleReceiverFactory := &config.ExampleReceiverFactory{}
	exampleProcessorFactory := &config.ExampleProcessorFactory{}
	exampleExporterFactory := &config.ExampleExporterFactory{}
	exampleExtensionFactory := &config.ExampleExtensionFactory{}

	factories := config.Factories{
		Receivers: map[configmodels.Type]component.ReceiverFactoryBase{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
		},
		Processors: map[configmodels.Type]component.ProcessorFactoryBase{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
		},
		Exporters: map[configmodels.Type]component.ExporterFactoryBase{
			exampleExporterFactory.Type(): exampleExporterFactory,
		},
		Extensions: map[configmodels.Type]component.ExtensionFactory{
			exampleExtensionFactory.Type(): exampleExtensionFactory,
		},
	}

	// Create an App with factories.
	app, err := New(Parameters{Factories: factories})
	require.NoError(t, err)

	// Verify GetFactory call for all component kinds.

	factory := app.GetFactory(component.KindReceiver, exampleReceiverFactory.Type())
	assert.EqualValues(t, exampleReceiverFactory, factory)
	factory = app.GetFactory(component.KindReceiver, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = app.GetFactory(component.KindProcessor, exampleProcessorFactory.Type())
	assert.EqualValues(t, exampleProcessorFactory, factory)
	factory = app.GetFactory(component.KindProcessor, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = app.GetFactory(component.KindExporter, exampleExporterFactory.Type())
	assert.EqualValues(t, exampleExporterFactory, factory)
	factory = app.GetFactory(component.KindExporter, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = app.GetFactory(component.KindExtension, exampleExtensionFactory.Type())
	assert.EqualValues(t, exampleExtensionFactory, factory)
	factory = app.GetFactory(component.KindExtension, "wrongtype")
	assert.EqualValues(t, nil, factory)
}

func createExampleApplication(t *testing.T) *Application {
	// Create some factories.
	exampleReceiverFactory := &config.ExampleReceiverFactory{}
	exampleProcessorFactory := &config.ExampleProcessorFactory{}
	exampleExporterFactory := &config.ExampleExporterFactory{}
	exampleExtensionFactory := &config.ExampleExtensionFactory{}
	factories := config.Factories{
		Receivers: map[configmodels.Type]component.ReceiverFactoryBase{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
		},
		Processors: map[configmodels.Type]component.ProcessorFactoryBase{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
		},
		Exporters: map[configmodels.Type]component.ExporterFactoryBase{
			exampleExporterFactory.Type(): exampleExporterFactory,
		},
		Extensions: map[configmodels.Type]component.ExtensionFactory{
			exampleExtensionFactory.Type(): exampleExtensionFactory,
		},
	}

	app, err := New(Parameters{
		Factories: factories,
		ConfigFactory: func(v *viper.Viper, factories config.Factories) (c *configmodels.Config, err error) {
			config := &configmodels.Config{
				Receivers: map[string]configmodels.Receiver{
					string(exampleReceiverFactory.Type()): exampleReceiverFactory.CreateDefaultConfig(),
				},
				Exporters: map[string]configmodels.Exporter{
					string(exampleExporterFactory.Type()): exampleExporterFactory.CreateDefaultConfig(),
				},
				Extensions: map[string]configmodels.Extension{
					string(exampleExtensionFactory.Type()): exampleExtensionFactory.CreateDefaultConfig(),
				},
				Service: configmodels.Service{
					Extensions: []string{string(exampleExtensionFactory.Type())},
					Pipelines: map[string]*configmodels.Pipeline{
						"trace": {
							Name:       "traces",
							InputType:  configmodels.TracesDataType,
							Receivers:  []string{string(exampleReceiverFactory.Type())},
							Processors: []string{},
							Exporters:  []string{string(exampleExporterFactory.Type())},
						},
					},
				},
			}
			return config, nil
		},
	})
	require.NoError(t, err)
	return app
}

func TestApplication_GetExtensions(t *testing.T) {
	app := createExampleApplication(t)

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.NoError(t, app.Start())
	}()

	<-app.readyChan

	// Verify GetExensions(). The results must match what we have in testdata/otelcol-config.yaml.

	extMap := app.GetExtensions()
	var extTypes []string
	for cfg, ext := range extMap {
		assert.NotNil(t, ext)
		extTypes = append(extTypes, string(cfg.Type()))
	}
	sort.Strings(extTypes)

	assert.Equal(t, []string{"exampleextension"}, extTypes)

	// Stop the Application.
	close(app.stopTestChan)
	<-appDone
}

func TestApplication_GetExporters(t *testing.T) {
	app := createExampleApplication(t)

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.NoError(t, app.Start())
	}()

	<-app.readyChan

	// Verify GetExporters().

	expMap := app.GetExporters()
	var expTypes []string
	var expCfg configmodels.Exporter
	for _, m := range expMap {
		for cfg, exp := range m {
			if exp != nil {
				expTypes = append(expTypes, string(cfg.Type()))
				assert.Nil(t, expCfg)
				expCfg = cfg
			}
		}
	}
	sort.Strings(expTypes)

	assert.Equal(t, []string{"exampleexporter"}, expTypes)

	assert.EqualValues(t, 0, len(expMap[configmodels.MetricsDataType]))
	assert.NotNil(t, expMap[configmodels.TracesDataType][expCfg])
	assert.EqualValues(t, "exampleexporter", expCfg.Type())

	// Stop the Application.
	close(app.stopTestChan)
	<-appDone
}
