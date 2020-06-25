// Copyright  OpenTelemetry Authors
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

package otlpwsexporter

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver/otlpwsreceiver/encoding"
)

type exporterImp struct {
	logger *zap.Logger

	// Input configuration.
	config *Config

	stopOnce sync.Once

	conn            *websocket.Conn
	pendingAck      map[uint64]*encoding.Message
	pendingAckMutex sync.Mutex
	nextID          uint64
	Compression     encoding.CompressionMethod

	encoder encoding.Encoder
}

var dialer = &websocket.Dialer{
	Proxy:            http.ProxyFromEnvironment,
	HandshakeTimeout: 45 * time.Second,
	// Ensure large enough buffer so that big messages do not get fragmented too much
	// (which degrades performance). See https://godoc.org/github.com/gorilla/websocket#hdr-Buffers
	// Note that we don't care about read buffer size since the exporter does not receive
	// anything large.
	WriteBufferSize: 4000 * 1024,
}

// Crete new exporter and start it. The exporter will begin connecting but
// this function may return before the connection is established.
func newExporter(config *Config, logger *zap.Logger) (*exporterImp, error) {
	e := &exporterImp{}
	e.config = config
	e.logger = logger
	e.encoder = encoding.NewEncoder()

	// Set up a connection to the server.
	e.pendingAck = make(map[uint64]*encoding.Message)

	u := url.URL{Scheme: "ws", Host: e.config.Endpoint, Path: "/v1/ws"}

	var err error
	e.conn, _, err = dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	go e.readStream()

	return e, nil
}

func (e *exporterImp) stop() error {
	var err error
	e.stopOnce.Do(func() {
		// Close the connection.
		err = e.conn.Close()
	})
	return err
}

// Send a trace or metrics request to the server. "perform" function is expected to make
// the actual gRPC unary call that sends the request. This function implements the
// common OTLP logic around request handling such as retries and throttling.
func (e *exporterImp) exportRequest(ctx context.Context, body gogoproto.Message) error {

	id := atomic.AddUint64(&e.nextID, 1)
	message := encoding.Message{
		ID:        id,
		Body:      body,
		Direction: encoding.MessageDirectionRequest,
	}

	bytes, err := e.encoder.Encode(&message, e.Compression)
	if err != nil {
		return err
	}

	// Add the ID to pendingAck map
	e.pendingAckMutex.Lock()
	e.pendingAck[id] = &message
	e.pendingAckMutex.Unlock()

	err = e.conn.WriteMessage(websocket.BinaryMessage, bytes)
	if err != nil {
		return err
	}
	return nil
}

func (e *exporterImp) readStream() {
	lastID := uint64(0)
	for {
		mt, bytes, err := e.conn.ReadMessage()
		if err != nil {
			e.logger.Error("cannot read from WebSocket", zap.Error(err))
			break
		}
		if mt != websocket.BinaryMessage {
			e.logger.Error("expecting binary message type but got a different type", zap.Int("mt", mt))
			continue
		}

		var message encoding.Message
		err = encoding.Decode(bytes, &message)
		if err != nil {
			e.logger.Debug("Cannot decode the message", zap.Error(err))
		}

		id := message.ID
		if id != lastID+1 {
			log.Fatalf("Received out of order response ID=%d", id)
		}
		lastID = id

		e.pendingAckMutex.Lock()
		_, ok := e.pendingAck[id]
		if !ok {
			e.pendingAckMutex.Unlock()
			log.Fatalf("Received ack on batch ID that does not exist: %v", id)
		}
		delete(e.pendingAck, id)
		e.pendingAckMutex.Unlock()
	}
}
