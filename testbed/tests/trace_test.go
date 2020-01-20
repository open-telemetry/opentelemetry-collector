// Copyright 2019, OpenTelemetry Authors
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

// Package tests contains test cases. To run the tests go to tests directory and run:
// TESTBED_CONFIG=local.yaml go test -v

package tests

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
)

// TestMain is used to initiate setup, execution and tear down of testbed.
func TestMain(m *testing.M) {
	testbed.DoTestMain(m)
}

func TestTrace10kSPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"JaegerThrift",
			testbed.NewJaegerThriftDataSender(testbed.GetAvailablePort(t)),
			testbed.NewJaegerDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 52,
				ExpectedMaxRAM: 89,
			},
		},
		{
			"OpenCensus",
			testbed.NewOCTraceDataSender(testbed.GetAvailablePort(t)),
			testbed.NewOCDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 42,
				ExpectedMaxRAM: 84,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				testbed.LoadOptions{},
				test.resourceSpec,
			)
		})
	}
}

func TestTraceNoBackend10kSPSJaeger(t *testing.T) {
	tests := []struct {
		name                string
		configFileName      string
		expectedMaxRAM      uint32
		expectedMinFinalRAM uint32
	}{
		{name: "NoMemoryLimit", configFileName: "agent-config.yaml", expectedMaxRAM: 200, expectedMinFinalRAM: 100},

		// Memory limiter in memory-limiter.yaml is configured to allow max 10MiB of heap size.
		// However, heap is not the only memory user, so the total limit we set for this
		// test is 60MiB. Note: to ensure this test verifies memorylimiter correctly
		// expectedMaxRAM of this test case must be lower than expectedMinFinalRAM of the
		// previous test case (which runs without memorylimiter).
		{name: "MemoryLimiter", configFileName: "memory-limiter.yaml", expectedMaxRAM: 60, expectedMinFinalRAM: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			configFilePath := path.Join("testdata", test.configFileName)

			tc := testbed.NewTestCase(
				t,
				testbed.NewJaegerThriftDataSender(testbed.DefaultJaegerPort),
				testbed.NewOCDataReceiver(testbed.DefaultOCPort),
				testbed.WithConfigFile(configFilePath),
			)
			defer tc.Stop()

			tc.SetResourceLimits(testbed.ResourceSpec{
				ExpectedMaxCPU: 60,
				ExpectedMaxRAM: 198,
			})

			tc.StartAgent()
			tc.StartLoad(testbed.LoadOptions{DataItemsPerSecond: 10000})

			tc.Sleep(tc.Duration)

			rss, _, _ := tc.AgentMemoryInfo()
			assert.True(t, rss > test.expectedMinFinalRAM)
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
		},

		// We generate 10 attributes each with average key length of 100 bytes and
		// average value length of 50 bytes so total size of attributes values is
		// 15000 bytes.
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 120,
			expectedMaxRAM: 100,
		},

		// Approx 10 KiB attributes.
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 100,
			expectedMaxRAM: 100,
		},

		// Approx 100 KiB attributes.
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 250,
			expectedMaxRAM: 100,
		},
	})
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
		},
		{
			attrCount:      100,
			attrSizeByte:   50,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
		},
		{
			attrCount:      10,
			attrSizeByte:   1000,
			expectedMaxCPU: 80,
			expectedMaxRAM: 2000,
		},
		{
			attrCount:      20,
			attrSizeByte:   5000,
			expectedMaxCPU: 120,
			expectedMaxRAM: 2000,
		},
	})
}

func TestTraceBallast1kSPSAddAttrs(t *testing.T) {
	args := []string{"--mem-ballast-size-mib", "1000"}
	Scenario1kSPSWithAttrs(
		t,
		args,
		[]TestCase{
			{
				attrCount:      0,
				attrSizeByte:   0,
				expectedMaxCPU: 30,
				expectedMaxRAM: 2000,
			},
			{
				attrCount:      100,
				attrSizeByte:   50,
				expectedMaxCPU: 80,
				expectedMaxRAM: 2000,
			},
			{
				attrCount:      10,
				attrSizeByte:   1000,
				expectedMaxCPU: 80,
				expectedMaxRAM: 2000,
			},
			{
				attrCount:      20,
				attrSizeByte:   5000,
				expectedMaxCPU: 120,
				expectedMaxRAM: 2000,
			},
		},
		testbed.WithConfigFile(path.Join("testdata", "add-attributes-config.yaml")),
	)
}

// verifySingleSpan sends a single span to Collector, waits until the span is forwarded
// and received by MockBackend and calls user-supplied verification function on
// received span.
func verifySingleSpan(
	t *testing.T,
	tc *testbed.TestCase,
	span *tracepb.Span,
	verifyReceived func(span *tracepb.Span),
) {

	// Clear previously received traces.
	tc.MockBackend.ClearReceivedItems()
	startCounter := tc.MockBackend.DataItemsReceived()

	// Add dummy IDs
	span.TraceId = testbed.GenerateTraceID(1)
	span.SpanId = testbed.GenerateSpanID(1)
	td := consumerdata.TraceData{Spans: []*tracepb.Span{span}}

	sender := tc.Sender.(testbed.TraceDataSender)

	// Send the span.
	sender.SendSpans(td)

	// We bypass the load generator in this test, but make sure to increment the
	// counter since it is used in final reports.
	tc.LoadGenerator.IncDataItemsSent()

	// Wait until span is received.
	tc.WaitFor(func() bool { return tc.MockBackend.DataItemsReceived() == startCounter+1 },
		"span received")

	// Verify received span.
	count := 0
	for _, td := range tc.MockBackend.ReceivedTraces {
		for _, span := range td.Spans {
			verifyReceived(span)
			count++
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
			"JaegerThrift",
			testbed.NewJaegerThriftDataSender(testbed.GetAvailablePort(t)),
			testbed.NewJaegerDataReceiver(testbed.GetAvailablePort(t)),
		},
		{
			"OpenCensus",
			testbed.NewOCTraceDataSender(testbed.GetAvailablePort(t)),
			testbed.NewOCDataReceiver(testbed.GetAvailablePort(t)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			if err != nil {
				t.Fatal(err)
			}

			// Use processor to add attributes to certain spans.
			processors := map[string]string{
				"attributes": `
  attributes:
    include:
      span_names: ["span-to-add-.*"]
    actions:
      - action: insert
        key: "new_attr"
        value: "string value"
`,
			}

			configFile := createConfigFile(test.sender, test.receiver, resultDir, processors)
			defer os.Remove(configFile)

			if configFile == "" {
				t.Fatal("Cannot create config file")
			}

			tc := testbed.NewTestCase(t, test.sender, test.receiver, testbed.WithConfigFile(configFile))
			defer tc.Stop()

			tc.StartBackend()
			tc.StartAgent()
			defer tc.StopAgent()

			tc.EnableRecording()

			sender := test.sender.(testbed.TraceDataSender)
			sender.Start()

			// Create a span that matches "include" filter.
			span := &tracepb.Span{
				Name: &tracepb.TruncatableString{Value: "span-to-add-attr"},
			}

			verifySingleSpan(t, tc, span, func(span *tracepb.Span) {
				// Verify attributes was added.
				attrVal, ok := span.Attributes.AttributeMap["new_attr"]
				assert.True(t, ok)
				assert.NotNil(t, attrVal)
				assert.EqualValues(t, "string value", attrVal.GetStringValue().Value)
			})

			// Create another span that does not match "include" filter.
			span = &tracepb.Span{
				Name: &tracepb.TruncatableString{Value: "span-not-to-add-attr"},
			}
			verifySingleSpan(t, tc, span, func(span *tracepb.Span) {
				// Verify attributes was not added.
				assert.Nil(t, span.Attributes)
			})
		})
	}
}
