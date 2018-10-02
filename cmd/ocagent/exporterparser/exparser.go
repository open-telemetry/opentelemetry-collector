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

// This package provides support for parsing and creating the
// respective exporters given a YAML configuration payload.
// For now it currently only provides statically imported OpenCensus
// exporters like:
//  * Stackdriver Tracing and Monitoring
//  * DataDog
//  * Zipkin

package exporterparser

import (
	"sync"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var (
	exportersMu sync.Mutex
	exporters   []Exporter
)

func init() {
	RegisterExporter(&datadogExporter{})
	RegisterExporter(&stackdriverExporter{})
	RegisterExporter(&zipkinExporter{})
}

type Exporter interface {
	MakeExporters(config []byte) (se view.Exporter, te trace.Exporter, closer func())
}

// RegisterExporter allows users to dyanmically add additional exporters
// before the configuration is parsed. RegisterExporter should be called
// before ExportersFromYAMLConfig.
func RegisterExporter(e Exporter) {
	exportersMu.Lock()
	exporters = append(exporters, e)
	exportersMu.Unlock()
}

// ExportersFromYAMLConfig parses the config yaml payload and returns the respective exporters
func ExportersFromYAMLConfig(config []byte) (traceExporters []trace.Exporter, statsExporters []view.Exporter, closeFns []func()) {
	for _, e := range exporters {
		se, te, closer := e.MakeExporters(config)
		if se != nil {
			statsExporters = append(statsExporters, se)
		}
		if te != nil {
			traceExporters = append(traceExporters, te)
		}
		if closer != nil {
			closeFns = append(closeFns, closer)
		}
	}
	return
}
