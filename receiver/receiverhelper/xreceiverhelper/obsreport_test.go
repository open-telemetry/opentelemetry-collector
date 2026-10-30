// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xreceiverhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadatatest"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	receiverID = component.MustNewID("fakeReceiver")

	errFake = errors.New("errFake")
)

func TestReceiveProfilesOpBySampleType(t *testing.T) {
	originalState := metadata.ReceiverhelperNewReceiverMetricsFeatureGate.IsEnabled()
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), originalState))
	})

	samplesByType := map[string]int{
		"samples:count":       7,
		"off_cpu:nanoseconds": 2,
	}

	for _, tc := range []struct {
		name        string
		gateEnabled bool
		err         error
		wantMetric  string
	}{
		{name: "accepted", wantMetric: "accepted"},
		{name: "accepted_gate_enabled", gateEnabled: true, wantMetric: "accepted"},
		{name: "downstream_error", err: consumererror.NewDownstream(errFake), wantMetric: "refused"},
		{name: "downstream_error_gate_enabled", gateEnabled: true, err: consumererror.NewDownstream(errFake), wantMetric: "refused"},
		{name: "internal_error", err: errFake, wantMetric: "refused"},
		{name: "internal_error_gate_enabled", gateEnabled: true, err: errFake, wantMetric: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.ReceiverhelperNewReceiverMetricsFeatureGate.ID(), tc.gateEnabled))
			testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
				rec, err := NewObsReport(receiverhelper.ObsReportSettings{
					ReceiverID:             receiverID,
					Transport:              transport,
					ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
				})
				require.NoError(t, err)
				ctx := rec.StartProfilesOp(context.Background())
				require.NotNil(t, ctx)
				rec.EndProfilesOpBySampleType(ctx, format, samplesByType, tc.err)

				spans := tt.SpanRecorder.Ended()
				require.Len(t, spans, 1)
				var wantAccepted, wantRefused, wantFailed int64
				switch tc.wantMetric {
				case "accepted":
					wantAccepted = 9
				case "refused":
					wantRefused = 9
				case "failed":
					wantFailed = 9
				}
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.AcceptedProfileSamplesKey, Value: attribute.Int64Value(wantAccepted)})
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.RefusedProfileSamplesKey, Value: attribute.Int64Value(wantRefused)})
				require.Contains(t, spans[0].Attributes(), attribute.KeyValue{Key: internal.FailedProfileSamplesKey, Value: attribute.Int64Value(wantFailed)})

				dataPoints := func(metric string) []metricdata.DataPoint[int64] {
					var dps []metricdata.DataPoint[int64]
					for sampleType, n := range samplesByType {
						value := int64(0)
						if metric == tc.wantMetric {
							value = int64(n)
						}
						dps = append(dps, metricdata.DataPoint[int64]{
							Attributes: attribute.NewSet(
								attribute.String(internal.ReceiverKey, receiverID.String()),
								attribute.String(internal.TransportKey, transport),
								attribute.String(internal.SampleTypeKey, sampleType),
							),
							Value: value,
						})
					}
					return dps
				}
				metadatatest.AssertEqualReceiverAcceptedProfileSamples(t, tt, dataPoints("accepted"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverRefusedProfileSamples(t, tt, dataPoints("refused"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
				metadatatest.AssertEqualReceiverFailedProfileSamples(t, tt, dataPoints("failed"), metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			})
		})
	}
}

func TestReceiveProfilesOp(t *testing.T) {
	testTelemetry(t, func(t *testing.T, tt *componenttest.Telemetry) {
		rec, err := NewObsReport(receiverhelper.ObsReportSettings{
			ReceiverID:             receiverID,
			Transport:              transport,
			ReceiverCreateSettings: receiver.Settings{ID: receiverID, TelemetrySettings: tt.NewTelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		ctx := rec.StartProfilesOp(context.Background())
		require.NotNil(t, ctx)
		rec.EndProfilesOp(ctx, format, 7, nil)

		metadatatest.AssertEqualReceiverAcceptedProfileSamples(t, tt,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String(internal.ReceiverKey, receiverID.String()),
						attribute.String(internal.TransportKey, transport),
					),
					Value: int64(7),
				},
			}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	})
}

func TestProfileSampleCountsByType(t *testing.T) {
	pd := pprofile.NewProfiles()
	strTable := pd.Dictionary().StringTable()
	strTable.Append("", "samples", "count", "off_cpu", "nanoseconds", "events")

	ps := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	for _, p := range []struct {
		typeIdx, unitIdx int32
		samples          int
	}{
		{typeIdx: 1, unitIdx: 2, samples: 3},
		{typeIdx: 3, unitIdx: 4, samples: 5},
		{typeIdx: 1, unitIdx: 2, samples: 4},
		{typeIdx: 5, unitIdx: 2, samples: 0},
		// Out of range sample type indices.
		{typeIdx: 42, unitIdx: -1, samples: 1},
		// Out of range unit index only.
		{typeIdx: 1, unitIdx: 42, samples: 1},
		// Unset sample type.
		{typeIdx: 0, unitIdx: 0, samples: 2},
		// Unset unit only.
		{typeIdx: 1, unitIdx: 0, samples: 1},
	} {
		profile := ps.AppendEmpty()
		profile.SampleType().SetTypeStrindex(p.typeIdx)
		profile.SampleType().SetUnitStrindex(p.unitIdx)
		for range p.samples {
			profile.Samples().AppendEmpty()
		}
	}

	assert.Equal(t, map[string]int{
		"samples:count":       7,
		"off_cpu:nanoseconds": 5,
		"events:count":        0,
		"<unset>:<unset>":     2,
		"samples:<unset>":     1,
		"<invalid>:<invalid>": 1,
		"samples:<invalid>":   1,
	}, ProfileSampleCountsByType(pd))
	assert.Empty(t, ProfileSampleCountsByType(pprofile.NewProfiles()))
}

func testTelemetry(t *testing.T, testFunc func(t *testing.T, tt *componenttest.Telemetry)) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	testFunc(t, tt)
}
