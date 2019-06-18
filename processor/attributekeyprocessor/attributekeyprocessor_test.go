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

package attributekeyprocessor

import (
	"context"
	"reflect"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/google/go-cmp/cmp"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/processor/processortest"
)

func TestNewTraceProcessor(t *testing.T) {
	nopProcessor := processortest.NewNopTraceProcessor(nil)
	type args struct {
		nextConsumer consumer.TraceConsumer
		replacements []KeyReplacement
	}
	tests := []struct {
		name    string
		args    args
		want    processor.TraceProcessor
		wantErr bool
	}{
		{
			name:    "nextConsumer_nil",
			wantErr: true,
		},
		{
			name: "empty_replacements",
			args: args{
				nextConsumer: nopProcessor,
			},
			want: &attributekeyprocessor{
				nextConsumer: nopProcessor,
			},
		},
		{
			name: "duplicated_key",
			args: args{
				nextConsumer: nopProcessor,
				replacements: []KeyReplacement{
					{
						Key:    "foo",
						NewKey: "baz",
					},
					{
						Key:    "bar",
						NewKey: "biz",
					},
					{
						Key:    "foo",
						NewKey: "bit",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "key_cycle",
			args: args{
				nextConsumer: nopProcessor,
				replacements: []KeyReplacement{
					{
						Key:    "foo",
						NewKey: "bar",
					},
					{
						Key:    "bar",
						NewKey: "foo",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "happy_path",
			args: args{
				nextConsumer: nopProcessor,
				replacements: []KeyReplacement{
					{
						Key:    "foo",
						NewKey: "biz",
					},
					{
						Key:    "bar",
						NewKey: "biz",
					},
				},
			},
			want: &attributekeyprocessor{
				nextConsumer: nopProcessor,
				replacements: []KeyReplacement{
					{
						Key:    "foo",
						NewKey: "biz",
					},
					{
						Key:    "bar",
						NewKey: "biz",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTraceProcessor(tt.args.nextConsumer, tt.args.replacements...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTraceProcessor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTraceProcessor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_attributekeyprocessor_ConsumeTraceData(t *testing.T) {
	tests := []struct {
		name string
		args []KeyReplacement
		td   data.TraceData
		want []data.TraceData
	}{
		{
			name: "keyMap_nil",
			want: []data.TraceData{
				{},
			},
		},
		{
			name: "span_nil",
			args: []KeyReplacement{
				{
					Key:    "foo",
					NewKey: "bar",
				},
			},
			td: data.TraceData{
				Spans: make([]*tracepb.Span, 1, 1),
			},
			want: []data.TraceData{
				{
					Spans: make([]*tracepb.Span, 1, 1),
				},
			},
		},
		{
			name: "span_Attributes_nil",
			args: []KeyReplacement{
				{
					Key:    "foo",
					NewKey: "bar",
				},
			},
			td: data.TraceData{
				Spans: []*tracepb.Span{
					{
						Name: &tracepb.TruncatableString{Value: "test"},
					},
				},
			},
			want: []data.TraceData{
				{
					Spans: []*tracepb.Span{
						{
							Name: &tracepb.TruncatableString{Value: "test"},
						},
					},
				},
			},
		},
		{
			name: "span_Attributes_AttributeMap_nil",
			args: []KeyReplacement{
				{
					Key:    "foo",
					NewKey: "bar",
				},
			},
			td: data.TraceData{
				Spans: []*tracepb.Span{
					{
						Name:       &tracepb.TruncatableString{Value: "test"},
						Attributes: &tracepb.Span_Attributes{},
					},
				},
			},
			want: []data.TraceData{
				{
					Spans: []*tracepb.Span{
						{
							Name:       &tracepb.TruncatableString{Value: "test"},
							Attributes: &tracepb.Span_Attributes{},
						},
					},
				},
			},
		},
		{
			name: "span_replace_key",
			args: []KeyReplacement{
				{
					Key:          "foo",
					NewKey:       "bar",
					KeepOriginal: true,
				},
				{
					Key:    "servertracer.http.responsecode",
					NewKey: "http.status_code",
				},
				{
					Key:       "servertracer.http.responsephrase",
					NewKey:    "http.message",
					Overwrite: true,
				},
				{
					Key:          "server.process.instance",
					NewKey:       "process",
					Overwrite:    true,
					KeepOriginal: true,
				},
			},
			td: data.TraceData{
				Spans: []*tracepb.Span{
					{
						Name: &tracepb.TruncatableString{Value: "test"},
						Attributes: &tracepb.Span_Attributes{
							AttributeMap: map[string]*tracepb.AttributeValue{
								"servertracer.http.responsecode": {
									Value: &tracepb.AttributeValue_IntValue{IntValue: 503},
								},
								"servertracer.http.responsephrase": {
									Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "Service Unavailable"}},
								},
								"http.message": {
									Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "Server Internal Error"}},
								},
								"server.process.instance": {
									Value: &tracepb.AttributeValue_IntValue{IntValue: 3},
								},
								"process": {
									Value: &tracepb.AttributeValue_IntValue{IntValue: 5},
								},
								"foo": {
									Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
								},
								"biz": {
									Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
								},
							},
						},
					},
				},
			},
			want: []data.TraceData{
				{
					Spans: []*tracepb.Span{
						{
							Name: &tracepb.TruncatableString{Value: "test"},
							Attributes: &tracepb.Span_Attributes{
								AttributeMap: map[string]*tracepb.AttributeValue{
									"http.status_code": {
										Value: &tracepb.AttributeValue_IntValue{IntValue: 503},
									},
									"http.message": {
										Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "Service Unavailable"}},
									},
									"server.process.instance": {
										Value: &tracepb.AttributeValue_IntValue{IntValue: 3},
									},
									"process": {
										Value: &tracepb.AttributeValue_IntValue{IntValue: 3},
									},
									"bar": {
										Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
									},
									"foo": {
										Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
									},
									"biz": {
										Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sinkExporter := &exportertest.SinkTraceExporter{}
			akp, err := NewTraceProcessor(sinkExporter, tt.args...)
			if err != nil {
				t.Errorf("NewTraceProcessor() error = %v, want nil", err)
				return
			}

			if err := akp.ConsumeTraceData(context.Background(), tt.td); err != nil {
				t.Fatalf("ConsumeTraceData() error = %v, want nil", err)
			}

			if diff := cmp.Diff(sinkExporter.AllTraces(), tt.want); diff != "" {
				t.Errorf("Mismatched TraceData\n-Got +Want:\n\t%s", diff)
			}
		})
	}
}
