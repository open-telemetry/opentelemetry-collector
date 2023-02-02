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
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

// StatusCode mirrors the codes defined at
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#set-status
type StatusCode int32

const (
	StatusCodeUnset = StatusCode(otlptrace.Status_STATUS_CODE_UNSET)
	StatusCodeOk    = StatusCode(otlptrace.Status_STATUS_CODE_OK)
	StatusCodeError = StatusCode(otlptrace.Status_STATUS_CODE_ERROR)
)

// String returns the string representation of the StatusCode.
func (sc StatusCode) String() string {
	switch sc {
	case StatusCodeUnset:
		return "Unset"
	case StatusCodeOk:
		return "Ok"
	case StatusCodeError:
		return "Error"
	}
	return ""
}
