// Copyright 2018, OpenCensus Authors
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

package jaeger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	tchannel "github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/thrift"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/receiver"
	"github.com/census-instrumentation/opencensus-service/translator/trace"
)

// Receiver type is used to receive spans that were originally intended to be sent to Jaeger.
// This receiver is basically a Jaeger collector.
type jReceiver struct {
	// mu protects the fields of this type
	mu sync.Mutex

	spanSink receiver.TraceReceiverSink

	startOnce sync.Once
	stopOnce  sync.Once

	tchannelPort      int
	collectorHTTPPort int

	tchannel        *tchannel.Channel
	collectorServer *http.Server
}

const (
	// As per https://www.jaegertracing.io/docs/1.7/deployment/
	// By default, the port used by jaeger-agent to send spans in jaeger.thrift format
	defaultTChannelPort = 14267
	// By default, can accept spans directly from clients in jaeger.thrift format over binary thrift protocol
	defaultCollectorHTTPPort = 14268
)

// New creates a TraceReceiver that receives traffic as a collector with both Thrift and HTTP transports.
func New(ctx context.Context, tchannelPort, collectorHTTPPort int) (receiver.TraceReceiver, error) {
	return &jReceiver{tchannelPort: tchannelPort, collectorHTTPPort: collectorHTTPPort}, nil
}

var _ receiver.TraceReceiver = (*jReceiver)(nil)

var (
	errAlreadyStarted = errors.New("already started")
	errAlreadyStopped = errors.New("already stopped")
)

func (jr *jReceiver) collectorAddr() string {
	port := jr.collectorHTTPPort
	if port <= 0 {
		port = defaultCollectorHTTPPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) tchannelAddr() string {
	port := jr.tchannelPort
	if port <= 0 {
		port = defaultTChannelPort
	}
	return fmt.Sprintf(":%d", port)
}

func (jr *jReceiver) StartTraceReception(ctx context.Context, spanSink receiver.TraceReceiverSink) error {
	jr.mu.Lock()
	defer jr.mu.Unlock()

	var err = errAlreadyStarted
	jr.startOnce.Do(func() {
		tch, terr := tchannel.NewChannel("recv", new(tchannel.ChannelOptions))
		if terr != nil {
			err = fmt.Errorf("Failed to create NewTChannel: %v", terr)
			return
		}

		taddr := jr.tchannelAddr()
		tln, terr := net.Listen("tcp", taddr)
		if terr != nil {
			err = fmt.Errorf("Failed to bind to TChannnel address %q: %v", taddr, terr)
			return
		}
		tch.Serve(tln)
		jr.tchannel = tch

		// Now the collector that runs over HTTP
		caddr := jr.collectorAddr()
		cln, cerr := net.Listen("tcp", caddr)
		if cerr != nil {
			// Abort and close tch
			tch.Close()
			err = fmt.Errorf("Failed to bind to Collector address %q: %v", caddr, cerr)
			return
		}

		nr := mux.NewRouter()
		apiHandler := app.NewAPIHandler(jr)
		apiHandler.RegisterRoutes(nr)
		jr.collectorServer = &http.Server{Handler: nr}
		go func() {
			_ = jr.collectorServer.Serve(cln)
		}()

		// Otherwise no error was encountered,
		// finally set the spanSink
		jr.spanSink = spanSink
		err = nil
	})
	return err
}

func (jr *jReceiver) StopTraceReception(ctx context.Context) error {
	jr.mu.Lock()
	defer jr.mu.Unlock()

	var err = errAlreadyStopped
	jr.stopOnce.Do(func() {
		var errs []error
		if jr.collectorServer != nil {
			if cerr := jr.collectorServer.Close(); cerr != nil {
				errs = append(errs, cerr)
			}
			jr.collectorServer = nil
		}
		if jr.tchannel != nil {
			jr.tchannel.Close()
			jr.tchannel = nil
		}
		if len(errs) == 0 {
			err = nil
			return
		}

		// Otherwise combine all these errors
		buf := new(bytes.Buffer)
		for _, err := range errs {
			fmt.Fprintf(buf, "%s\n", err.Error())
		}
		err = errors.New(buf.String())
	})

	return err
}

func (jr *jReceiver) SubmitBatches(ctx thrift.Context, batches []*jaeger.Batch) ([]*jaeger.BatchSubmitResponse, error) {
	jbsr := make([]*jaeger.BatchSubmitResponse, 0, len(batches))

	for _, batch := range batches {
		octrace, err := tracetranslator.JaegerThriftBatchToOCProto(batch)
		// TODO: (@odeke-em) add this error for Jaeger observability
		ok := false

		if err == nil && octrace != nil {
			ok = true
			jr.spanSink.ReceiveTraceData(ctx, data.TraceData{Node: octrace.Node, Spans: octrace.Spans})
		}

		jbsr = append(jbsr, &jaeger.BatchSubmitResponse{
			Ok: ok,
		})
	}
	return jbsr, nil
}
