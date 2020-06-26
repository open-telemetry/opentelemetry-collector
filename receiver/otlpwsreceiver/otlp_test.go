// Copyright The OpenTelemetry Authors
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

//lint:file-ignore U1000 t.Skip() flaky test causes unused function warning.

package otlpwsreceiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlpwsexporter"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/observability/observabilitytest"
	"go.opentelemetry.io/collector/testutil"
	"go.opentelemetry.io/collector/translator/conventions"
)

const otlpReceiver = "otlp_receiver_test"

const (
	asSubContentType = true
	asContentType    = false
)

func TestStopWithoutStartNeverCrashes(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ocr, err := New(zap.NewNop(), otlpReceiver, "tcp", addr, nil, nil)
	require.NoError(t, err, "Failed to create an OpenCensus receiver: %v", err)
	// Stop it before ever invoking Start*.
	ocr.Shutdown(context.Background())
}

func TestNewPortAlreadyUsed(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen on %q: %v", addr, err)
	defer ln.Close()

	r, err := New(zap.NewNop(), otlpReceiver, "tcp", addr, nil, nil)
	require.Error(t, err)
	require.Nil(t, r)
}

func TestMultipleStopReceptionShouldNotError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r, err := New(zap.NewNop(), otlpReceiver, "tcp", addr, nil, nil)
	r.traceConsumer = new(exportertest.SinkTraceExporter)
	r.metricsConsumer = new(exportertest.SinkMetricsExporter)
	require.NoError(t, err)
	require.NotNil(t, r)

	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, r.Shutdown(context.Background()))
}

func TestStartWithoutConsumersShouldFail(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	r, err := New(zap.NewNop(), otlpReceiver, "tcp", addr, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	require.Error(t, r.Start(context.Background(), componenttest.NewNopHost()))
}

func tempSocketName(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "sock")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}

// TestOTLPReceiverTrace_HandleNextConsumerResponse checks if the trace receiver
// is returning the proper response (return and metrics) when the next consumer
// in the pipeline reports error. The test changes the responses returned by the
// next trace consumer, checks if data was passed down the pipeline and if
// proper metrics were recorded. It also uses all endpoints supported by the
// trace receiver.
func TestOTLPReceiverTrace_HandleNextConsumerResponse(t *testing.T) {
	type ingestionStateTest struct {
		okToIngest   bool
		expectedCode codes.Code
	}
	tests := []struct {
		name                         string
		expectedReceivedBatches      int
		expectedIngestionBlockedRPCs int
		ingestionStates              []ingestionStateTest
	}{
		{
			name:                         "IngestTest",
			expectedReceivedBatches:      2,
			expectedIngestionBlockedRPCs: 1,
			ingestionStates: []ingestionStateTest{
				{
					okToIngest:   true,
					expectedCode: codes.OK,
				},
				//{
				//	okToIngest:   false,
				//	expectedCode: codes.Unknown,
				//},
			},
		},
	}

	addr := testutil.GetAvailableLocalAddress(t)
	req := &collectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   conventions.AttributeServiceName,
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test-svc"}},
						},
					},
				},
				InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
					{
						Spans: []*otlptrace.Span{
							{
								TraceId: []byte{
									0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
									0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
								},
							},
						},
					},
				},
			},
		},
	}

	exportBidiFn := func(
		t *testing.T,
		cc *grpc.ClientConn,
		msg *collectortrace.ExportTraceServiceRequest) error {

		acc := collectortrace.NewTraceServiceClient(cc)
		_, err := acc.Export(context.Background(), req)

		return err
	}

	exporters := []struct {
		receiverTag string
		exportFn    func(
			t *testing.T,
			cc *grpc.ClientConn,
			msg *collectortrace.ExportTraceServiceRequest) error
	}{
		{
			receiverTag: "otlp_trace",
			exportFn:    exportBidiFn,
		},
	}
	for _, exporter := range exporters {
		for _, tt := range tests {
			t.Run(tt.name+"/"+exporter.receiverTag, func(t *testing.T) {
				doneFn := observabilitytest.SetupRecordedMetricsTest()
				defer doneFn()

				sink := new(exportertest.SinkTraceExporter)

				ocr, err := New(zap.NewNop(), otlpReceiver, "tcp", addr, sink, nil)
				require.Nil(t, err)
				require.NotNil(t, ocr)

				ocr.traceConsumer = sink
				require.Nil(t, ocr.Start(context.Background(), componenttest.NewNopHost()))
				defer ocr.Shutdown(context.Background())

				traceExporter, err := otlpwsexporter.NewTraceExporter(context.Background(), component.ExporterCreateParams{}, &otlpwsexporter.Config{Endpoint: addr})
				require.Nil(t, err)

				for _, ingestionState := range tt.ingestionStates {
					if ingestionState.okToIngest {
						sink.SetConsumeTraceError(nil)
					} else {
						sink.SetConsumeTraceError(fmt.Errorf("%q: consumer error", tt.name))
					}

					err = traceExporter.ConsumeTraces(context.Background(), pdata.TracesFromOtlp(req.ResourceSpans))

					status, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, ingestionState.expectedCode, status.Code())
				}

				//				require.Equal(t, tt.expectedReceivedBatches, len(sink.AllTraces()))
				//require.Nil(
				//	t,
				//	observabilitytest.CheckValueViewReceiverReceivedSpans(
				//		exporter.receiverTag,
				//		tt.expectedReceivedBatches),
				//)
			})
		}
	}
}
