// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

var _ Marshaler = (*JSONMarshaler)(nil)

// JSONMarshaler marshals Metrics to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalMetrics to the OTLP/JSON format.
func (*JSONMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	md.getOrig().MarshalJSON(dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to Metrics.
type JSONUnmarshaler struct {
	// DisallowUnknownFields causes UnmarshalMetrics to return an error when the
	// input contains JSON object fields that are not defined by the OTLP
	// schema. When false (the default), unknown fields are silently ignored.
	DisallowUnknownFields bool
	// prevent unkeyed literal initialization
	_ struct{}
}

// UnmarshalMetrics from OTLP/JSON format into Metrics.
func (u *JSONUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	iter.SetDisallowUnknownFields(u.DisallowUnknownFields)
	md := NewMetrics()
	md.getOrig().UnmarshalJSON(iter)
	if iter.Error() != nil {
		return Metrics{}, iter.Error()
	}
	otlp.MigrateMetrics(md.getOrig().ResourceMetrics)
	return md, nil
}
