// Copyright 2018, OpenCensus Authors
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

package config_test

import (
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
	"github.com/census-instrumentation/opencensus-service/internal/config"
)

// Issue #233: Zipkin receiver and exporter loopback detection
// would mistakenly report that "localhost:9410" and "localhost:9411"
// were equal, due to a mistake in parsing out their addresses,
// but also after IP resolution, the equivalence of ports was not being
// checked.
func TestZipkinReceiverExporterLogicalConflictChecks(t *testing.T) {
	regressionYAML := []byte(`
receivers:
    zipkin:
        address: "localhost:9410"

exporters:
    zipkin:
        endpoint: "http://localhost:9411/api/v2/spans"
`)

	cfg, err := config.ParseOCAgentConfig(regressionYAML)
	if err != nil {
		t.Fatalf("Unexpected YAML parse error: %v", err)
	}
	if err := cfg.CheckLogicalConflicts(regressionYAML); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if g, w := cfg.Receivers.Zipkin.Address, "localhost:9410"; g != w {
		t.Errorf("Receivers.Zipkin.EndpointURL mismatch\nGot: %s\nWant:%s", g, w)
	}

	var ecfg struct {
		Exporters *struct {
			Zipkin *exporterparser.ZipkinConfig `yaml:"zipkin"`
		} `yaml:"exporters"`
	}
	_ = yaml.Unmarshal(regressionYAML, &ecfg)
	if g, w := ecfg.Exporters.Zipkin.EndpointURL(), "http://localhost:9411/api/v2/spans"; g != w {
		t.Errorf("Exporters.Zipkin.EndpointURL mismatch\nGot: %s\nWant:%s", g, w)
	}
}

// Issue #377: If Config.OpenCensus == nil, invoking
// CanRunOpenCensus{Metrics, Trace}Receiver() would crash.
func TestOpenCensusTraceReceiverEnabledNoCrash(t *testing.T) {
	// 1. Test with an in-code struct.
	cfg := &config.Config{
		Receivers: &config.Receivers{
			OpenCensus: nil,
		},
	}
	if cfg.CanRunOpenCensusTraceReceiver() {
		t.Fatal("CanRunOpenCensusTraceReceiver: Unexpected True for a nil Receiver.OpenCensus")
	}
	if cfg.CanRunOpenCensusMetricsReceiver() {
		t.Fatal("CanRunOpenCensusMetricsReceiver: Unexpected True for a nil Receiver.OpenCensus")
	}

	// 2. Test with a struct unmarshalled from a configuration file's YAML.
	regressionYAML := []byte(`
receivers:
    zipkin:
        address: "localhost:9410"`)

	cfg, err := config.ParseOCAgentConfig(regressionYAML)
	if err != nil {
		t.Fatalf("Unexpected YAML parse error: %v", err)
	}

	if cfg.CanRunOpenCensusTraceReceiver() {
		t.Fatal("yaml.CanRunOpenCensusTraceReceiver: Unexpected True for a nil Receiver.OpenCensus")
	}
	if cfg.CanRunOpenCensusMetricsReceiver() {
		t.Fatal("yaml.CanRunOpenCensusMetricsReceiver: Unexpected True for a nil Receiver.OpenCensus")
	}
}
