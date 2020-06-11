// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fluentforwardreceiver

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/fluentforwardreceiver/testdata"
	"go.opentelemetry.io/collector/testutil/logtest"
)

func TestMessageEventConversion(t *testing.T) {
	eventBytes := testdata.ParseHexDump("message-event")
	reader := msgp.NewReader(bytes.NewReader(eventBytes))

	var event MessageEventLogRecord
	err := event.DecodeMsg(reader)
	require.Nil(t, err)

	le := event.LogRecords().At(0)
	le.Attributes().Sort()

	expected := logtest.Logs(
		logtest.Log{
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

	b := []byte{}

	b = msgp.AppendArrayHeader(b, 3)
	b = msgp.AppendString(b, "my-tag")
	b = msgp.AppendInt(b, 5000)
	b = msgp.AppendMapHeader(b, 14)
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

	reader := msgp.NewReader(bytes.NewReader(b))

	var event MessageEventLogRecord
	err := event.DecodeMsg(reader)
	require.Nil(t, err)

	le := event.LogRecords().At(0)
	le.Attributes().Sort()
	require.EqualValues(t, logtest.Logs(
		logtest.Log{
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
			},
		},
	).ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0), le)
}
