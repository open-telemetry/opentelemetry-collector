// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// NetworkReporter is a helper to add network-level observability to
// an exporter or receiver.
type NetworkReporter struct {
	attrs         []attribute.KeyValue
	sentBytes     syncint64.Counter
	sentWireBytes syncint64.Counter
	recvBytes     syncint64.Counter
	recvWireBytes syncint64.Counter
}

// SizesStruct is used to pass uncompressed on-wire message lengths to
// the CountSend() and CountReceive() methods.
type SizesStruct struct {
	Length     int64
	WireLength int64
}

// makeSentMetrics builds the sent and sent-wire metric instruments
// for an exporter or receiver using the corresponding `prefix`.
func makeSentMetrics(prefix string, meter metric.Meter) (sent, sentWire syncint64.Counter, _ error) {
	sentBytes, err1 := meter.SyncInt64().Counter(
		prefix+obsmetrics.SentBytes,
		instrument.WithDescription("Number of bytes sent by the component."),
		instrument.WithUnit(unit.Bytes))

	sentWireBytes, err2 := meter.SyncInt64().Counter(
		prefix+obsmetrics.SentWireBytes,
		instrument.WithDescription("Number of bytes sent on the wire by the component."),
		instrument.WithUnit(unit.Bytes))
	return sentBytes, sentWireBytes, multierr.Append(err1, err2)
}

// makeRecvMetrics builds the  and received-wire metric instruments
// for an exporter or receiver using the corresponding `prefix`.
func makeRecvMetrics(prefix string, meter metric.Meter) (recv, recvWire syncint64.Counter, _ error) {
	recvBytes, err1 := meter.SyncInt64().Counter(
		prefix+obsmetrics.RecvBytes,
		instrument.WithDescription("Number of bytes received by the component."),
		instrument.WithUnit(unit.Bytes))

	recvWireBytes, err2 := meter.SyncInt64().Counter(
		prefix+obsmetrics.RecvWireBytes,
		instrument.WithDescription("Number of bytes received on the wire by the component."),
		instrument.WithUnit(unit.Bytes))
	return recvBytes, recvWireBytes, multierr.Append(err1, err2)
}

// NewExporterNetworkReporter creates a new NetworkReporter configured for an exporter.
func NewExporterNetworkReporter(settings component.ExporterCreateSettings) (*NetworkReporter, error) {
	level := settings.TelemetrySettings.MetricsLevel

	if level <= configtelemetry.LevelBasic {
		// Note: NetworkReporter implements nil a check.
		return nil, nil
	}

	meter := settings.TelemetrySettings.MeterProvider.Meter(exporterScope)
	rep := &NetworkReporter{
		attrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ExporterKey, settings.ID.String()),
		},
	}

	var errors, err error
	rep.sentBytes, rep.sentWireBytes, err = makeSentMetrics(obsmetrics.ExporterPrefix, meter)
	errors = multierr.Append(errors, err)

	// Normally, an exporter counts sent bytes, and skips received
	// bytes.  LevelDetailed will reveal exporter-received bytes.
	if level > configtelemetry.LevelNormal {
		rep.recvBytes, rep.recvWireBytes, err = makeRecvMetrics(obsmetrics.ExporterPrefix, meter)
		errors = multierr.Append(errors, err)
	}

	return rep, errors
}

// NewReceiverNetworkReporter creates a new NetworkReporter configured for an exporter.
func NewReceiverNetworkReporter(settings component.ReceiverCreateSettings) (*NetworkReporter, error) {
	level := settings.TelemetrySettings.MetricsLevel

	if level <= configtelemetry.LevelBasic {
		// Note: NetworkReporter implements nil a check.
		return nil, nil
	}

	meter := settings.MeterProvider.Meter(receiverScope)
	rep := &NetworkReporter{
		attrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ReceiverKey, settings.ID.String()),
		},
	}

	var errors, err error
	rep.recvBytes, rep.recvWireBytes, err = makeRecvMetrics(obsmetrics.ReceiverPrefix, meter)
	errors = multierr.Append(errors, err)

	// Normally, a receiver counts received bytes, and skips sent
	// bytes.  LevelDetailed will reveal receiver-sent bytes.
	if level > configtelemetry.LevelNormal {
		rep.sentBytes, rep.sentWireBytes, err = makeSentMetrics(obsmetrics.ReceiverPrefix, meter)
		errors = multierr.Append(errors, err)
	}

	return rep, errors
}

// CountSend is used to report a message sent by the component.  For
// exporters, SizesStruct indicates the size of a request.  For
// receivers, SizesStruct indicates the size of a response.
func (rep *NetworkReporter) CountSend(ctx context.Context, ss SizesStruct) {
	// Indicates basic level telemetry, not counting bytes.
	if rep == nil {
		return
	}

	if rep.sentBytes != nil && ss.Length > 0 {
		rep.sentBytes.Add(ctx, ss.Length, rep.attrs...)
	}
	if rep.sentWireBytes != nil && ss.WireLength > 0 {
		rep.sentWireBytes.Add(ctx, ss.WireLength, rep.attrs...)
	}
}

// CountReceive is used to report a message received by the component.  For
// exporters, SizesStruct indicates the size of a response.  For
// receivers, SizesStruct indicates the size of a request.
func (rep *NetworkReporter) CountReceive(ctx context.Context, ss SizesStruct) {
	// Indicates basic level telemetry, not counting bytes.
	if rep == nil {
		return
	}

	if rep.recvBytes != nil && ss.Length > 0 {
		rep.recvBytes.Add(ctx, ss.Length, rep.attrs...)
	}
	if rep.recvWireBytes != nil && ss.WireLength > 0 {
		rep.recvWireBytes.Add(ctx, ss.WireLength, rep.attrs...)
	}
}
