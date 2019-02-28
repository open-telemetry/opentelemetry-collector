// Copyright 2019, OpenCensus Authors
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

import (
	"encoding/binary"
)

// Int64TraceIDToByteTraceID takes a long representaition of a trace id
// and converts it to a []byte representation.
func Int64TraceIDToByteTraceID(high, low int64) []byte {
	if high == 0 && low == 0 {
		return nil
	}
	traceID := make([]byte, 16)
	binary.BigEndian.PutUint64(traceID[:8], uint64(high))
	binary.BigEndian.PutUint64(traceID[8:], uint64(low))
	return traceID
}

// Int64SpanIDToByteSpanID takes a long representation of a span id and
// converts it to a []byte representation.
func Int64SpanIDToByteSpanID(id int64) []byte {
	if id == 0 {
		return nil
	}
	spanID := make([]byte, 8)
	binary.BigEndian.PutUint64(spanID, uint64(id))
	return spanID
}
