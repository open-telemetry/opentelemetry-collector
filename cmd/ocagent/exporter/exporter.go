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

package exporter

import (
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var (
	exporters []Exporter

	statsExporters []view.Exporter
	traceExporters []trace.Exporter
	closers        []func()
)

func init() {
	RegisterExporter(&datadogExporter{})
	RegisterExporter(&stackdriverExporter{})
	RegisterExporter(&zipkinExporter{})
}

type Exporter interface {
	MakeExporters(config []byte) (se view.Exporter, te trace.Exporter, closer func())
}

// RegisterExporter allows users to export additional exporters
// before configuration is parsed. RegisterExporter should be called
// before Parse.
func RegisterExporter(e Exporter) {
	exporters = append(exporters, e)
}

// Parse parses the config yaml payload and configures
// the exporters accordingly.
func Parse(config []byte) {
	for _, e := range exporters {
		se, te, closer := e.MakeExporters(config)
		if se != nil {
			statsExporters = append(statsExporters, se)
		}
		if te != nil {
			traceExporters = append(traceExporters, te)
		}
		if closer != nil {
			closers = append(closers, closer)
		}
	}
}

// ExportView exports the view data to all registered view exporters.
func ExportView(vs *view.Data) {
	// TODO(jbd): Implement.
}

// ExportSpan exports the span data to all registered trace exporters.
func ExportSpan(sd *trace.SpanData) {
	for _, exporter := range traceExporters {
		exporter.ExportSpan(sd)
	}
}

// CloseAll closes all of the exporters.
func CloseAll() {
	for _, fn := range closers {
		fn()
	}
}
