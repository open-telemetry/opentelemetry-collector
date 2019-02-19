// Copyright 2019, OpenCensus Authors
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

package scribe

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"sync"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"github.com/omnition/scribe-go/if/scribe/gen-go/scribe"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/receiver"
	zipkintranslator "github.com/census-instrumentation/opencensus-service/translator/trace/zipkin"
)

var (
	errAlreadyStarted = errors.New("already started")
	errAlreadyStopped = errors.New("already stopped")
)

var _ receiver.TraceReceiver = (*scribeReceiver)(nil)

// scribeReceiver implements the receiver.TraceReceiver for Zipkin Scribe protocol.
type scribeReceiver struct {
	sync.Mutex
	addr      string
	port      uint16
	collector *scribeCollector
	server    *thrift.TSimpleServer

	startOnce sync.Once
	stopOnce  sync.Once
}

// NewReceiver creates the Zipkin Scribe receiver with the given parameters.
func NewReceiver(addr string, port uint16, category string) (receiver.TraceReceiver, error) {
	r := &scribeReceiver{
		addr: addr,
		port: port,
		collector: &scribeCollector{
			category:            category,
			msgDecoder:          base64.StdEncoding.WithPadding('='),
			tBinProtocolFactory: thrift.NewTBinaryProtocolFactory(true, false),
		},
	}
	return r, nil
}

func (r *scribeReceiver) StartTraceReception(ctx context.Context, destination receiver.TraceReceiverSink) error {
	r.Lock()
	defer r.Unlock()

	if destination == nil {
		return errors.New("trace reception requires a non-nil destination")
	}

	err := errAlreadyStarted
	r.startOnce.Do(func() {
		err = nil
		r.collector.traceSink = destination
		serverSocket, sockErr := thrift.NewTServerSocket(r.addr + ":" + strconv.Itoa(int(r.port)))
		if sockErr != nil {
			err = sockErr
			return
		}

		sockErr = serverSocket.Open()
		if sockErr != nil {
			err = sockErr
			return
		}

		r.server = thrift.NewTSimpleServer4(
			scribe.NewScribeProcessor(r.collector),
			serverSocket,
			thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory()),
			thrift.NewTBinaryProtocolFactory(true, false),
		)

		// The interface doesn't support reporting anything from the async function.
		go r.server.Serve()
	})

	return err
}

func (r *scribeReceiver) StopTraceReception(ctx context.Context) error {
	r.Lock()
	defer r.Unlock()

	var err = errAlreadyStopped
	r.stopOnce.Do(func() {
		err = r.server.Stop()
	})
	return err
}

// scribeCollector implements the Thrift interface of a scribe server as supported by zipkin.
// See https://github.com/openzipkin/zipkin/tree/master/zipkin-collector/scribe
type scribeCollector struct {
	category            string
	msgDecoder          *base64.Encoding
	tBinProtocolFactory *thrift.TBinaryProtocolFactory
	traceSink           receiver.TraceReceiverSink
}

var _ scribe.Scribe = (*scribeCollector)(nil)

// Log is the function that receives the messages sent to the scribe server. It is required
func (sc *scribeCollector) Log(messages []*scribe.LogEntry) (r scribe.ResultCode, err error) {
	zSpans := make([]*zipkincore.Span, 0, len(messages))
	for _, logEntry := range messages {
		if sc.category != logEntry.Category {
			// Not the specified category, do nothing
			continue
		}

		b, err := sc.msgDecoder.DecodeString(logEntry.Message)
		if err != nil {
			return scribe.ResultCode_OK, err
		}

		r := bytes.NewReader(b)
		st := thrift.NewStreamTransportR(r)
		zs := &zipkincore.Span{}
		if err := zs.Read(sc.tBinProtocolFactory.GetProtocol(st)); err != nil {
			return scribe.ResultCode_OK, err
		}

		zSpans = append(zSpans, zs)
	}

	if len(zSpans) == 0 {
		return scribe.ResultCode_OK, nil
	}

	ocBatches, err := zipkintranslator.V1ThriftBatchToOCProto(zSpans)
	if err != nil {
		return scribe.ResultCode_OK, err
	}

	ctx := context.Background()
	spansMetricsFn := internal.NewReceivedSpansRecorderStreaming(ctx, "zipkin-scribe")

	for _, ocBatch := range ocBatches {
		sc.traceSink.ReceiveTraceData(ctx, data.TraceData{Node: ocBatch.Node, Spans: ocBatch.Spans})
		spansMetricsFn(ocBatch.Node, ocBatch.Spans)
	}

	return scribe.ResultCode_OK, nil
}
