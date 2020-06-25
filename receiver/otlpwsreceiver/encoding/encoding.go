// Copyright The OpenTelemetry Authors
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

package encoding

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"log"

	gogoproto "github.com/gogo/protobuf/proto"
	"google.golang.org/genproto/googleapis/rpc/status"

	collectormetric "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

type MessageDirection int

const (
	MessageDirectionRequest  MessageDirection = 0
	MessageDirectionResponse MessageDirection = 1
)

type BodyType int

const (
	_ BodyType = iota
	BodyTypeStatus
	BodyTypeExportTraceRequest
	BodyTypeExportTraceResponse
	BodyTypeExportMetricRequest
	BodyTypeExportMetricResponse
)

const (
	HeaderFlagsMessageDirectionMask   = 0b000000000001
	HeaderFlagsCompressionMethodMask  = 0b000000001110
	HeaderFlagsCompressionMethodShift = 1
	HeaderFlagsBodyTypeMask           = 0b111111110000
	HeaderFlagsBodyTypeShift          = 4
)

type CompressionMethod int32

const (
	CompressionMethodNONE CompressionMethod = 0
	CompressionMethodZLIB CompressionMethod = 1
	CompressionMethodLZ4  CompressionMethod = 2
)

type Message struct {
	ID        uint64
	Body      gogoproto.Message
	Direction MessageDirection
}

type Encoder struct {
	buf *gogoproto.Buffer
}

func NewEncoder() Encoder {
	return Encoder{buf: gogoproto.NewBuffer(nil)}
}

// Encode request body into a message of continuous bytes. The message starts with uint64
// Id, followed by uint64 Flags, followed by Body encoded in Protobuf format.
// +-------------+-----------+----------------------+
// + Message Id  | Flags     | Variable length Body |
// + Varuint64   | Varuint64 | (Protobuf-encoded)   |
// +-------------+-----------+----------------------+
func (e *Encoder) Encode(
	message *Message,
	compression CompressionMethod,
) ([]byte, error) {
	//e.buf = gogoproto.NewBuffer(nil)
	e.buf.Reset()
	//bodyBytes, err := gogoproto.Marshal(message.Body)
	//if err != nil {
	//	return nil, err
	//}

	// Compose flags.

	// Set direction.
	flags := uint64(message.Direction)

	// Set compression method.
	switch compression {
	case CompressionMethodNONE:
		flags |= uint64(CompressionMethodNONE) << HeaderFlagsCompressionMethodShift
		//case CompressionMethodZLIB:
		//	flags |= uint64(CompressionMethodZLIB) << HeaderFlagsCompressionMethodShift
		//	var b bytes.Buffer
		//	w := zlib.NewWriter(&b)
		//	bodyBytes, err := gogoproto.Marshal(message.Body)
		//	if err != nil {
		//		return nil, err
		//	}
		//	w.Write(bodyBytes)
		//	w.Close()
		//	bodyBytes = b.Bytes()
	}

	// Set body type.
	flags |= uint64(getBodyType(message.Body)) << HeaderFlagsBodyTypeShift

	// Write message id.
	err := e.buf.EncodeVarint(message.ID)
	if err != nil {
		return nil, err
	}

	// Write flags.
	err = e.buf.EncodeVarint(flags)
	if err != nil {
		return nil, err
	}

	// Write body.
	err = e.buf.Marshal(message.Body)
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func getBodyType(body gogoproto.Message) BodyType {
	switch (body).(type) {
	case *status.Status:
		return BodyTypeStatus
	case *collectortrace.ExportTraceServiceRequest:
		return BodyTypeExportTraceRequest
	case *collectortrace.ExportTraceServiceResponse:
		return BodyTypeExportTraceResponse
	case *collectormetric.ExportMetricsServiceRequest:
		return BodyTypeExportMetricRequest
	case *collectormetric.ExportMetricsServiceResponse:
		return BodyTypeExportMetricResponse
	}
	panic("Unknown body type")
}

// Decode a continuous message of bytes into a Body. This function perform the
// reverse of Encode operation.
func Decode(messageBytes []byte, message *Message) error {

	readBuf := bytes.NewBuffer(messageBytes)
	msgID, err := binary.ReadUvarint(readBuf)
	if err != nil {
		return err
	}

	message.ID = msgID

	flags, err := binary.ReadUvarint(readBuf)
	if err != nil {
		return err
	}

	message.Direction = MessageDirection(flags & HeaderFlagsMessageDirectionMask)
	compressionMethod := CompressionMethod((flags & HeaderFlagsCompressionMethodMask) >> HeaderFlagsCompressionMethodShift)

	var bodyBytes []byte

	switch compressionMethod {
	case CompressionMethodNONE:
		bodyBytes = readBuf.Bytes()
	case CompressionMethodZLIB:
		var r io.ReadCloser
		r, err = zlib.NewReader(readBuf)
		if err != nil {
			log.Fatal("cannot decode:", err)
		}

		bodyBytes, err = ioutil.ReadAll(r)
		if err != nil {
			log.Fatal("cannot decode:", err)
		}
	}

	bodyType := BodyType((flags & HeaderFlagsBodyTypeMask) >> HeaderFlagsBodyTypeShift)
	body, err := createBody(bodyType)
	if err != nil {
		return err
	}

	err = gogoproto.Unmarshal(bodyBytes, body)
	if err != nil {
		return err
	}

	message.Body = body

	return nil
}

func createBody(bodyType BodyType) (gogoproto.Message, error) {
	switch bodyType {
	case BodyTypeStatus:
		return &status.Status{}, nil
	case BodyTypeExportTraceRequest:
		return &collectortrace.ExportTraceServiceRequest{}, nil
	case BodyTypeExportTraceResponse:
		return &collectortrace.ExportTraceServiceResponse{}, nil
	case BodyTypeExportMetricRequest:
		return &collectormetric.ExportMetricsServiceRequest{}, nil
	case BodyTypeExportMetricResponse:
		return &collectormetric.ExportMetricsServiceResponse{}, nil
	}
	return nil, errors.New("unknown body type")
}
