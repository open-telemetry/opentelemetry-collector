// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracetranslator

// https://github.com/googleapis/googleapis/blob/bee79fbe03254a35db125dc6d2f1e9b752b390fe/google/rpc/code.proto#L33-L186
const (
	OCOK                 = 0
	OCCancelled          = 1
	OCUnknown            = 2
	OCInvalidArgument    = 3
	OCDeadlineExceeded   = 4
	OCNotFound           = 5
	OCAlreadyExists      = 6
	OCPermissionDenied   = 7
	OCResourceExhausted  = 8
	OCFailedPrecondition = 9
	OCAborted            = 10
	OCOutOfRange         = 11
	OCUnimplemented      = 12
	OCInternal           = 13
	OCUnavailable        = 14
	OCDataLoss           = 15
	OCUnauthenticated    = 16
)

var httpToOCCodeMap = map[int32]int32{
	400: OCInvalidArgument,
	401: OCUnauthenticated,
	403: OCPermissionDenied,
	404: OCNotFound,
	409: OCAborted,
	429: OCResourceExhausted,
	499: OCCancelled,
	500: OCInternal,
	501: OCUnimplemented,
	503: OCUnavailable,
	504: OCDeadlineExceeded,
}

// OCStatusCodeFromHTTP takes an HTTP status code and return the appropriate OC status code
func OCStatusCodeFromHTTP(code int32) int32 {
	if code >= 200 && code < 300 {
		return OCOK
	}
	if code, ok := httpToOCCodeMap[code]; ok {
		return code
	}
	return OCUnknown
}
