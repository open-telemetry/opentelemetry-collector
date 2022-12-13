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
	"bufio"
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/testutil"
)

type labelState int

const (
	labelNotPresent labelState = iota
	labelSpecificValue
	labelAnyValue
)

type labelValue struct {
	label string
	state labelState
}

type ownMetricsTestCase struct {
	name                string
	userDefinedResource map[string]*string
	expectedLabels      map[string]labelValue
}

var testResourceAttrValue = "resource_attr_test_value"
var testInstanceID = "test_instance_id"
var testServiceVersion = "2022-05-20"

const metricsVersion = "test version"

func ownMetricsTestCases() []ownMetricsTestCase {
	return []ownMetricsTestCase{{
		name:                "no resource",
		userDefinedResource: nil,
		// All labels added to all collector metrics by default are listed below.
		// These labels are hard coded here in order to avoid inadvertent changes:
		// at this point changing labels should be treated as a breaking changing
		// and requires a good justification. The reason is that changes to metric
		// names or labels can break alerting, dashboards, etc that are used to
		// monitor the Collector in production deployments.
		expectedLabels: map[string]labelValue{
			"service_instance_id": {state: labelAnyValue},
			"service_version":     {label: metricsVersion, state: labelSpecificValue},
		},
	},
		{
			name: "resource with custom attr",
			userDefinedResource: map[string]*string{
				"custom_resource_attr": &testResourceAttrValue,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id":  {state: labelAnyValue},
				"service_version":      {label: metricsVersion, state: labelSpecificValue},
				"custom_resource_attr": {label: "resource_attr_test_value", state: labelSpecificValue},
			},
		},
		{
			name: "override service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": &testInstanceID,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {label: "test_instance_id", state: labelSpecificValue},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": nil, // nil value in config is used to suppress attributes.
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelNotPresent},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "override service.version",
			userDefinedResource: map[string]*string{
				"service.version": &testServiceVersion,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_version":     {label: "2022-05-20", state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.version",
			userDefinedResource: map[string]*string{
				"service.version": nil, // nil value in config is used to suppress attributes.
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_version":     {state: labelNotPresent},
			},
		}}
}

func TestService_GetFactory(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	assert.Nil(t, srv.host.GetFactory(component.KindReceiver, "wrongtype"))
	assert.Equal(t, factories.Receivers["nop"], srv.host.GetFactory(component.KindReceiver, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindProcessor, "wrongtype"))
	assert.Equal(t, factories.Processors["nop"], srv.host.GetFactory(component.KindProcessor, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindExporter, "wrongtype"))
	assert.Equal(t, factories.Exporters["nop"], srv.host.GetFactory(component.KindExporter, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindExtension, "wrongtype"))
	assert.Equal(t, factories.Extensions["nop"], srv.host.GetFactory(component.KindExtension, "nop"))

	// Try retrieve non existing component.Kind.
	assert.Nil(t, srv.host.GetFactory(42, "nop"))
}

func TestServiceGetExtensions(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.host.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, component.NewID("nop"))
}

func TestServiceGetExporters(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expMap := srv.host.GetExporters()
	assert.Len(t, expMap, 3)
	assert.Len(t, expMap[component.DataTypeTraces], 1)
	assert.Contains(t, expMap[component.DataTypeTraces], component.NewID("nop"))
	assert.Len(t, expMap[component.DataTypeMetrics], 1)
	assert.Contains(t, expMap[component.DataTypeMetrics], component.NewID("nop"))
	assert.Len(t, expMap[component.DataTypeLogs], 1)
	assert.Contains(t, expMap[component.DataTypeLogs], component.NewID("nop"))
}

// TestServiceTelemetryCleanupOnError tests that if newService errors due to an invalid config telemetry is cleaned up
// and another service with a valid config can be started right after.
func TestServiceTelemetryCleanupOnError(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	// Read invalid yaml config from file
	invalidProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-invalid.yaml")}))
	require.NoError(t, err)
	invalidCfg, err := invalidProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service with an invalid config and expect an error
	_, err = newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    invalidCfg,
	})
	require.Error(t, err)

	// Read valid yaml config from file
	validProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	validCfg, err := validProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service with a valid config and expect no error
	srv, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
	})
	require.NoError(t, err)
	assert.NoError(t, srv.Shutdown(context.Background()))
}

func TestServiceTelemetryWithOpenCensusMetrics(t *testing.T) {
	for _, tc := range ownMetricsTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			testCollectorStartHelper(t, featuregate.NewRegistry(), tc)
		})
	}
}

func TestServiceTelemetryWithOpenTelemetryMetrics(t *testing.T) {
	for _, tc := range ownMetricsTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			registry := featuregate.NewRegistry()
			obsreportconfig.RegisterInternalMetricFeatureGate(registry)
			colTel := newColTelemetry(registry)
			require.NoError(t, colTel.registry.Apply(map[string]bool{obsreportconfig.UseOtelForInternalMetricsfeatureGateID: true}))
			testCollectorStartHelper(t, registry, tc)
		})
	}
}

func testCollectorStartHelper(t *testing.T, reg *featuregate.Registry, tc ownMetricsTestCase) {
	factories, err := componenttest.NopFactories()
	zpagesExt := zpagesextension.NewFactory()
	factories.Extensions[zpagesExt.Type()] = zpagesExt
	require.NoError(t, err)
	var once sync.Once
	loggingHookCalled := false
	hook := func(entry zapcore.Entry) error {
		once.Do(func() {
			loggingHookCalled = true
		})
		return nil
	}

	metricsAddr := testutil.GetAvailableLocalAddress(t)
	zpagesAddr := testutil.GetAvailableLocalAddress(t)
	// Prepare config properties to be merged with the main config.
	extraCfgAsProps := map[string]interface{}{
		// Setup the zpages extension.
		"extensions::zpages::endpoint": zpagesAddr,
		"service::extensions":          "nop, zpages",
		// Set the metrics address to expose own metrics on.
		"service::telemetry::metrics::address": metricsAddr,
	}
	// Also include resource attributes under the service::telemetry::resource key.
	for k, v := range tc.userDefinedResource {
		extraCfgAsProps["service::telemetry::resource::"+k] = v
	}

	cfgSet := newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")})
	cfgSet.ResolverSettings.Converters = append([]confmap.Converter{
		mapConverter{extraCfgAsProps}},
		cfgSet.ResolverSettings.Converters...,
	)
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	cfg, err := cfgProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service, check for metrics, shutdown and repeat to ensure that telemetry can be started/shutdown and started again.
	for i := 0; i < 2; i++ {
		srv, err := newService(&settings{
			BuildInfo:      component.BuildInfo{Version: "test version"},
			Factories:      factories,
			Config:         cfg,
			LoggingOptions: []zap.Option{zap.Hooks(hook)},
			registry:       reg,
		})
		require.NoError(t, err)

		require.NoError(t, srv.Start(context.Background()))
		// Sleep for 1 second to ensure the http server is started.
		time.Sleep(1 * time.Second)
		assert.True(t, loggingHookCalled)
		assertMetrics(t, metricsAddr, tc.expectedLabels)
		assertZPages(t, zpagesAddr)
		require.NoError(t, srv.Shutdown(context.Background()))
	}
}

// TestServiceTelemetryRestart tests that the service correctly restarts the telemetry server.
func TestServiceTelemetryRestart(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	// Read valid yaml config from file
	validProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	validCfg, err := validProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service
	srvOne, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
	})
	require.NoError(t, err)

	// URL of the telemetry service metrics endpoint
	telemetryURL := "http://localhost:8888/metrics"

	// Start the service
	require.NoError(t, srvOne.Start(context.Background()))

	// check telemetry server to ensure we get a response
	var resp *http.Response

	// #nosec G107
	resp, err = http.Get(telemetryURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown the service
	require.NoError(t, srvOne.Shutdown(context.Background()))

	// Create a new service with the same telemetry
	srvTwo, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
	})
	require.NoError(t, err)

	// Start the new service
	require.NoError(t, srvTwo.Start(context.Background()))

	// check telemetry server to ensure we get a response
	// #nosec G107
	resp, err = http.Get(telemetryURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown the new service
	require.NoError(t, srvTwo.Shutdown(context.Background()))
}

func createExampleService(t *testing.T, factories component.Factories) *service {
	// Read yaml config from file
	prov, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	cfg, err := prov.Get(context.Background(), factories)
	require.NoError(t, err)

	srv, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    cfg,
	})
	require.NoError(t, err)
	return srv
}

func assertMetrics(t *testing.T, metricsAddr string, expectedLabels map[string]labelValue) {
	client := &http.Client{}
	resp, err := client.Get("http://" + metricsAddr + "/metrics")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, resp.Body.Close())
	})
	reader := bufio.NewReader(resp.Body)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(reader)
	require.NoError(t, err)

	prefix := "otelcol"
	for metricName, metricFamily := range parsed {
		// require is used here so test fails with a single message.
		require.True(
			t,
			strings.HasPrefix(metricName, prefix),
			"expected prefix %q but string starts with %q",
			prefix,
			metricName[:len(prefix)+1]+"...")

		for _, metric := range metricFamily.Metric {
			labelMap := map[string]string{}
			for _, labelPair := range metric.Label {
				labelMap[*labelPair.Name] = *labelPair.Value
			}

			for k, v := range expectedLabels {
				switch v.state {
				case labelNotPresent:
					_, present := labelMap[k]
					assert.Falsef(t, present, "label %q must not be present", k)
				case labelSpecificValue:
					require.Equalf(t, v.label, labelMap[k], "mandatory label %q value mismatch", k)
				case labelAnyValue:
					assert.NotEmptyf(t, labelMap[k], "mandatory label %q not present", k)
				}
			}
		}
	}
}

func assertZPages(t *testing.T, zpagesAddr string) {
	paths := []string{
		"/debug/tracez",
		// TODO: enable this when otel-metrics is used and this page is available.
		// "/debug/rpcz",
		"/debug/pipelinez",
		"/debug/servicez",
		"/debug/extensionz",
	}

	testZPagePathFn := func(t *testing.T, path string) {
		client := &http.Client{}
		resp, err := client.Get("http://" + zpagesAddr + path)
		if !assert.NoError(t, err, "error retrieving zpage at %q", path) {
			return
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unsuccessful zpage %q GET", path)
		assert.NoError(t, resp.Body.Close())
	}

	for _, path := range paths {
		testZPagePathFn(t, path)
	}
}

// mapConverter applies extraMap of config settings. Useful for overriding the config
// for testing purposes. Keys must use "::" delimiter between levels.
type mapConverter struct {
	extraMap map[string]interface{}
}

func (m mapConverter) Convert(ctx context.Context, conf *confmap.Conf) error {
	return conf.Merge(confmap.NewFromStringMap(m.extraMap))
}
