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

package fluentforwardreceiver

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testutil/logstest"
)

func TestMessageEventConversion(t *testing.T) {
	eventBytes := parseHexDump("testdata/message-event")
	reader := msgp.NewReader(bytes.NewReader(eventBytes))

	var event MessageEventLogRecord
	err := event.DecodeMsg(reader)
	require.Nil(t, err)

	le := event.LogRecords().At(0)
	le.Attributes().Sort()

	expected := logstest.Logs(
		logstest.Log{
			Timestamp: 1593031012000000000,
			Body:      pdata.NewAttributeValueString("..."),
			Attributes: map[string]pdata.AttributeValue{
				"container_id":   pdata.NewAttributeValueString("b00a67eb645849d6ab38ff8beb4aad035cc7e917bf123c3e9057c7e89fc73d2d"),
				"container_name": pdata.NewAttributeValueString("/unruffled_cannon"),
				"fluent.tag":     pdata.NewAttributeValueString("b00a67eb6458"),
				"source":         pdata.NewAttributeValueString("stdout"),
			},
		},
	)
	require.EqualValues(t, expected.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0), le)
}

func TestAttributeTypeConversion(t *testing.T) {
	var b []byte

	b = msgp.AppendArrayHeader(b, 3)
	b = msgp.AppendString(b, "my-tag")
	b = msgp.AppendInt(b, 5000)
	b = msgp.AppendMapHeader(b, 15)
	b = msgp.AppendString(b, "a")
	b = msgp.AppendFloat64(b, 5.0)
	b = msgp.AppendString(b, "b")
	b = msgp.AppendFloat32(b, 6.0)
	b = msgp.AppendString(b, "c")
	b = msgp.AppendBool(b, true)
	b = msgp.AppendString(b, "d")
	b = msgp.AppendInt8(b, 1)
	b = msgp.AppendString(b, "e")
	b = msgp.AppendInt16(b, 2)
	b = msgp.AppendString(b, "f")
	b = msgp.AppendInt32(b, 3)
	b = msgp.AppendString(b, "g")
	b = msgp.AppendInt64(b, 4)
	b = msgp.AppendString(b, "h")
	b = msgp.AppendUint8(b, ^uint8(0))
	b = msgp.AppendString(b, "i")
	b = msgp.AppendUint16(b, ^uint16(0))
	b = msgp.AppendString(b, "j")
	b = msgp.AppendUint32(b, ^uint32(0))
	b = msgp.AppendString(b, "k")
	b = msgp.AppendUint64(b, ^uint64(0))
	b = msgp.AppendString(b, "l")
	b = msgp.AppendComplex64(b, complex64(0))
	b = msgp.AppendString(b, "m")
	b = msgp.AppendBytes(b, []byte{0x1, 0x65, 0x2})
	b = msgp.AppendString(b, "n")
	b = msgp.AppendArrayHeader(b, 2)
	b = msgp.AppendString(b, "first")
	b = msgp.AppendString(b, "second")
	b = msgp.AppendString(b, "o")
	b, err := msgp.AppendIntf(b, []uint8{99, 100, 101})

	require.NoError(t, err)

	reader := msgp.NewReader(bytes.NewReader(b))

	var event MessageEventLogRecord
	err = event.DecodeMsg(reader)
	require.Nil(t, err)

	le := event.LogRecords().At(0)
	le.Attributes().Sort()
	require.EqualValues(t, logstest.Logs(
		logstest.Log{
			Timestamp: 5000000000000,
			Body:      pdata.NewAttributeValueNull(),
			Attributes: map[string]pdata.AttributeValue{
				"a":          pdata.NewAttributeValueDouble(5.0),
				"b":          pdata.NewAttributeValueDouble(6.0),
				"c":          pdata.NewAttributeValueBool(true),
				"d":          pdata.NewAttributeValueInt(1),
				"e":          pdata.NewAttributeValueInt(2),
				"f":          pdata.NewAttributeValueInt(3),
				"fluent.tag": pdata.NewAttributeValueString("my-tag"),
				"g":          pdata.NewAttributeValueInt(4),
				"h":          pdata.NewAttributeValueInt(255),
				"i":          pdata.NewAttributeValueInt(65535),
				"j":          pdata.NewAttributeValueInt(4294967295),
				"k":          pdata.NewAttributeValueInt(-1),
				"l":          pdata.NewAttributeValueString("(0+0i)"),
				"m":          pdata.NewAttributeValueString("\001e\002"),
				"n":          pdata.NewAttributeValueString(`["first","second"]`),
				"o":          pdata.NewAttributeValueString("cde"),
			},
		},
	).ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0), le)
}

func TestEventMode(t *testing.T) {
	require.Equal(t, "unknown", UnknownMode.String())
	require.Equal(t, "message", MessageMode.String())
	require.Equal(t, "forward", ForwardMode.String())
	require.Equal(t, "packedforward", PackedForwardMode.String())

	const TestMode EventMode = 6
	require.Panics(t, func() { _ = TestMode.String() })
}

func TestTimeFromTimestampBadType(t *testing.T) {
	_, err := timeFromTimestamp("bad")
	require.NotNil(t, err)
}

func TestMessageEventConversionWithErrors(t *testing.T) {
	var b []byte

	b = msgp.AppendArrayHeader(b, 3)
	b = msgp.AppendString(b, "my-tag")
	b = msgp.AppendInt(b, 5000)
	b = msgp.AppendMapHeader(b, 1)
	b = msgp.AppendString(b, "a")
	b = msgp.AppendFloat64(b, 5.0)

	for i := 0; i < len(b)-1; i++ {
		t.Run(fmt.Sprintf("EOF at byte %d", i), func(t *testing.T) {
			reader := msgp.NewReader(bytes.NewReader(b[:i]))

			var event MessageEventLogRecord
			err := event.DecodeMsg(reader)
			require.NotNil(t, err)
		})
	}

	t.Run("Invalid timestamp type uint", func(t *testing.T) {
		in := make([]byte, len(b))
		copy(in, b)
		in[8] = 0xcd
		reader := msgp.NewReader(bytes.NewReader(in))

		var event MessageEventLogRecord
		err := event.DecodeMsg(reader)
		require.NotNil(t, err)
	})
}

func TestForwardEventConversionWithErrors(t *testing.T) {
	b := parseHexDump("testdata/forward-event")

	for i := 0; i < len(b)-1; i++ {
		t.Run(fmt.Sprintf("EOF at byte %d", i), func(t *testing.T) {
			reader := msgp.NewReader(bytes.NewReader(b[:i]))

			var event ForwardEventLogRecords
			err := event.DecodeMsg(reader)
			require.NotNil(t, err)
		})
	}
}

func TestPackedForwardEventConversionWithErrors(t *testing.T) {
	b := parseHexDump("testdata/forward-packed-compressed")

	for i := 0; i < len(b)-1; i++ {
		t.Run(fmt.Sprintf("EOF at byte %d", i), func(t *testing.T) {
			reader := msgp.NewReader(bytes.NewReader(b[:i]))

			var event PackedForwardEventLogRecords
			err := event.DecodeMsg(reader)
			require.NotNil(t, err)
		})
	}

	t.Run("Invalid gzip header", func(t *testing.T) {
		in := make([]byte, len(b))
		copy(in, b)
		in[0x71] = 0xff
		reader := msgp.NewReader(bytes.NewReader(in))

		var event PackedForwardEventLogRecords
		err := event.DecodeMsg(reader)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "gzip")
		print(err.Error())
	})
}
