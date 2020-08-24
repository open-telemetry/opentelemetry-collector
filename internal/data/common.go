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

package data

import commonproto "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"

func NewStringValue(s string) *commonproto.AnyValue {
	return &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{StringValue: s}}
}

func NewIntValue(i int64) *commonproto.AnyValue {
	return &commonproto.AnyValue{Value: &commonproto.AnyValue_IntValue{IntValue: i}}
}

func NewDoubleValue(d float64) *commonproto.AnyValue {
	return &commonproto.AnyValue{Value: &commonproto.AnyValue_DoubleValue{DoubleValue: d}}
}

func NewBoolValue(b bool) *commonproto.AnyValue {
	return &commonproto.AnyValue{Value: &commonproto.AnyValue_BoolValue{BoolValue: b}}
}
