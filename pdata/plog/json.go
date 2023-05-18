// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/plog/internal/plogjson"
)

var delegate = plogjson.JSONMarshaler

var _ Marshaler = (*JSONMarshaler)(nil)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.LogsToProto(internal.Logs(ld))
	err := delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	var ld otlplogs.LogsData
	if err := plogjson.UnmarshalLogsData(buf, &ld); err != nil {
		return Logs{}, err
	}
	return Logs(internal.LogsFromProto(ld)), nil
}
