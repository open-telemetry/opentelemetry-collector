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

package tests

// This file defines parametrized test scenarios and makes them public so that they can be
// also used by tests in custom builds of Collector (e.g. Collector Contrib).

import (
	"fmt"
	"math/rand"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/testbed/testbed"
)

var (
	performanceResultsSummary testbed.TestResultsSummary = &testbed.PerformanceResults{}
)

// createConfigYaml creates a collector config file that corresponds to the
// sender and receiver used in the test and returns the config file name.
// Map of processor names to their configs. Config is in YAML and must be
// indented by 2 spaces. Processors will be placed between batch and queue for traces
// pipeline. For metrics pipeline these will be sole processors.
func createConfigYaml(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resultDir string,
	processors map[string]string,
	extensions map[string]string,
) string {

	// Create a config. Note that our DataSender is used to generate a config for Collector's
	// receiver and our DataReceiver is used to generate a config for Collector's exporter.
	// This is because our DataSender sends to Collector's receiver and our DataReceiver
	// receives from Collector's exporter.

	// Prepare extra processor config section and comma-separated list of extra processor
	// names to use in corresponding "processors" settings.
	processorsSections := ""
	processorsList := ""
	if len(processors) > 0 {
		first := true
		for name, cfg := range processors {
			processorsSections += cfg + "\n"
			if !first {
				processorsList += ","
			}
			processorsList += name
			first = false
		}
	}

	// Prepare extra extension config section and comma-separated list of extra extension
	// names to use in corresponding "extensions" settings.
	extensionsSections := ""
	extensionsList := ""
	if len(extensions) > 0 {
		first := true
		for name, cfg := range extensions {
			extensionsSections += cfg + "\n"
			if !first {
				extensionsList += ","
			}
			extensionsList += name
			first = false
		}
	}

	// Set pipeline based on DataSender type
	var pipeline string
	switch sender.(type) {
	case testbed.TraceDataSender:
		pipeline = "traces"
	case testbed.MetricDataSender:
		pipeline = "metrics"
	case testbed.LogDataSender:
		pipeline = "logs"
	default:
		t.Error("Invalid DataSender type")
	}

	format := `
receivers:%v
exporters:%v
processors:
  %s

extensions:
  pprof:
    save_to_file: %v/cpu.prof
  %s

service:
  extensions: [pprof, %s]
  pipelines:
    %s:
      receivers: [%v]
      processors: [%s]
      exporters: [%v]
`

	// Put corresponding elements into the config template to generate the final config.
	return fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		processorsSections,
		resultDir,
		extensionsSections,
		extensionsList,
		pipeline,
		sender.ProtocolName(),
		processorsList,
		receiver.ProtocolName(),
	)
}

// Scenario10kItemsPerSecond runs 10k data items/sec test using specified sender and receiver protocols.
func Scenario10kItemsPerSecond(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resourceSpec testbed.ResourceSpec,
	resultsSummary testbed.TestResultsSummary,
	processors map[string]string,
	extensions map[string]string,
) {
	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	require.NoError(t, err)

	options := testbed.LoadOptions{
		DataItemsPerSecond: 10_000,
		ItemsPerBatch:      100,
		Parallel:           1,
	}
	agentProc := &testbed.ChildProcess{}

	configStr := createConfigYaml(t, sender, receiver, resultDir, processors, extensions)
	configCleanup, err := agentProc.PrepareConfig(configStr)
	require.NoError(t, err)
	defer configCleanup()

	dataProvider := testbed.NewPerfTestDataProvider(options)
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		agentProc,
		&testbed.PerfTestValidator{},
		resultsSummary,
	)
	defer tc.Stop()

	tc.SetResourceLimits(resourceSpec)
	tc.StartBackend()
	tc.StartAgent("--log-level=debug")

	tc.StartLoad(options)

	tc.Sleep(tc.Duration)

	tc.StopLoad()

	tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() > 0 }, "load generator started")
	tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
		"all data items received")

	tc.StopAgent()

	tc.ValidateData()
}

// TestCase for Scenario1kSPSWithAttrs func.
type TestCase struct {
	attrCount      int
	attrSizeByte   int
	expectedMaxCPU uint32
	expectedMaxRAM uint32
	resultsSummary testbed.TestResultsSummary
}

func genRandByteString(len int) string {
	b := make([]byte, len)
	for i := range b {
		b[i] = byte(rand.Intn(128))
	}
	return string(b)
}

// Scenario1kSPSWithAttrs runs a performance test at 1k sps with specified span attributes
// and test options.
func Scenario1kSPSWithAttrs(t *testing.T, args []string, tests []TestCase, processors map[string]string) {
	for i := range tests {
		test := tests[i]

		t.Run(fmt.Sprintf("%d*%dbytes", test.attrCount, test.attrSizeByte), func(t *testing.T) {

			options := constructLoadOptions(test)

			agentProc := &testbed.ChildProcess{}

			// Prepare results dir.
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			require.NoError(t, err)

			// Create sender and receiver on available ports.
			sender := testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
			receiver := testbed.NewOCDataReceiver(testbed.GetAvailablePort(t))

			// Prepare config.
			configStr := createConfigYaml(t, sender, receiver, resultDir, processors, nil)
			configCleanup, err := agentProc.PrepareConfig(configStr)
			require.NoError(t, err)
			defer configCleanup()

			tc := testbed.NewTestCase(
				t,
				testbed.NewPerfTestDataProvider(options),
				sender,
				receiver,
				agentProc,
				&testbed.PerfTestValidator{},
				test.resultsSummary,
			)
			defer tc.Stop()

			tc.SetResourceLimits(testbed.ResourceSpec{
				ExpectedMaxCPU: test.expectedMaxCPU,
				ExpectedMaxRAM: test.expectedMaxRAM,
			})

			tc.StartBackend()
			tc.StartAgent(args...)

			tc.StartLoad(options)
			tc.Sleep(tc.Duration)
			tc.StopLoad()

			tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() > 0 }, "load generator started")
			tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
				"all spans received")

			tc.StopAgent()

			tc.ValidateData()
		})
	}
}

// Structure used for TestTraceNoBackend10kSPS.
// Defines RAM usage range for defined processor type.
type processorConfig struct {
	Name string
	// map of processor types to their config YAML to use.
	Processor           map[string]string
	ExpectedMaxRAM      uint32
	ExpectedMinFinalRAM uint32
}

func ScenarioTestTraceNoBackend10kSPS(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resourceSpec testbed.ResourceSpec,
	resultsSummary testbed.TestResultsSummary,
	configuration processorConfig,
) {

	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	require.NoError(t, err)

	options := testbed.LoadOptions{DataItemsPerSecond: 10000, ItemsPerBatch: 10}
	agentProc := &testbed.ChildProcess{}
	configStr := createConfigYaml(t, sender, receiver, resultDir, configuration.Processor, nil)
	configCleanup, err := agentProc.PrepareConfig(configStr)
	require.NoError(t, err)
	defer configCleanup()

	dataProvider := testbed.NewPerfTestDataProvider(options)
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		agentProc,
		&testbed.PerfTestValidator{},
		resultsSummary,
	)

	defer tc.Stop()

	tc.SetResourceLimits(resourceSpec)

	tc.StartAgent()
	tc.StartLoad(options)

	tc.Sleep(tc.Duration)

	rss, _, _ := tc.AgentMemoryInfo()
	assert.Less(t, configuration.ExpectedMinFinalRAM, rss)
}

func constructLoadOptions(test TestCase) testbed.LoadOptions {
	options := testbed.LoadOptions{DataItemsPerSecond: 1000, ItemsPerBatch: 10}
	options.Attributes = make(map[string]string)

	// Generate attributes.
	for i := 0; i < test.attrCount; i++ {
		attrName := genRandByteString(rand.Intn(199) + 1)
		options.Attributes[attrName] = genRandByteString(rand.Intn(test.attrSizeByte*2-1) + 1)
	}
	return options
}
