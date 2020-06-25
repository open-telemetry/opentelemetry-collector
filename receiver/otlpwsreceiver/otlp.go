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

package otlpwsreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	collectormetric "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/trace"
	"go.opentelemetry.io/collector/receiver/otlpwsreceiver/encoding"
)

// Receiver is the type that exposes Trace and Metrics reception.
type Receiver struct {
	logger     *zap.Logger
	mu         sync.Mutex
	ln         net.Listener
	serverHTTP *http.Server

	traceReceiver   *trace.Receiver
	metricsReceiver *metrics.Receiver

	traceConsumer   consumer.TraceConsumer
	metricsConsumer consumer.MetricsConsumer

	stopOnce                 sync.Once
	startTraceReceiverOnce   sync.Once
	startMetricsReceiverOnce sync.Once

	instanceName string
}

var upgrader = websocket.Upgrader{} // use default options

// New just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func New(
	logger *zap.Logger,
	instanceName string,
	transport string,
	addr string,
	tc consumer.TraceConsumer,
	mc consumer.MetricsConsumer,
) (*Receiver, error) {
	ln, err := net.Listen(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %q: %v", addr, err)
	}

	r := &Receiver{
		ln:              ln,
		logger:          logger,
		traceConsumer:   tc,
		metricsConsumer: mc,
	}

	r.instanceName = instanceName

	return r, nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *Receiver) Start(ctx context.Context, host component.Host) error {
	hasConsumer := false
	if r.traceConsumer != nil {
		hasConsumer = true
		if err := r.registerTraceConsumer(); err != nil && err != componenterror.ErrAlreadyStarted {
			return err
		}
	}

	if r.metricsConsumer != nil {
		hasConsumer = true
		if err := r.registerMetricsConsumer(); err != nil && err != componenterror.ErrAlreadyStarted {
			return err
		}
	}

	if !hasConsumer {
		return errors.New("cannot start receiver: no consumers were specified")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ws", func(w http.ResponseWriter, resp *http.Request) {
		r.onReceive(w, resp)
	})
	server := &http.Server{
		Handler: mux,
	}
	r.serverHTTP = server
	go func() {
		err := server.Serve(r.ln)
		if err != nil && err != http.ErrServerClosed {
			host.ReportFatalError(err)
		}
	}()
	return nil
}

func (r *Receiver) onReceive(w http.ResponseWriter, resp *http.Request) {
	r.logger.Debug("Incoming connection")
	c, err := upgrader.Upgrade(w, resp, nil)
	if err != nil {
		r.logger.Error("cannot upgrade WebSocket", zap.Error(err))
		return
	}
	defer c.Close()
	lastID := uint64(0)

	encoder := encoding.NewEncoder()

	for {
		mt, bytes, err := c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				r.logger.Error("cannot read from WebSocket", zap.Error(err))
			}
			break
		}
		if mt != websocket.BinaryMessage {
			r.logger.Error("expecting binary message type but got a different type", zap.Int("mt", mt))
			continue
		}

		var message encoding.Message
		err = encoding.Decode(bytes, &message)
		if err != nil {
			r.logger.Debug("Cannot decode the message", zap.Error(err))
			continue
		}

		id := message.ID
		if id == 0 {
			r.logger.Debug("Received message with 0 Id")
		}
		if id != lastID+1 {
			r.logger.Debug("Received out of order request",
				zap.Uint64("expected ID", lastID+1), zap.Uint64("actual ID", id))
		}
		lastID = id

		var responseBody gogoproto.Message
		switch body := message.Body.(type) {
		case *collectortrace.ExportTraceServiceRequest:
			responseBody, err = r.traceReceiver.Export(resp.Context(), body)
		case *collectormetric.ExportMetricsServiceRequest:
			responseBody, err = r.metricsReceiver.Export(resp.Context(), body)
		default:
			r.logger.Debug("Received unexpected body type. Ignoring.")
			continue
		}

		if err != nil {
			r.logger.Debug("Failed to process the request", zap.Error(err))
			responseBody = &status.Status{
				Code:    0, // TODO: return proper error code.
				Message: err.Error(),
				Details: nil,
			}
		}

		responseMsg := encoding.Message{
			ID:        message.ID,
			Body:      responseBody,
			Direction: encoding.MessageDirectionResponse,
		}

		responseBytes, err := encoder.Encode(&responseMsg, encoding.CompressionMethodNONE)
		if err != nil {
			r.logger.Debug("cannot encode the response:", zap.Error(err))
			continue
		}

		err = c.WriteMessage(websocket.BinaryMessage, responseBytes)
		if err != nil {
			r.logger.Debug("cannot send the response:", zap.Error(err))
			continue
		}
	}
}

func (r *Receiver) registerTraceConsumer() error {
	var err = componenterror.ErrAlreadyStarted

	r.startTraceReceiverOnce.Do(func() {
		r.traceReceiver, err = trace.New(r.instanceName, r.traceConsumer)
		if err != nil {
			return
		}
	})

	return err
}

func (r *Receiver) registerMetricsConsumer() error {
	var err = componenterror.ErrAlreadyStarted

	r.startMetricsReceiverOnce.Do(func() {
		r.metricsReceiver, err = metrics.New(r.instanceName, r.metricsConsumer)
		if err != nil {
			return
		}
	})

	return err
}

// Shutdown is a method to turn off receiving.
func (r *Receiver) Shutdown(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err = componenterror.ErrAlreadyStopped
	r.stopOnce.Do(func() {
		err = nil

		if r.serverHTTP != nil {
			_ = r.serverHTTP.Close()
		}

		if r.ln != nil {
			_ = r.ln.Close()
		}
	})
	return err
}
