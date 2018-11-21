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

// Package exporterparser provides support for parsing and creating the
// respective exporters given a YAML configuration payload.
// For now it currently only provides statically imported OpenCensus
// exporters like:
//  * Stackdriver Tracing and Monitoring
//  * DataDog
//  * Zipkin
package exporterparser

import (
	"context"
	"fmt"

	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal"
	tracetranslator "github.com/census-instrumentation/opencensus-service/translator/trace"
)

func exportSpans(ctx context.Context, exporterName string, te trace.Exporter, td data.TraceData) error {
	var errs []error
	var goodSpans []*tracepb.Span
	for _, span := range td.Spans {
		sd, err := tracetranslator.ProtoSpanToOCSpanData(span)
		if err == nil {
			te.ExportSpan(sd)
			goodSpans = append(goodSpans, span)
		} else {
			errs = append(errs, err)
		}
	}

	// And finally record metrics on the number of exported spans.
	nSpansCounter := internal.NewExportedSpansRecorder(exporterName)
	nSpansCounter(ctx, td.Node, goodSpans)

	return internal.CombineErrors(errs)
}

func yamlUnmarshal(yamlBlob []byte, dest interface{}) error {
	if err := yaml.Unmarshal(yamlBlob, dest); err != nil {
		return fmt.Errorf("Cannot YAML unmarshal data: %v", err)
	}
	return nil
}
