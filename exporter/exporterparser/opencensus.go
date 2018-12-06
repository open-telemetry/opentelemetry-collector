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

package exporterparser

import (
	"context"
	"fmt"

	"contrib.go.opencensus.io/exporter/ocagent"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type opencensusConfig struct {
	Endpoint string `yaml:"endpoint,omitempty"`
	// TODO: add insecure, service name options.
}

type ocagentExporter struct {
	exporter *ocagent.Exporter
}

var _ exporter.TraceExporter = (*ocagentExporter)(nil)

// OpenCensusTraceExportersFromYAML parses the yaml bytes and returns an exporter.TraceExporter targeting
// OpenCensus Agent/Collector according to the configuration settings.
func OpenCensusTraceExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var cfg struct {
		Exporters *struct {
			OpenCensus *opencensusConfig `yaml:"opencensus"`
		} `yaml:"exporters"`
	}
	if err := yamlUnmarshal(config, &cfg); err != nil {
		return nil, nil, err
	}
	if cfg.Exporters == nil {
		return nil, nil, nil
	}
	ocac := cfg.Exporters.OpenCensus
	if ocac == nil {
		return nil, nil, nil
	}

	if ocac.Endpoint == "" {
		return nil, nil, fmt.Errorf("OpenCensus config requires an Endpoint")
	}

	sde, serr := ocagent.NewExporter(ocagent.WithAddress(ocac.Endpoint), ocagent.WithInsecure())
	if serr != nil {
		return nil, nil, fmt.Errorf("Cannot configure OpenCensus Trace exporter: %v", serr)
	}

	tes = append(tes, &ocagentExporter{exporter: sde})
	doneFns = append(doneFns, func() error {
		sde.Flush()
		return nil
	})
	return tes, doneFns, nil
}

func (sde *ocagentExporter) ExportSpans(ctx context.Context, td data.TraceData) error {
	return exportSpans(ctx, "ocagent", sde.exporter, td)
}
