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

package correctness

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/collector/testbed/testbed"
)

// CreateConfigYaml creates a yaml config for an otel collector given a testbed sender, testbed receiver, any
// processors, and a pipeline type. A collector created from the resulting yaml string should be able to talk
// the specified sender and receiver.
func CreateConfigYaml(
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	processors map[string]string,
	pipelineType string,
) string {

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
    %s:
      receivers: [%v]
      processors: [%s]
      exporters: [%v]
`

	return fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		processorsSections,
		pipelineType,
		sender.ProtocolName(),
		processorsList,
		receiver.ProtocolName(),
	)
}

// PipelineDef holds the information necessary to run a single testbed configuration.
type PipelineDef struct {
	Receiver     string
	Exporter     string
	TestName     string
	DataSender   testbed.DataSender
	DataReceiver testbed.DataReceiver
	ResourceSpec testbed.ResourceSpec
}

// LoadPictOutputPipelineDefs generates a slice of PipelineDefs from the passed-in generated PICT file. The
// result should be a set of PipelineDefs that covers all possible pipeline configurations.
func LoadPictOutputPipelineDefs(fileName string) ([]PipelineDef, error) {
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
		if s[0] == "Receiver" {
			continue
		}

		var aDef PipelineDef
		aDef.Receiver, aDef.Exporter = s[0], s[1]
		defs = append(defs, aDef)
	}

	return defs, err
}

// ConstructTraceSender creates a testbed trace sender from the passed-in trace sender identifier.
func ConstructTraceSender(t *testing.T, receiver string) testbed.DataSender {
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

// ConstructMetricsSender creates a testbed metrics sender from the passed-in metrics sender identifier.
func ConstructMetricsSender(t *testing.T, receiver string) testbed.MetricDataSender {
	var sender testbed.MetricDataSender
	switch receiver {
	case "otlp":
		sender = testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	case "opencensus":
		sender = testbed.NewOCMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	case "prometheus":
		sender = testbed.NewPrometheusDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	default:
		t.Errorf("unknown receiver type: %s", receiver)
	}
	return sender
}

// ConstructReceiver creates a testbed receiver from the passed-in recevier identifier.
func ConstructReceiver(t *testing.T, exporter string) testbed.DataReceiver {
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
	case "prometheus":
		receiver = testbed.NewPrometheusDataReceiver(testbed.GetAvailablePort(t))
	default:
		t.Errorf("unknown exporter type: %s", exporter)
	}
	return receiver
}
