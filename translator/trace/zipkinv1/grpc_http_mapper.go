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

package zipkinv1

// https://github.com/googleapis/googleapis/blob/bee79fbe03254a35db125dc6d2f1e9b752b390fe/google/rpc/code.proto#L33-L186
const (
	ocOK               = 0
	ocCancelled        = 1
	ocUnknown          = 2
	ocInvalidArgument  = 3
	ocDeadlineExceeded = 4
	ocNotFound         = 5
	// ocAlreadyExists     = 6
	ocPermissionDenied  = 7
	ocResourceExhausted = 8
	// ocFailedPrecondition = 9
	// ocAborted            = 10
	// ocOutOfRange      = 11
	ocUnimplemented = 12
	ocInternal      = 13
	ocUnavailable   = 14
	// ocDataLoss        = 15
	ocUnauthenticated = 16
)

var httpToOCCodeMap = map[int32]int32{
	401: ocUnauthenticated,
	403: ocPermissionDenied,
	404: ocNotFound,
	429: ocResourceExhausted,
	499: ocCancelled,
	501: ocUnimplemented,
	503: ocUnavailable,
	504: ocDeadlineExceeded,
}

// ocStatusCodeFromHTTP takes an HTTP status code and return the appropriate OpenTelemetry status code
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/data-http.md
func ocStatusCodeFromHTTP(code int32) int32 {
	if code >= 100 && code < 400 {
		return ocOK
	}
	if c, ok := httpToOCCodeMap[code]; ok {
		return c
	}
	if code >= 400 && code < 500 {
		return ocInvalidArgument
	}
	if code >= 500 && code < 600 {
		return ocInternal
	}
	return ocUnknown
}
