// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xreceiverhelper provides utilities for receivers of experimental signals.
package xreceiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper/xreceiverhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal"
	"go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"
)

// ObsReport is a helper to add observability to a profiles receiver.
type ObsReport struct {
	obsrep *receiverhelper.ObsReport

	otelAttrs        metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

// NewObsReport creates a new ObsReport.
func NewObsReport(cfg receiverhelper.ObsReportSettings) (*ObsReport, error) {
	obsrep, err := receiverhelper.NewObsReport(cfg)
	if err != nil {
		return nil, err
	}
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ReceiverCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		obsrep: obsrep,
		otelAttrs: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(internal.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(internal.TransportKey, cfg.Transport),
		)),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// StartProfilesOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *ObsReport) StartProfilesOp(operationCtx context.Context) context.Context {
	return rec.obsrep.StartProfilesOp(operationCtx)
}

// EndProfilesOp completes the receive operation that was started with
// StartProfilesOp.
func (rec *ObsReport) EndProfilesOp(
	receiverCtx context.Context,
	format string,
	numReceivedProfileSamples int,
	err error,
) {
	rec.obsrep.EndProfilesOp(receiverCtx, format, numReceivedProfileSamples, err)
}

// EndProfilesOpBySampleType completes the receive operation that was started with
// StartProfilesOp. In contrast to EndProfilesOp, the profile sample metrics are
// recorded per sample type, as a single payload can hold multiple profiles with
// different sample types. samplesByType maps the sample type, formatted as
// "type:unit", to the number of received samples of this type.
// Use ProfileSampleCountsByType to compute samplesByType before the profiles are
// passed to the next consumer, as the data may be mutated downstream.
func (rec *ObsReport) EndProfilesOpBySampleType(
	receiverCtx context.Context,
	format string,
	samplesByType map[string]int,
	err error,
) {
	numReceivedProfileSamples := 0
	for _, n := range samplesByType {
		numReceivedProfileSamples += n
	}
	numAccepted, numRefused, numFailedErrors := internal.SplitItems(numReceivedProfileSamples, err)

	for sampleType, n := range samplesByType {
		accepted, refused, failed := internal.SplitItems(n, err)
		sampleTypeAttr := metric.WithAttributes(attribute.String(internal.SampleTypeKey, sampleType))
		rec.telemetryBuilder.ReceiverAcceptedProfileSamples.Add(receiverCtx, int64(accepted), rec.otelAttrs, sampleTypeAttr)
		rec.telemetryBuilder.ReceiverRefusedProfileSamples.Add(receiverCtx, int64(refused), rec.otelAttrs, sampleTypeAttr)
		rec.telemetryBuilder.ReceiverFailedProfileSamples.Add(receiverCtx, int64(failed), rec.otelAttrs, sampleTypeAttr)
	}

	internal.RecordRequests(receiverCtx, rec.telemetryBuilder.ReceiverRequests, rec.otelAttrs, err)
	internal.EndSpan(trace.SpanFromContext(receiverCtx), format, xpipeline.SignalProfiles,
		numAccepted, numRefused, numFailedErrors, err)
}

// ProfileSampleCountsByType returns the number of samples for each sample type,
// formatted as "type:unit", of the profiles in pd.
//
// The sample types are taken from the payload and are currently unbounded.
// TODO: Cap the cardinality of the sample type once the Profiles specification
// defines the list of known sample types, which the Profiling SIG is working on.
// Sample types outside that list will then be reported as "unknown".
func ProfileSampleCountsByType(pd pprofile.Profiles) map[string]int {
	strTable := pd.Dictionary().StringTable()
	counts := make(map[string]int)

	rps := pd.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		sps := rps.At(i).ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			ps := sps.At(j).Profiles()
			for k := 0; k < ps.Len(); k++ {
				p := ps.At(k)
				st := p.SampleType()
				sampleType := lookupString(strTable, st.TypeStrindex()) + ":" +
					lookupString(strTable, st.UnitStrindex())
				counts[sampleType] += p.Samples().Len()
			}
		}
	}
	return counts
}

const (
	// unsetStrindex is used in place of a sample type or unit whose string
	// table index is 0, which indicates an unset value.
	unsetStrindex = "<unset>"
	// invalidStrindex is used in place of a sample type or unit whose string
	// table index is out of range.
	invalidStrindex = "<invalid>"
)

// lookupString returns the string at idx in strTable, unsetStrindex if idx is 0
// or invalidStrindex if idx is out of range.
func lookupString(strTable pcommon.StringSlice, idx int32) string {
	if idx == 0 {
		return unsetStrindex
	}
	if idx < 0 || int(idx) >= strTable.Len() {
		return invalidStrindex
	}
	return strTable.At(int(idx))
}
