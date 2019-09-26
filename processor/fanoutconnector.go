// Copyright 2019, OpenTelemetry Authors
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

package processor

import (
	"context"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// This file contains implementations of Trace/Metrics connectors
// that fan out the data to multiple other consumers.

// NewMetricsFanOutConnector wraps multiple metrics consumers in a single one.
func NewMetricsFanOutConnector(mcs []consumer.MetricsConsumer) MetricsProcessor {
	return metricsFanOutConnector(mcs)
}

type metricsFanOutConnector []consumer.MetricsConsumer

var _ MetricsProcessor = (*metricsFanOutConnector)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsFanOutConnector) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(mfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone, err := cloneMetricsData(&md)
		if err != nil {
			errs = append(errs, err)
			break
		} else {
			if err := mfc[i].ConsumeMetricsData(ctx, *clone); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(mfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := mfc[len(mfc)-1]
		if err := lastTc.ConsumeMetricsData(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}

	return oterr.CombineErrors(errs)
}

// NewTraceFanOutConnector wraps multiple trace consumers in a single one.
func NewTraceFanOutConnector(tcs []consumer.TraceConsumer) TraceProcessor {
	return traceFanOutConnector(tcs)
}

type traceFanOutConnector []consumer.TraceConsumer

var _ TraceProcessor = (*traceFanOutConnector)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc traceFanOutConnector) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone, err := cloneTraceData(&td)
		if err != nil {
			errs = append(errs, err)
			break
		} else {
			if err := tfc[i].ConsumeTraceData(ctx, *clone); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}

	return oterr.CombineErrors(errs)
}

func cloneTraceData(td *consumerdata.TraceData) (*consumerdata.TraceData, error) {
	clone := &consumerdata.TraceData{
		SourceFormat: td.SourceFormat,
	}

	if td.Node != nil {
		clone.Node = &commonpb.Node{}

		bytes, err := proto.Marshal(td.Node)
		if err != nil {
			return nil, err
		}

		err = proto.Unmarshal(bytes, clone.Node)
		if err != nil {
			return nil, err
		}
	}

	if td.Resource != nil {
		clone.Resource = &resourcepb.Resource{}

		bytes, err := proto.Marshal(td.Resource)
		if err != nil {
			return nil, err
		}

		err = proto.Unmarshal(bytes, clone.Resource)
		if err != nil {
			return nil, err
		}
	}

	if td.Spans != nil {
		clone.Spans = make([]*tracepb.Span, 0, len(td.Spans))

		for _, span := range td.Spans {
			if span != nil {
				bytes, err := proto.Marshal(span)
				if err != nil {
					return nil, err
				}

				var spanClone tracepb.Span
				err = proto.Unmarshal(bytes, &spanClone)
				if err != nil {
					return nil, err
				}
				clone.Spans = append(clone.Spans, &spanClone)
			} else {
				clone.Spans = append(clone.Spans, nil)
			}
		}
	}

	return clone, nil
}

func cloneMetricsData(md *consumerdata.MetricsData) (*consumerdata.MetricsData, error) {
	clone := &consumerdata.MetricsData{}

	if md.Node != nil {
		clone.Node = &commonpb.Node{}

		bytes, err := proto.Marshal(md.Node)
		if err != nil {
			return nil, err
		}

		err = proto.Unmarshal(bytes, clone.Node)
		if err != nil {
			return nil, err
		}
	}

	if md.Resource != nil {
		clone.Resource = &resourcepb.Resource{}

		bytes, err := proto.Marshal(md.Resource)
		if err != nil {
			return nil, err
		}

		err = proto.Unmarshal(bytes, clone.Resource)
		if err != nil {
			return nil, err
		}
	}

	if md.Metrics != nil {
		clone.Metrics = make([]*metricspb.Metric, 0, len(md.Metrics))

		for _, span := range md.Metrics {
			if span != nil {
				bytes, err := proto.Marshal(span)
				if err != nil {
					return nil, err
				}

				var metricClone metricspb.Metric
				err = proto.Unmarshal(bytes, &metricClone)
				if err != nil {
					return nil, err
				}
				clone.Metrics = append(clone.Metrics, &metricClone)
			} else {
				clone.Metrics = append(clone.Metrics, nil)
			}
		}
	}

	return clone, nil
}
