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

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/ptrace/internal/ptracejson"
)

var delegate = ptracejson.JSONMarshaler

var _ Marshaler = (*JSONMarshaler)(nil)

type JSONMarshaler struct{}

func (*JSONMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.TracesToProto(internal.Traces(td))
	err := delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

type JSONUnmarshaler struct{}

func (*JSONUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	var td otlptrace.TracesData
	if err := ptracejson.UnmarshalTraceData(buf, &td); err != nil {
		return Traces{}, err
	}
	return Traces(internal.TracesFromProto(td)), nil
}
