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

package obsreport

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/observability/observabilitytest"
)

const (
	exporter   = "fakeExporter"
	receiver   = "fakeReicever"
	transport  = "fakeTransport"
	format     = "fakeFormat"
	legacyName = "fakeLegacyName"
)

var (
	errFake = errors.New("errFake")
)

func TestConfigure(t *testing.T) {
	type args struct {
		generateLegacy bool
		generateNew    bool
	}
	tests := []struct {
		name      string
		args      args
		wantViews []*view.View
	}{
		{
			name: "none",
		},
		{
			name: "legacy_only",
			args: args{
				generateLegacy: true,
			},
			wantViews: observability.AllViews,
		},
		{
			name: "new_only",
			args: args{
				generateNew: true,
			},
			wantViews: genAllViews(),
		},
		{
			name: "new_only",
			args: args{
				generateNew:    true,
				generateLegacy: true,
			},
			wantViews: append(
				observability.AllViews,
				genAllViews()...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotViews := Configure(tt.args.generateLegacy, tt.args.generateNew)
			assert.Equal(t, tt.wantViews, gotViews)
		})
	}
}

func Test_obsreport_ReceiveTraceDataOp(t *testing.T) {
	doneFn, err := setupViews()
	defer doneFn()
	require.NoError(t, err)

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	receiverCtx := ReceiverContext(parentCtx, receiver, transport, legacyName)
	errs := []error{nil, errFake}
	rcvdSpans := []int{13, 42}
	for i, err := range errs {
		ctx, span := StartTraceDataReceiveOp(receiverCtx, receiver, transport)
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)

		EndTraceDataReceiveOp(
			ctx,
			span,
			format,
			rcvdSpans[i],
			err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var acceptedSpans, refusedSpans int
	for i, span := range spans {
		assert.Equal(t, receiverPrefix+receiver+receiveTraceDataOperationSuffix, span.Name)
		assert.Equal(t, transport, span.Attributes[TransportKey])
		switch errs[i] {
		case nil:
			acceptedSpans += rcvdSpans[i]
			assert.Equal(t, int64(rcvdSpans[i]), span.Attributes[AcceptedSpansKey])
			assert.Equal(t, int64(0), span.Attributes[RefusedSpansKey])
			assert.Equal(t, okStatus, span.Status)
		case errFake:
			refusedSpans += rcvdSpans[i]
			assert.Equal(t, int64(0), span.Attributes[AcceptedSpansKey])
			assert.Equal(t, int64(rcvdSpans[i]), span.Attributes[RefusedSpansKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	// Check legacy metrics.
	assert.NoError(t, observabilitytest.CheckValueViewReceiverReceivedSpans(legacyName, acceptedSpans))
	assert.NoError(t, observabilitytest.CheckValueViewReceiverDroppedSpans(legacyName, refusedSpans))
	// Check new metrics.
	receiverTags := receiverViewTags(receiver, transport)
	checkValueForSumView(t, mReceiverAcceptedSpans.Name(), receiverTags, acceptedSpans)
	checkValueForSumView(t, mReceiverRefusedSpans.Name(), receiverTags, refusedSpans)
}

func Test_obsreport_ReceiveMetricsOp(t *testing.T) {
	doneFn, err := setupViews()
	defer doneFn()
	require.NoError(t, err)

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	receiverCtx := ReceiverContext(parentCtx, receiver, transport, legacyName)
	errs := []error{errFake, nil}
	rcvdMetricPts := []int{23, 29}
	rcvdTimeSeries := []int{2, 3}
	for i, err := range errs {
		ctx, span := StartMetricsReceiveOp(receiverCtx, receiver, transport)
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)

		EndMetricsReceiveOp(
			ctx,
			span,
			format,
			rcvdMetricPts[i],
			rcvdTimeSeries[i],
			err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var receivedTimeSeries, droppedTimeSeries int
	var acceptedMetricPoints, refusedMetricPoints int
	for i, span := range spans {
		assert.Equal(t, receiverPrefix+receiver+receiverMetricsOperationSuffix, span.Name)
		assert.Equal(t, transport, span.Attributes[TransportKey])
		switch errs[i] {
		case nil:
			receivedTimeSeries += rcvdTimeSeries[i]
			acceptedMetricPoints += rcvdMetricPts[i]
			assert.Equal(t, int64(rcvdMetricPts[i]), span.Attributes[AcceptedMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[RefusedMetricPointsKey])
			assert.Equal(t, okStatus, span.Status)
		case errFake:
			droppedTimeSeries += rcvdTimeSeries[i]
			refusedMetricPoints += rcvdMetricPts[i]
			assert.Equal(t, int64(0), span.Attributes[AcceptedMetricPointsKey])
			assert.Equal(t, int64(rcvdMetricPts[i]), span.Attributes[RefusedMetricPointsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	// Check legacy metrics.
	assert.NoError(t, observabilitytest.CheckValueViewReceiverReceivedTimeSeries(legacyName, receivedTimeSeries))
	assert.NoError(t, observabilitytest.CheckValueViewReceiverDroppedTimeSeries(legacyName, droppedTimeSeries))
	// Check new metrics.
	receiverTags := receiverViewTags(receiver, transport)
	checkValueForSumView(t, mReceiverAcceptedMetricPoints.Name(), receiverTags, acceptedMetricPoints)
	checkValueForSumView(t, mReceiverRefusedMetricPoints.Name(), receiverTags, refusedMetricPoints)
}

func Test_obsreport_ExportTraceDataOp(t *testing.T) {
	doneFn, err := setupViews()
	defer doneFn()
	require.NoError(t, err)

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	// observabilitytest for exporters expects the context to flow the original
	// receiver tags, adding that to parent span.
	parentCtx = observability.ContextWithReceiverName(parentCtx, receiver)

	exporterCtx := ExporterContext(parentCtx, exporter)
	errs := []error{nil, errFake}
	numExportedSpans := []int{22, 14}
	for i, err := range errs {
		ctx, span := StartTraceDataExportOp(exporterCtx, exporter)
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)

		var numDroppedSpans int
		if err != nil {
			numDroppedSpans = numExportedSpans[i]
		}

		EndTraceDataExportOp(
			ctx, span, numExportedSpans[i], numDroppedSpans, err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var sentSpans, failedToSendSpans int
	for i, span := range spans {
		assert.Equal(t, exporterPrefix+exporter+exportTraceDataOperationSuffix, span.Name)
		switch errs[i] {
		case nil:
			sentSpans += numExportedSpans[i]
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[SentSpansKey])
			assert.Equal(t, int64(0), span.Attributes[FailedToSendSpansKey])
			assert.Equal(t, okStatus, span.Status)
		case errFake:
			failedToSendSpans += numExportedSpans[i]
			assert.Equal(t, int64(0), span.Attributes[SentSpansKey])
			assert.Equal(t, int64(numExportedSpans[i]), span.Attributes[FailedToSendSpansKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	// Check legacy metrics.
	assert.NoError(t, observabilitytest.CheckValueViewExporterReceivedSpans(receiver, exporter, sentSpans+failedToSendSpans))
	assert.NoError(t, observabilitytest.CheckValueViewExporterDroppedSpans(receiver, exporter, failedToSendSpans))
	// Check new metrics.
	exporterTags := exporterViewTags(exporter)
	checkValueForSumView(t, mExporterSentSpans.Name(), exporterTags, sentSpans)
	checkValueForSumView(t, mExporterFailedToSendSpans.Name(), exporterTags, failedToSendSpans)
}

func Test_obsreport_ExportMetricsOp(t *testing.T) {
	doneFn, err := setupViews()
	defer doneFn()
	require.NoError(t, err)

	ss := &spanStore{}
	trace.RegisterExporter(ss)
	defer trace.UnregisterExporter(ss)

	parentCtx, parentSpan := trace.StartSpan(context.Background(),
		t.Name(), trace.WithSampler(trace.AlwaysSample()))
	defer parentSpan.End()

	// observabilitytest for exporters expects the context to flow the original
	// receiver tags, adding that to parent span.
	parentCtx = observability.ContextWithReceiverName(parentCtx, receiver)

	exporterCtx := ExporterContext(parentCtx, exporter)
	errs := []error{nil, errFake}
	toSendMetricPts := []int{17, 23}
	toSendTimeSeries := []int{3, 5}
	for i, err := range errs {
		ctx, span := StartMetricsExportOp(exporterCtx, exporter)
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)

		var numDroppedTimeSeires int
		if err != nil {
			numDroppedTimeSeires = toSendTimeSeries[i]
		}

		EndMetricsExportOp(
			ctx,
			span,
			toSendMetricPts[i],
			toSendTimeSeries[i],
			numDroppedTimeSeires,
			err)
	}

	spans := ss.PullAllSpans()
	require.Equal(t, len(errs), len(spans))

	var receivedTimeSeries, droppedTimeSeries int
	var sentPoints, failedToSendPoints int
	for i, span := range spans {
		assert.Equal(t, exporterPrefix+exporter+exportMetricsOperationSuffix, span.Name)
		receivedTimeSeries += toSendTimeSeries[i]
		switch errs[i] {
		case nil:
			sentPoints += toSendMetricPts[i]
			assert.Equal(t, int64(toSendMetricPts[i]), span.Attributes[SentMetricPointsKey])
			assert.Equal(t, int64(0), span.Attributes[FailedToSendMetricPointsKey])
			assert.Equal(t, okStatus, span.Status)
		case errFake:
			failedToSendPoints += toSendMetricPts[i]
			droppedTimeSeries += toSendTimeSeries[i]
			assert.Equal(t, int64(0), span.Attributes[SentMetricPointsKey])
			assert.Equal(t, int64(toSendMetricPts[i]), span.Attributes[FailedToSendMetricPointsKey])
			assert.Equal(t, errs[i].Error(), span.Status.Message)
		default:
			t.Fatalf("unexpected error: %v", errs[i])
		}
	}

	// Check legacy metrics.
	assert.NoError(t, observabilitytest.CheckValueViewExporterReceivedTimeSeries(receiver, exporter, receivedTimeSeries))
	assert.NoError(t, observabilitytest.CheckValueViewExporterDroppedTimeSeries(receiver, exporter, droppedTimeSeries))
	// Check new metrics.
	exporterTags := exporterViewTags(exporter)
	checkValueForSumView(t, mExporterSentMetricPoints.Name(), exporterTags, sentPoints)
	checkValueForSumView(t, mExporterFailedToSendMetricPoints.Name(), exporterTags, failedToSendPoints)
}

func setupViews() (doneFn func(), err error) {
	genLegacy := true
	genNew := true
	views := Configure(genLegacy, genNew)
	err = view.Register(views...)

	return func() {
		view.Unregister(views...)
	}, err
}

func checkValueForSumView(t *testing.T, vName string, wantTags []tag.Tag, value int) {
	// Make sure the tags slice is sorted by tag keys.
	sortTags(wantTags)

	rows, err := view.RetrieveData(vName)
	require.NoError(t, err, "error retrieving view data for view %s", vName)

	for _, row := range rows {
		// Make sure the tags slice is sorted by tag keys.
		sortTags(row.Tags)
		if reflect.DeepEqual(wantTags, row.Tags) {
			sum := row.Data.(*view.SumData)
			assert.Equal(t, float64(value), sum.Value)
			return
		}
	}

	assert.Fail(t, "row not found", "no row matches %v in rows %v", wantTags, rows)
}

func receiverViewTags(receiver, transport string) []tag.Tag {
	return []tag.Tag{
		{Key: tagKeyReceiver, Value: receiver},
		{Key: tagKeyTransport, Value: transport},
	}
}

func exporterViewTags(exporter string) []tag.Tag {
	return []tag.Tag{
		{Key: tagKeyExporter, Value: exporter},
	}
}

func sortTags(tags []tag.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key.Name() < tags[j].Key.Name()
	})
}

type spanStore struct {
	sync.Mutex
	spans []*trace.SpanData
}

func (ss *spanStore) ExportSpan(sd *trace.SpanData) {
	ss.Lock()
	ss.spans = append(ss.spans, sd)
	ss.Unlock()
}

func (ss *spanStore) PullAllSpans() []*trace.SpanData {
	ss.Lock()
	capturedSpans := ss.spans
	ss.spans = nil
	ss.Unlock()
	return capturedSpans
}
