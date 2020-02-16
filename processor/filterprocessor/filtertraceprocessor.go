// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"context"

	v1 "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type filterTraceProcessor struct {
	nameFilters  filterset.FilterSet
	cfg          *Config
	capabilities processor.Capabilities
	next         consumer.TraceConsumer
}

func newFilterTraceProcessor(next consumer.TraceConsumer, cfg *Config) (*filterTraceProcessor, error) {
	factory := filterset.Factory{}
	nf, err := factory.CreateFilterSet(&cfg.Traces.NameFilter)
	if err != nil {
		return nil, err
	}

	return &filterTraceProcessor{
		nameFilters:  nf,
		cfg:          cfg,
		capabilities: processor.Capabilities{MutatesConsumedData: true},
		next:         next,
	}, nil
}

// ConsumeTraceData implements the TraceProcessor interface
func (ftp *filterTraceProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return ftp.next.ConsumeTraceData(ctx, consumerdata.TraceData{
		Node:         td.Node,
		Resource:     td.Resource,
		Spans:        ftp.filterSpans(td.Spans),
		SourceFormat: td.SourceFormat,
	})
}

// GetCapabilities returns the Capabilities assocciated with the resource processor.
func (ftp *filterTraceProcessor) GetCapabilities() processor.Capabilities {
	return ftp.capabilities
}

// Start is invoked during service startup.
func (*filterTraceProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*filterTraceProcessor) Shutdown() error {
	return nil
}

// filterSpans filters the given spans based off the filterTraceProcessor's filters.
func (ftp *filterTraceProcessor) filterSpans(spans []*v1.Span) []*v1.Span {
	keep := []*v1.Span{}
	for _, s := range spans {
		if ftp.shouldKeepSpan(s) {
			keep = append(keep, s)
		}
	}

	return keep
}

// shouldKeepSpan determines whether or not a span should be kept based off the filterTraceProcessor's filters.
func (ftp *filterTraceProcessor) shouldKeepSpan(span *v1.Span) bool {
	name := span.GetName().GetValue()
	nameMatch := ftp.nameFilters.Matches(name)
	return nameMatch == (ftp.cfg.Action == INCLUDE)
}
