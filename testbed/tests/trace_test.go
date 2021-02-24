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

// Package tests contains test cases. To run the tests go to tests directory and run:
// RUN_TESTBED=1 go test -v

package tests

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"context"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testbed/testbed"
	"go.opentelemetry.io/collector/translator/conventions"
)

// TestMain is used to initiate setup, execution and tear down of testbed.
func TestMain(m *testing.M) {
	testbed.DoTestMain(m, performanceResultsSummary)
}

func TestTrace10kSPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"JaegerGRPC",
			testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewJaegerDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 40,
				ExpectedMaxRAM: 70,
			},
		},
		{
			"OpenCensus",
			testbed.NewOCTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOCDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 39,
				ExpectedMaxRAM: 82,
			},
		},
		{
			"OTLP-gRPC",
			testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 20,
				ExpectedMaxRAM: 70,
			},
		},
		{
			"OTLP-gRPC-gzip",
			testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)).WithCompression("gzip"),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 30,
				ExpectedMaxRAM: 100,
			},
		},
		{
			"OTLP-HTTP",
			testbed.NewOTLPHTTPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPHTTPDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 20,
				ExpectedMaxRAM: 100,
			},
		},
		{
			"OTLP-HTTP-gzip",
			testbed.NewOTLPHTTPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPHTTPDataReceiver(testbed.GetAvailablePort(t)).WithCompression("gzip"),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 25,
				ExpectedMaxRAM: 100,
			},
		},
		{
			"Zipkin",
			testbed.NewZipkinDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewZipkinDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 80,
			},
		},
	}

	processors := map[string]string{
		"batch": `
  batch:
`,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processors,
				nil,
			)
		})
	}
}

func TestTraceNoBackend10kSPS(t *testing.T) {

	limitProcessors := map[string]string{
		"memory_limiter": `
  memory_limiter:
   check_interval: 100ms
   limit_mib: 20
`,
	}

	noLimitProcessors := map[string]string{}

	var processorsConfig = []processorConfig{
		{
			Name:                "NoMemoryLimit",
			Processor:           noLimitProcessors,
			ExpectedMaxRAM:      170,
			ExpectedMinFinalRAM: 100,
		},
		{
			Name:                "MemoryLimit",
			Processor:           limitProcessors,
			ExpectedMaxRAM:      70,
			ExpectedMinFinalRAM: 40,
		},
	}

	for _, testConf := range processorsConfig {
		t.Run(testConf.Name, func(t *testing.T) {
			ScenarioTestTraceNoBackend10kSPS(
				t,
				testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
				testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
				testbed.ResourceSpec{
					ExpectedMaxCPU: 50,
					ExpectedMaxRAM: testConf.ExpectedMaxRAM,
				},
				performanceResultsSummary,
				testConf,
			)
		})
	}
}

func TestTrace1kSPSWithAttrs(t *testing.T) {
	Scenario1kSPSWithAttrs(t, []string{}, []TestCase{
		// No attributes.
		{
			attrCount:      0,
			attrSizeByte:   0,
			expectedMaxCPU: 30,
			expectedMaxRAM: 100,
			resultsSummary: performanceResultsSummary,
		},

		// We generate 10 attributes each with average key length of 100 bytes and
		// average value length of 50 bytes so total size of attributes values is
		// 15000 bytes.
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 120,
			expectedMaxRAM: 100,
			resultsSummary: performanceResultsSummary,
		},

		// Approx 10 KiB attributes.
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 100,
			expectedMaxRAM: 100,
			resultsSummary: performanceResultsSummary,
		},

		// Approx 100 KiB attributes.
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 250,
			expectedMaxRAM: 100,
			resultsSummary: performanceResultsSummary,
		},
	}, nil)
}

func TestTraceBallast1kSPSWithAttrs(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}
	Scenario1kSPSWithAttrs(t, args, []TestCase{
		// No attributes.
		{
			attrCount:      0,
			attrSizeByte:   0,
			expectedMaxCPU: 30,
			expectedMaxRAM: 2000,
			resultsSummary: performanceResultsSummary,
		},
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
			resultsSummary: performanceResultsSummary,
		},
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
			resultsSummary: performanceResultsSummary,
		},
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 120,
			expectedMaxRAM: 2000,
			resultsSummary: performanceResultsSummary,
		},
	}, nil)
}

func TestTraceBallast1kSPSAddAttrs(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}

	attrProcCfg := `
  attributes:
    actions:
      - key: attrib.key00
        value: 123
        action: insert
      - key: attrib.key01
        value: "a small string for this attribute"
        action: insert
      - key: attrib.key02
        value: true
        action: insert
      - key: region
        value: test-region
        action: insert
      - key: data-center
        value: test-datacenter
        action: insert`

	Scenario1kSPSWithAttrs(
		t,
		args,
		[]TestCase{
			{
				attrCount:      0,
				attrSizeByte:   0,
				expectedMaxCPU: 30,
				expectedMaxRAM: 2000,
				resultsSummary: performanceResultsSummary,
			},
			{
				attrCount:      100,
				attrSizeByte:   50,
				expectedMaxCPU: 80,
				expectedMaxRAM: 2000,
				resultsSummary: performanceResultsSummary,
			},
			{
				attrCount:      10,
				attrSizeByte:   1000,
				expectedMaxCPU: 80,
				expectedMaxRAM: 2000,
				resultsSummary: performanceResultsSummary,
			},
			{
				attrCount:      20,
				attrSizeByte:   5000,
				expectedMaxCPU: 120,
				expectedMaxRAM: 2000,
				resultsSummary: performanceResultsSummary,
			},
		},
		map[string]string{"attributes": attrProcCfg},
	)
}

// verifySingleSpan sends a single span to Collector, waits until the span is forwarded
// and received by MockBackend and calls user-supplied verification functions on
// received span.
// Temporarily, we need two verification functions in order to verify spans in
// new and old format received by MockBackend.
func verifySingleSpan(
	t *testing.T,
	tc *testbed.TestCase,
	serviceName string,
	spanName string,
	verifyReceived func(span pdata.Span),
) {

	// Clear previously received traces.
	tc.MockBackend.ClearReceivedItems()
	startCounter := tc.MockBackend.DataItemsReceived()

	// Send one span.
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	td.ResourceSpans().At(0).Resource().Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.AttributeServiceName: pdata.NewAttributeValueString(serviceName),
	})
	td.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
	spans.Resize(1)
	spans.At(0).SetTraceID(testbed.GenerateSequentialTraceID(1))
	spans.At(0).SetSpanID(testbed.GenerateSequentialSpanID(1))
	spans.At(0).SetName(spanName)

	sender := tc.Sender.(testbed.TraceDataSender)
	require.NoError(t, sender.ConsumeTraces(context.Background(), td))

	// We bypass the load generator in this test, but make sure to increment the
	// counter since it is used in final reports.
	tc.LoadGenerator.IncDataItemsSent()

	// Wait until span is received.
	tc.WaitFor(func() bool { return tc.MockBackend.DataItemsReceived() == startCounter+1 },
		"span received")

	// Verify received span.
	count := 0
	for _, td := range tc.MockBackend.ReceivedTraces {
		rs := td.ResourceSpans()
		for i := 0; i < rs.Len(); i++ {
			ils := rs.At(i).InstrumentationLibrarySpans()
			for j := 0; j < ils.Len(); j++ {
				spans := ils.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					verifyReceived(spans.At(k))
					count++
				}
			}
		}
	}
	assert.EqualValues(t, 1, count, "must receive one span")
}

func TestTraceAttributesProcessor(t *testing.T) {
	tests := []struct {
		name     string
		sender   testbed.DataSender
		receiver testbed.DataReceiver
	}{
		{
			"JaegerGRPC",
			testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewJaegerDataReceiver(testbed.GetAvailablePort(t)),
		},
		{
			"OTLP",
			testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			require.NoError(t, err)

			// Use processor to add attributes to certain spans.
			processors := map[string]string{
				"batch": `
  batch:
`,
				"attributes": `
  attributes:
    include:
      match_type: regexp
      services: ["service-to-add.*"]
      span_names: ["span-to-add-.*"]
    actions:
      - action: insert
        key: "new_attr"
        value: "string value"
`,
			}

			agentProc := &testbed.ChildProcess{}
			configStr := createConfigYaml(t, test.sender, test.receiver, resultDir, processors, nil)
			configCleanup, err := agentProc.PrepareConfig(configStr)
			require.NoError(t, err)
			defer configCleanup()

			options := testbed.LoadOptions{DataItemsPerSecond: 10000, ItemsPerBatch: 10}
			dataProvider := testbed.NewPerfTestDataProvider(options)
			tc := testbed.NewTestCase(
				t,
				dataProvider,
				test.sender,
				test.receiver,
				agentProc,
				&testbed.PerfTestValidator{},
				performanceResultsSummary,
			)
			defer tc.Stop()

			tc.StartBackend()
			tc.StartAgent()
			defer tc.StopAgent()

			tc.EnableRecording()

			require.NoError(t, test.sender.Start())

			// Create a span that matches "include" filter.
			spanToInclude := "span-to-add-attr"
			// Create a service name that matches "include" filter.
			nodeToInclude := "service-to-add-attr"

			// verifySpan verifies that attributes was added to the internal data span.
			verifySpan := func(span pdata.Span) {
				require.NotNil(t, span)
				require.Equal(t, span.Attributes().Len(), 1)
				attrVal, ok := span.Attributes().Get("new_attr")
				assert.True(t, ok)
				assert.EqualValues(t, "string value", attrVal.StringVal())
			}

			verifySingleSpan(t, tc, nodeToInclude, spanToInclude, verifySpan)

			// Create a service name that does not match "include" filter.
			nodeToExclude := "service-not-to-add-attr"

			verifySingleSpan(t, tc, nodeToExclude, spanToInclude, func(span pdata.Span) {
				// Verify attributes was not added to the new internal data span.
				assert.Equal(t, span.Attributes().Len(), 0)
			})

			// Create another span that does not match "include" filter.
			spanToExclude := "span-not-to-add-attr"
			verifySingleSpan(t, tc, nodeToInclude, spanToExclude, func(span pdata.Span) {
				// Verify attributes was not added to the new internal data span.
				assert.Equal(t, span.Attributes().Len(), 0)
			})
		})
	}
}

func TestMetricsFromFile(t *testing.T) {
	// This test demonstrates usage of NewFileDataProvider to generate load using
	// previously recorded data.

	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	require.NoError(t, err)

	// Use metrics previously recorded using "file" exporter and "k8scluster" receiver.
	dataProvider, err := testbed.NewFileDataProvider("testdata/k8s-metrics.json", configmodels.MetricsDataType)
	assert.NoError(t, err)

	options := testbed.LoadOptions{
		DataItemsPerSecond: 1_000,
		Parallel:           1,
		// ItemsPerBatch is based on the data from the file.
		ItemsPerBatch: dataProvider.ItemsPerBatch,
	}
	agentProc := &testbed.ChildProcess{}

	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))

	configStr := createConfigYaml(t, sender, receiver, resultDir, nil, nil)
	configCleanup, err := agentProc.PrepareConfig(configStr)
	require.NoError(t, err)
	defer configCleanup()

	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		agentProc,
		&testbed.PerfTestValidator{},
		performanceResultsSummary,
	)
	defer tc.Stop()

	tc.SetResourceLimits(testbed.ResourceSpec{
		ExpectedMaxCPU: 120,
		ExpectedMaxRAM: 70,
	})
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
