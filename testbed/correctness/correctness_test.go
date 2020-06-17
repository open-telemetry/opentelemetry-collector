// Copyright 2020, OpenTelemetry Authors
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

package correctness

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/service/defaultcomponents"
	"go.opentelemetry.io/collector/testbed/testbed"
)

type PipelineDef struct {
	receiver     string
	exporter     string
	testName     string
	dataSender   testbed.DataSender
	dataReceiver testbed.DataReceiver
	resourceSpec testbed.ResourceSpec
}

var correctnessResults testbed.TestResultsSummary = &testbed.CorrectnessResults{}

func TestMain(m *testing.M) {
	testbed.DoTestMain(m, correctnessResults)
}

func TestTracingGoldenData(t *testing.T) {
	tests, err := loadPictOutputPipelineDefs("testdata/generated_pict_pairs_traces_pipeline.txt")
	assert.NoError(t, err)
	processors := map[string]string{
		"batch": `
  batch:
    send_batch_size: 1024
`,
	}
	for _, test := range tests {
		test.testName = fmt.Sprintf("%s-%s", test.receiver, test.exporter)
		test.dataSender = constructSender(t, test.receiver)
		test.dataReceiver = constructReceiver(t, test.exporter)
		t.Run(test.testName, func(t *testing.T) {
			testWithTracingGoldenDataset(t, test.dataSender, test.dataReceiver, test.resourceSpec, processors)
		})
	}
}

func constructSender(t *testing.T, receiver string) testbed.DataSender {
	var sender testbed.DataSender
	switch receiver {
	case "otlp":
		sender = testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	case "opencensus":
		sender = testbed.NewOCTraceDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	case "jaeger":
		sender = testbed.NewJaegerGRPCDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	case "zipkin":
		sender = testbed.NewZipkinDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	default:
		t.Errorf("unknown receiver type: %s", receiver)
	}
	return sender
}

func constructReceiver(t *testing.T, exporter string) testbed.DataReceiver {
	var receiver testbed.DataReceiver
	switch exporter {
	case "otlp":
		receiver = testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))
	case "opencensus":
		receiver = testbed.NewOCDataReceiver(testbed.GetAvailablePort(t))
	case "jaeger":
		receiver = testbed.NewJaegerDataReceiver(testbed.GetAvailablePort(t))
	case "zipkin":
		receiver = testbed.NewZipkinDataReceiver(testbed.GetAvailablePort(t))
	default:
		t.Errorf("unknown exporter type: %s", exporter)
	}
	return receiver
}

func loadPictOutputPipelineDefs(fileName string) ([]PipelineDef, error) {
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	defs := make([]PipelineDef, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), "\t")
		if "Receiver" == s[0] {
			continue
		}

		var aDef PipelineDef
		aDef.receiver, aDef.exporter = s[0], s[1]
		defs = append(defs, aDef)
	}

	return defs, err
}

func testWithTracingGoldenDataset(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resourceSpec testbed.ResourceSpec,
	processors map[string]string,
) {
	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	require.NoError(t, err)

	dataProvider := testbed.NewGoldenDataProvider(
		"../../internal/goldendataset/testdata/generated_pict_pairs_traces.txt",
		"../../internal/goldendataset/testdata/generated_pict_pairs_spans.txt",
		161803)
	factories, err := defaultcomponents.Components()
	assert.NoError(t, err)
	runner := testbed.NewInProcessCollector(factories, sender.GetCollectorPort())
	validator := testbed.NewCorrectTestValidator(dataProvider)
	config := createConfigYaml(t, sender, receiver, resultDir, processors)
	configCleanup, cfgErr := runner.PrepareConfig(config)
	assert.NoError(t, cfgErr)
	defer configCleanup()
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		runner,
		validator,
		correctnessResults,
	)
	defer tc.Stop()

	tc.SetResourceLimits(resourceSpec)
	tc.EnableRecording()
	tc.StartBackend()
	tc.StartAgent("--metrics-level=NONE")

	tc.StartLoad(testbed.LoadOptions{
		DataItemsPerSecond: 1024,
		ItemsPerBatch:      1,
	})

	duration := time.Second
	tc.Sleep(duration)

	tc.StopLoad()

	tc.WaitForN(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
		duration, "all data items received")

	tc.StopAgent()

	tc.ValidateData()
}

func createConfigYaml(t *testing.T, sender testbed.DataSender, receiver testbed.DataReceiver, resultDir string,
	processors map[string]string) string {

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

	format := `
receivers:%v
exporters:%v
processors:
  %s

extensions:

service:
  extensions:
  pipelines:
    traces:
      receivers: [%v]
      processors: [%s]
      exporters: [%v]
`

	return fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		processorsSections,
		sender.ProtocolName(),
		processorsList,
		receiver.ProtocolName(),
	)
}
