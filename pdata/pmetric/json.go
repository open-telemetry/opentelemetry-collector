// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pmetric/internal/pmetricjson"
)

var delegate = pmetricjson.JSONMarshaler

var _ Marshaler = (*JSONMarshaler)(nil)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalMetrics(md Metrics) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.MetricsToProto(internal.Metrics(md))
	err := delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalMetrics(buf []byte) (Metrics, error) {
	var md otlpmetrics.MetricsData
	if err := pmetricjson.UnmarshalMetricsData(buf, &md); err != nil {
		return Metrics{}, err
	}
	return Metrics(internal.MetricsFromProto(md)), nil
}
