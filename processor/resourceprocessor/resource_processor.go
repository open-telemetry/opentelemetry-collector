// Copyright The OpenTelemetry Authors
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

package resourceprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/processor/attraction"
)

type resourceTraceProcessor struct {
	attrProc *attraction.AttrProc
	next     consumer.TraceConsumer
}

func newResourceTraceProcessor(next consumer.TraceConsumer, attrProc *attraction.AttrProc) *resourceTraceProcessor {
	return &resourceTraceProcessor{
		attrProc: attrProc,
		next:     next,
	}
}

// ConsumeTraceData implements the TraceProcessor interface
func (rtp *resourceTraceProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		resource := rss.At(i).Resource()
		if resource.IsNil() {
			resource.InitEmpty()
		}
		attrs := resource.Attributes()
		rtp.attrProc.Process(attrs)
	}
	return rtp.next.ConsumeTraces(ctx, td)
}

// GetCapabilities returns the ProcessorCapabilities assocciated with the resource processor.
func (rtp *resourceTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (*resourceTraceProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*resourceTraceProcessor) Shutdown(context.Context) error {
	return nil
}

type resourceMetricProcessor struct {
	attrProc *attraction.AttrProc
	next     consumer.MetricsConsumer
}

func newResourceMetricProcessor(next consumer.MetricsConsumer, attrProc *attraction.AttrProc) *resourceMetricProcessor {
	return &resourceMetricProcessor{
		attrProc: attrProc,
		next:     next,
	}
}

// GetCapabilities returns the ProcessorCapabilities assocciated with the resource processor.
func (rmp *resourceMetricProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (*resourceMetricProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*resourceMetricProcessor) Shutdown(context.Context) error {
	return nil
}

// ConsumeMetricsData implements the MetricsProcessor interface
func (rmp *resourceMetricProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	imd := pdatautil.MetricsToInternalMetrics(md)
	rms := imd.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resource := rms.At(i).Resource()
		if resource.IsNil() {
			resource.InitEmpty()
		}
		if resource.Attributes().Len() == 0 {
			resource.Attributes().InitEmptyWithCapacity(1)
		}
		rmp.attrProc.Process(resource.Attributes())
	}
	return rmp.next.ConsumeMetrics(ctx, md)
}
