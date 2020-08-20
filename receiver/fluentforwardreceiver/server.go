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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/tinylib/msgp/msgp"
	"go.opencensus.io/stats"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver/fluentforwardreceiver/observ"
)

// The initial size of the read buffer. Messages can come in that are bigger
// than this, but this serves as a starting point.
const readBufferSize = 10 * 1024

type server struct {
	outCh  chan<- Event
	logger *zap.Logger
}

func newServer(outCh chan<- Event, logger *zap.Logger) *server {
	return &server{
		outCh:  outCh,
		logger: logger,
	}
}

func (s *server) Start(ctx context.Context, listener net.Listener) {
	go func() {
		s.handleConnections(ctx, listener)
		if ctx.Err() == nil {
			panic("logic error in receiver, connections should always be listened for while receiver is running")
		}
	}()
}

func (s *server) handleConnections(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if ctx.Err() != nil {
			return
		}
		// If there is an error and the receiver isn't shutdown, we need to
		// keep trying to accept connections if at all possible. Put in a sleep
		// to prevent hot loops in case the error persists.
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		stats.Record(ctx, observ.ConnectionsOpened.M(1))

		s.logger.Debug("Got connection", zap.String("remoteAddr", conn.RemoteAddr().String()))

		go func() {
			defer stats.Record(ctx, observ.ConnectionsClosed.M(1))

			err := s.handleConn(ctx, conn)
			if err != nil {
				if err == io.EOF {
					s.logger.Debug("Closing connection", zap.String("remoteAddr", conn.RemoteAddr().String()), zap.Error(err))
				} else {
					s.logger.Debug("Unexpected error handling connection", zap.String("remoteAddr", conn.RemoteAddr().String()), zap.Error(err))
				}
			}
			conn.Close()
		}()
	}
}

func (s *server) handleConn(ctx context.Context, conn net.Conn) error {
	reader := msgp.NewReaderSize(conn, readBufferSize)

	for {
		mode, err := DetermineNextEventMode(reader.R)
		if err != nil {
			return err
		}

		var event Event
		switch mode {
		case UnknownMode:
			return errors.New("could not determine event mode")
		case MessageMode:
			event = &MessageEventLogRecord{}
		case ForwardMode:
			event = &ForwardEventLogRecords{}
		case PackedForwardMode:
			event = &PackedForwardEventLogRecords{}
		default:
			panic("programmer bug in mode handling")
		}

		err = event.DecodeMsg(reader)
		if err != nil {
			if err != io.EOF {
				stats.Record(ctx, observ.FailedToParse.M(1))
			}
			return fmt.Errorf("failed to parse %s mode event: %v", mode.String(), err)
		}

		stats.Record(ctx, observ.EventsParsed.M(1))

		s.outCh <- event

		// We must acknowledge the 'chunk' option if given. We could do this in
		// another goroutine if it is too much of a bottleneck to reading
		// messages -- this is the only thing that sends data back to the
		// client.
		if event.Chunk() != "" {
			err := msgp.Encode(conn, AckResponse{Ack: event.Chunk()})
			if err != nil {
				return fmt.Errorf("failed to acknowledge chunk %s: %v", event.Chunk(), err)
			}
		}
	}
}

// DetermineNextEventMode inspects the next bit of data from the given peeker
// reader to determine which type of event mode it is.  According to the
// forward protocol spec: "Server MUST detect the carrier mode by inspecting
// the second element of the array."  It is assumed that peeker is aligned at
// the start of a new event, otherwise the result is undefined and will
// probably error.
func DetermineNextEventMode(peeker Peeker) (EventMode, error) {
	var chunk []byte
	var err error
	chunk, err = peeker.Peek(2)
	if err != nil {
		return UnknownMode, err
	}

	// The first byte is the array header, which will always be 1 byte since no
	// message modes have more than 4 entries. So skip to the second byte which
	// is the tag string header.
	tagType := chunk[1]
	// We already read the first type for the type
	tagLen := 1

	isFixStr := tagType&0b10100000 == 0b10100000
	if isFixStr {
		tagLen += int(tagType & 0b00011111)
	} else {
		switch tagType {
		case 0xd9:
			chunk, err = peeker.Peek(3)
			if err != nil {
				return UnknownMode, err
			}
			tagLen += 1 + int(chunk[2])
		case 0xda:
			chunk, err = peeker.Peek(4)
			if err != nil {
				return UnknownMode, err
			}
			tagLen += 2 + int(binary.BigEndian.Uint16(chunk[2:]))
		case 0xdb:
			chunk, err = peeker.Peek(6)
			if err != nil {
				return UnknownMode, err
			}
			tagLen += 4 + int(binary.BigEndian.Uint32(chunk[2:]))
		default:
			return UnknownMode, errors.New("malformed tag field")
		}
	}

	// Skip past the first byte (array header) and the entire tag and then get
	// one byte into the second field -- that is enough to know its type.
	chunk, err = peeker.Peek(1 + tagLen + 1)
	if err != nil {
		return UnknownMode, err
	}

	secondElmType := msgp.NextType(chunk[1+tagLen:])

	switch secondElmType {
	case msgp.IntType, msgp.UintType, msgp.ExtensionType:
		return MessageMode, nil
	case msgp.ArrayType:
		return ForwardMode, nil
	case msgp.BinType, msgp.StrType:
		return PackedForwardMode, nil
	default:
		return UnknownMode, fmt.Errorf("unable to determine next event mode for type %v", secondElmType)
	}
}
