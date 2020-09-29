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

package pdata

import (
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
)

// TraceID is an alias of OTLP TraceID data type.
type TraceID otlpcommon.TraceID

func NewTraceID(bytes []byte) TraceID {
	return TraceID(otlpcommon.NewTraceID(bytes))
}

func (t TraceID) Bytes() []byte {
	return otlpcommon.TraceID(t).Bytes()
}

func (t TraceID) HexString() string {
	return otlpcommon.TraceID(t).HexString()
}
