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

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"log"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/testbed/testbed"
)

func TestMetricNoBackend10kDPSOpenCensus(t *testing.T) {
	options := testbed.LoadOptions{DataItemsPerInterval: 10_000, ItemsPerBatch: 10}
	dataProvider := testbed.NewPerfTestDataProvider(options)
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		testbed.NewOCMetricDataSender(testbed.DefaultHost, 55678),
		testbed.NewOCDataReceiver(testbed.DefaultOCPort),
		&testbed.ChildProcess{},
		&testbed.PerfTestValidator{},
		performanceResultsSummary,
	)
	defer tc.Stop()

	tc.SetResourceLimits(testbed.ResourceSpec{ExpectedMaxCPU: 200, ExpectedMaxRAM: 200})
	tc.StartAgent()

	tc.StartLoad(testbed.LoadOptions{DataItemsPerInterval: 10_000})

	tc.Sleep(tc.Duration)
}

func TestMetric10kDPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"OpenCensus",
			testbed.NewOCMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOCDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 85,
				ExpectedMaxRAM: 75,
			},
		},
		{
			"OTLP",
			testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 65,
			},
		},
		{
			"OTLP-HTTP",
			testbed.NewOTLPHTTPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			testbed.NewOTLPHTTPDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 65,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				nil,
				nil,
			)
		})
	}

}

func TestScenario10kScrapeItemsPerInterval(t *testing.T) {
	//scrapeInterval := testbed.GetScrapeInterval()
	scrapeInterval := 15
	sender := testbed.NewPrometheusDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t), scrapeInterval)
	receiver := testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))
	resourceSpec := testbed.ResourceSpec{
			ExpectedMaxCPU: 650,
			ExpectedMaxRAM: 5000,
	}
	processors := map[string]string {
			"batch": `batch:
    send_batch_max_size: 5000
`,
		}

	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	require.NoError(t, err)

	itemsPerInterval := testbed.GetItemsPerInterval()
	// Because there are 7 datapoints per metric and a 15s scrape interval, this amounts to 10k items per second
	//itemsPerInterval := 21428
	options := testbed.LoadOptions{
		DataItemsPerInterval: itemsPerInterval,
		Interval:             scrapeInterval,
		ItemsPerBatch:        itemsPerInterval,
		Parallel:             1,
		IsScraping:           true,
	}
	agentProc := &testbed.ChildProcess{}

	timeToWait := scrapeInterval
	if timeToWait < 10 {
		timeToWait = 10
	}

	configStr := createConfigYaml(t, sender, receiver, resultDir, processors, nil)
	log.Printf(configStr)

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
		&testbed.PerfTestValidator{IsScraping: true, ScrapeInterval: scrapeInterval},
		performanceResultsSummary,
	)
	defer tc.Stop()

	// EnableRecording enables recording of all data received by MockBackend.
	tc.MockBackend.EnableMetricTimestampRecording()
	tc.SetResourceLimits(resourceSpec)
	tc.StartBackend()
	tc.StartAgent("--log-level=debug")

	tc.StartLoad(options)

	tc.Sleep(tc.Duration)

	tc.StopLoad()

	tc.WaitForN(func() bool { return tc.LoadGenerator.DataItemsSent() > 0 }, 
		time.Second * time.Duration(timeToWait),
		"load generator started")
	tc.WaitForN(func() bool { return tc.LoadGenerator.DataItemsSent() <= tc.MockBackend.DataItemsReceived() },
		time.Second * time.Duration(timeToWait),
		"all data items received")

	tc.StopAgent()

	tc.ValidateData()
}
