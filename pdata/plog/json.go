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
