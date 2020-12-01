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
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/tinylib/msgp/msgp"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const tagAttributeKey = "fluent.tag"

// Most of this logic is derived directly from
// https://github.com/fluent/fluentd/wiki/Forward-Protocol-Specification-v1,
// which describes the fields in much greater detail.

type Event interface {
	DecodeMsg(dc *msgp.Reader) error
	LogRecords() pdata.LogSlice
	Chunk() string
	Compressed() string
}

type OptionsMap map[string]interface{}

// Chunk returns the `chunk` option or blank string if it was not set.
func (om OptionsMap) Chunk() string {
	c, _ := om["chunk"].(string)
	return c
}

func (om OptionsMap) Compressed() string {
	compressed, _ := om["compressed"].(string)
	return compressed
}

type EventMode int

type Peeker interface {
	Peek(n int) ([]byte, error)
}

const (
	UnknownMode EventMode = iota
	MessageMode
	ForwardMode
	PackedForwardMode
)

func (em EventMode) String() string {
	switch em {
	case UnknownMode:
		return "unknown"
	case MessageMode:
		return "message"
	case ForwardMode:
		return "forward"
	case PackedForwardMode:
		return "packedforward"
	default:
		panic("programmer bug")
	}
}

func insertToAttributeMap(key string, val interface{}, dest *pdata.AttributeMap) {
	// See https://github.com/tinylib/msgp/wiki/Type-Mapping-Rules
	switch r := val.(type) {
	case bool:
		dest.InsertBool(key, r)
	case string:
		dest.InsertString(key, r)
	case uint64:
		dest.InsertInt(key, int64(r))
	case int64:
		dest.InsertInt(key, r)
	case []byte:
		dest.InsertString(key, string(r))
	case map[string]interface{}, []interface{}:
		encoded, err := json.Marshal(r)
		if err != nil {
			dest.InsertString(key, err.Error())
		}
		dest.InsertString(key, string(encoded))
	case float32:
		dest.InsertDouble(key, float64(r))
	case float64:
		dest.InsertDouble(key, r)
	default:
		dest.InsertString(key, fmt.Sprintf("%v", r))
	}
}

func timeFromTimestamp(ts interface{}) (time.Time, error) {
	switch v := ts.(type) {
	case int64:
		return time.Unix(v, 0), nil
	case *EventTimeExt:
		return time.Time(*v), nil
	default:
		return time.Time{}, fmt.Errorf("unknown type of value: %v", ts)
	}
}

func decodeTimestampToLogRecord(dc *msgp.Reader, lr pdata.LogRecord) error {
	tsIntf, err := dc.ReadIntf()
	if err != nil {
		return msgp.WrapError(err, "Time")
	}

	ts, err := timeFromTimestamp(tsIntf)
	if err != nil {
		return msgp.WrapError(err, "Time")
	}

	lr.SetTimestamp(pdata.TimestampUnixNano(ts.UnixNano()))
	return nil
}

func parseRecordToLogRecord(dc *msgp.Reader, lr pdata.LogRecord) error {
	attrs := lr.Attributes()

	recordLen, err := dc.ReadMapHeader()
	if err != nil {
		return msgp.WrapError(err, "Record")
	}

	for recordLen > 0 {
		recordLen--
		key, err := dc.ReadString()
		if err != nil {
			return msgp.WrapError(err, "Record")
		}
		val, err := dc.ReadIntf()
		if err != nil {
			return msgp.WrapError(err, "Record", key)
		}

		if s, ok := val.(string); ok && key == "log" {
			lr.Body().SetStringVal(s)
		} else {
			insertToAttributeMap(key, val, &attrs)
		}
	}

	return nil
}

type MessageEventLogRecord struct {
	pdata.LogSlice
	OptionsMap
}

func (melr *MessageEventLogRecord) LogRecords() pdata.LogSlice {
	return melr.LogSlice
}

func (melr *MessageEventLogRecord) DecodeMsg(dc *msgp.Reader) error {
	melr.LogSlice = pdata.NewLogSlice()
	melr.LogSlice.Resize(1)

	var arrLen uint32
	var err error

	arrLen, err = dc.ReadArrayHeader()
	if err != nil {
		return msgp.WrapError(err)
	}
	if arrLen > 4 || arrLen < 3 {
		return msgp.ArrayError{Wanted: 3, Got: arrLen}
	}

	tag, err := dc.ReadString()
	if err != nil {
		return msgp.WrapError(err, "Tag")
	}

	attrs := melr.LogSlice.At(0).Attributes()
	attrs.InsertString(tagAttributeKey, tag)

	err = decodeTimestampToLogRecord(dc, melr.LogSlice.At(0))
	if err != nil {
		return msgp.WrapError(err, "Time")
	}

	err = parseRecordToLogRecord(dc, melr.LogSlice.At(0))
	if err != nil {
		return err
	}

	if arrLen == 4 {
		melr.OptionsMap, err = parseOptions(dc)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseOptions(dc *msgp.Reader) (OptionsMap, error) {
	var optionLen uint32
	optionLen, err := dc.ReadMapHeader()
	if err != nil {
		return nil, msgp.WrapError(err, "Option")
	}
	out := make(OptionsMap, optionLen)

	for optionLen > 0 {
		optionLen--
		key, err := dc.ReadString()
		if err != nil {
			return nil, msgp.WrapError(err, "Option")
		}
		val, err := dc.ReadIntf()
		if err != nil {
			return nil, msgp.WrapError(err, "Option", key)
		}
		out[key] = val
	}
	return out, nil
}

type ForwardEventLogRecords struct {
	pdata.LogSlice
	OptionsMap
}

func (fe *ForwardEventLogRecords) LogRecords() pdata.LogSlice {
	return fe.LogSlice
}

func (fe *ForwardEventLogRecords) DecodeMsg(dc *msgp.Reader) (err error) {
	fe.LogSlice = pdata.NewLogSlice()

	var arrLen uint32
	arrLen, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if arrLen < 2 || arrLen > 3 {
		err = msgp.ArrayError{Wanted: 2, Got: arrLen}
		return
	}

	tag, err := dc.ReadString()
	if err != nil {
		return msgp.WrapError(err, "Tag")
	}

	entryLen, err := dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "Record")
		return
	}

	fe.LogSlice.Resize(int(entryLen))
	for i := 0; i < int(entryLen); i++ {
		lr := fe.LogSlice.At(i)

		err = parseEntryToLogRecord(dc, lr)
		if err != nil {
			return msgp.WrapError(err, "Entries", i)
		}
		fe.LogSlice.At(i).Attributes().InsertString(tagAttributeKey, tag)
	}

	if arrLen == 3 {
		fe.OptionsMap, err = parseOptions(dc)
		if err != nil {
			return err
		}
	}

	return
}

func parseEntryToLogRecord(dc *msgp.Reader, lr pdata.LogRecord) error {
	arrLen, err := dc.ReadArrayHeader()
	if err != nil {
		return msgp.WrapError(err)
	}
	if arrLen != 2 {
		return msgp.ArrayError{Wanted: 2, Got: arrLen}
	}

	err = decodeTimestampToLogRecord(dc, lr)
	if err != nil {
		return msgp.WrapError(err, "Time")
	}

	return parseRecordToLogRecord(dc, lr)
}

type PackedForwardEventLogRecords struct {
	pdata.LogSlice
	OptionsMap
}

func (pfe *PackedForwardEventLogRecords) LogRecords() pdata.LogSlice {
	return pfe.LogSlice
}

// DecodeMsg implements msgp.Decodable.  This was originally code generated but
// then manually copied here in order to handle the optional Options field.
func (pfe *PackedForwardEventLogRecords) DecodeMsg(dc *msgp.Reader) error {
	pfe.LogSlice = pdata.NewLogSlice()

	arrLen, err := dc.ReadArrayHeader()
	if err != nil {
		return msgp.WrapError(err)
	}
	if arrLen < 2 || arrLen > 3 {
		return msgp.ArrayError{Wanted: 2, Got: arrLen}
	}

	tag, err := dc.ReadString()
	if err != nil {
		return msgp.WrapError(err, "Tag")
	}

	entriesFirstByte, err := dc.R.Peek(1)
	if err != nil {
		return msgp.WrapError(err, "EntriesRaw")
	}

	entriesType := msgp.NextType(entriesFirstByte)
	// We have to read out the entries raw all the way first because we don't
	// know whether it is compressed or not until we read the options map which
	// comes after.  I guess we could use some kind of detection logic to
	// determine if it is gzipped by peeking and just ignoring options, but
	// this seems simpler for now.
	var entriesRaw []byte
	switch entriesType {
	case msgp.StrType:
		var entriesStr string
		entriesStr, err = dc.ReadString()
		if err != nil {
			return msgp.WrapError(err, "EntriesRaw")
		}
		entriesRaw = []byte(entriesStr)
	case msgp.BinType:
		entriesRaw, err = dc.ReadBytes(nil)
		if err != nil {
			return msgp.WrapError(err, "EntriesRaw")
		}
	default:
		return msgp.WrapError(fmt.Errorf("invalid type %d", entriesType), "EntriesRaw")
	}

	if arrLen == 3 {
		pfe.OptionsMap, err = parseOptions(dc)
		if err != nil {
			return err
		}
	}

	err = pfe.parseEntries(entriesRaw, pfe.Compressed() == "gzip", tag)
	if err != nil {
		return err
	}

	return nil
}

func (pfe *PackedForwardEventLogRecords) parseEntries(entriesRaw []byte, isGzipped bool, tag string) error {
	var reader io.Reader
	reader = bytes.NewReader(entriesRaw)

	if isGzipped {
		var err error
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer reader.(*gzip.Reader).Close()
	}

	msgpReader := msgp.NewReader(reader)
	for {
		lr := pdata.NewLogRecord()
		err := parseEntryToLogRecord(msgpReader, lr)
		if err != nil {
			if msgp.Cause(err) == io.EOF {
				return nil
			}
			return err
		}

		lr.Attributes().InsertString(tagAttributeKey, tag)

		pfe.LogSlice.Append(lr)
	}
}
