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

package attributesprocessor

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"math"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

const (
	int64ByteSize   = 8
	float64ByteSize = 8
)

var (
	byteTrue  = [1]byte{1}
	byteFalse = [1]byte{0}
)

// SHA1AttributeHahser hashes an AttributeValue using SHA1 and returns a
// hashed version of the attribute. In practice, this would mostly be used
// for string attributes but we support all types for completeness/correctness
// and eliminate any surprises.
func SHA1AttributeHahser(attr *tracepb.AttributeValue) *tracepb.AttributeValue {
	var val []byte
	switch attr.Value.(type) {
	case *tracepb.AttributeValue_StringValue:
		val = []byte(attr.GetStringValue().Value)
	case *tracepb.AttributeValue_BoolValue:
		if attr.GetBoolValue() {
			val = byteTrue[:]
		} else {
			val = byteFalse[:]
		}
	case *tracepb.AttributeValue_IntValue:
		val = make([]byte, int64ByteSize)
		binary.LittleEndian.PutUint64(val, uint64(attr.GetIntValue()))
	case *tracepb.AttributeValue_DoubleValue:
		val = make([]byte, float64ByteSize)
		binary.LittleEndian.PutUint64(val, math.Float64bits(attr.GetDoubleValue()))
	}

	var hashed string
	if len(val) > 0 {
		h := sha1.New()
		h.Write(val)
		val = h.Sum(nil)
		hashedBytes := make([]byte, hex.EncodedLen(len(val)))
		hex.Encode(hashedBytes, val)
		hashed = string(hashedBytes)
	}

	return &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_StringValue{
			StringValue: &tracepb.TruncatableString{Value: string(hashed)},
		},
	}
}
