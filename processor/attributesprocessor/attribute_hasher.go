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
	// #nosec
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"math"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

const (
	int64ByteSize   = 8
	float64ByteSize = 8
)

var (
	byteTrue  = [1]byte{1}
	byteFalse = [1]byte{0}
)

// SHA1AttributeHasher hashes an AttributeValue using SHA1 and returns a
// hashed version of the attribute. In practice, this would mostly be used
// for string attributes but we support all types for completeness/correctness
// and eliminate any surprises.
func SHA1AttributeHasher(attr data.AttributeValue) {
	var val []byte
	switch attr.Type() {
	case data.AttributeValueSTRING:
		val = []byte(attr.StringVal())
	case data.AttributeValueBOOL:
		if attr.BoolVal() {
			val = byteTrue[:]
		} else {
			val = byteFalse[:]
		}
	case data.AttributeValueINT:
		val = make([]byte, int64ByteSize)
		binary.LittleEndian.PutUint64(val, uint64(attr.IntVal()))
	case data.AttributeValueDOUBLE:
		val = make([]byte, float64ByteSize)
		binary.LittleEndian.PutUint64(val, math.Float64bits(attr.DoubleVal()))
	}

	var hashed string
	if len(val) > 0 {
		// #nosec
		h := sha1.New()
		h.Write(val)
		val = h.Sum(nil)
		hashedBytes := make([]byte, hex.EncodedLen(len(val)))
		hex.Encode(hashedBytes, val)
		hashed = string(hashedBytes)
	}

	attr.SetStringVal(hashed)
}
