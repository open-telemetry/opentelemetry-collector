// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json // import "go.opentelemetry.io/collector/pdata/internal/json"

import (
	"fmt"
	"io"
	"math"
	"strconv"

	jsoniter "github.com/json-iterator/go"
)

// Stream avoids the need to explicitly call the `Stream.WriteMore` method while marshaling objects by
// checking if a field was previously written inside the current object and automatically appending a ","
// if so before writing the next field.
type Stream struct {
	*jsoniter.Stream
	// wmTracker acts like a stack which pushes a new value when an object is started and removes the
	// top when it is ended. The value added for every object tracks if there is any written field
	// already for that object, and if it is then automatically add a "," before any new field.
	wmTracker []bool
}

func BorrowStream(writer io.Writer) *Stream {
	return &Stream{
		Stream:    jsoniter.ConfigFastest.BorrowStream(writer),
		wmTracker: make([]bool, 32),
	}
}

func ReturnStream(s *Stream) {
	jsoniter.ConfigFastest.ReturnStream(s.Stream)
}

func (ots *Stream) WriteObjectStart() {
	ots.Stream.WriteObjectStart()
	ots.wmTracker = append(ots.wmTracker, false)
}

func (ots *Stream) WriteObjectField(field string) {
	if ots.wmTracker[len(ots.wmTracker)-1] {
		ots.WriteMore()
	}

	ots.Stream.WriteObjectField(field)
	ots.wmTracker[len(ots.wmTracker)-1] = true
}

func (ots *Stream) WriteObjectEnd() {
	ots.Stream.WriteObjectEnd()
	ots.wmTracker = ots.wmTracker[:len(ots.wmTracker)-1]
}

// WriteInt64 writes the values as a decimal string. This is per the protobuf encoding rules for int64, fixed64, uint64.
func (ots *Stream) WriteInt64(val int64) {
	ots.WriteString(strconv.FormatInt(val, 10))
}

// WriteUint64 writes the values as a decimal string. This is per the protobuf encoding rules for int64, fixed64, uint64.
func (ots *Stream) WriteUint64(val uint64) {
	ots.WriteString(strconv.FormatUint(val, 10))
}

// WriteFloat64 gracefully handles infinity & NaN values
func (ots *Stream) WriteFloat64(val float64) {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		ots.WriteString(fmt.Sprintf("%f", val))
		return
	}

	ots.Stream.WriteFloat64(val)
}
