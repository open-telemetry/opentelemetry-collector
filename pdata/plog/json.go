// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog // import "go.opentelemetry.io/collector/pdata/plog"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pdata.Logs to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalLogs to the OTLP/JSON format.
func (*JSONMarshaler) MarshalLogs(ld Logs) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.LogsToProto(internal.Logs(ld))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

type JSONUnmarshaler struct{}

func (d *JSONUnmarshaler) UnmarshalLogs(buf []byte) (Logs, error) {
	td := otlplogs.LogsData{}
	if err := json.Unmarshal(bytes.NewReader(buf), &td); err != nil {
		return Logs{}, err
	}

	otlp.MigrateLogs(td.ResourceLogs)
	return Logs(internal.LogsFromProto(td)), nil
}
