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

package zipkinreceiver

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/translator/trace/zipkinv1"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
)

const (
	receiverTransportV1Thrift = "http_v1_thrift"
	receiverTransportV1JSON   = "http_v1_json"
	receiverTransportV2JSON   = "http_v2_json"
	receiverTransportV2PROTO  = "http_v2_proto"
)

var errNextConsumerRespBody = []byte(`"Internal Server Error"`)

// ZipkinReceiver type is used to handle spans received in the Zipkin format.
type ZipkinReceiver struct {
	// addr is the address onto which the HTTP server will be bound
	host         component.Host
	nextConsumer consumer.Traces
	id           config.ComponentID

	shutdownWG sync.WaitGroup
	server     *http.Server
	config     *Config

	v1ThriftUnmarshaler      pdata.TracesUnmarshaler
	v1JSONUnmarshaler        pdata.TracesUnmarshaler
	jsonUnmarshaler          pdata.TracesUnmarshaler
	protobufUnmarshaler      pdata.TracesUnmarshaler
	protobufDebugUnmarshaler pdata.TracesUnmarshaler
}

var _ http.Handler = (*ZipkinReceiver)(nil)

// New creates a new zipkinreceiver.ZipkinReceiver reference.
func New(config *Config, nextConsumer consumer.Traces) (*ZipkinReceiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	zr := &ZipkinReceiver{
		nextConsumer:             nextConsumer,
		id:                       config.ID(),
		config:                   config,
		v1ThriftUnmarshaler:      zipkinv1.NewThriftTracesUnmarshaler(),
		v1JSONUnmarshaler:        zipkinv1.NewJSONTracesUnmarshaler(config.ParseStringTags),
		jsonUnmarshaler:          zipkinv2.NewJSONTracesUnmarshaler(config.ParseStringTags),
		protobufUnmarshaler:      zipkinv2.NewProtobufTracesUnmarshaler(false, config.ParseStringTags),
		protobufDebugUnmarshaler: zipkinv2.NewProtobufTracesUnmarshaler(true, config.ParseStringTags),
	}
	return zr, nil
}

// Start spins up the receiver's HTTP server and makes the receiver start its processing.
func (zr *ZipkinReceiver) Start(_ context.Context, host component.Host) error {
	if host == nil {
		return errors.New("nil host")
	}

	zr.host = host
	zr.server = zr.config.HTTPServerSettings.ToServer(zr)
	var listener net.Listener
	listener, err := zr.config.HTTPServerSettings.ToListener()
	if err != nil {
		return err
	}
	zr.shutdownWG.Add(1)
	go func() {
		defer zr.shutdownWG.Done()

		if errHTTP := zr.server.Serve(listener); errHTTP != http.ErrServerClosed {
			host.ReportFatalError(errHTTP)
		}
	}()

	return nil
}

// v1ToTraceSpans parses Zipkin v1 JSON traces and converts them to OpenCensus Proto spans.
func (zr *ZipkinReceiver) v1ToTraceSpans(blob []byte, hdr http.Header) (reqs pdata.Traces, err error) {
	if hdr.Get("Content-Type") == "application/x-thrift" {
		return zr.v1ThriftUnmarshaler.UnmarshalTraces(blob)
	}
	return zr.v1JSONUnmarshaler.UnmarshalTraces(blob)
}

// v2ToTraceSpans parses Zipkin v2 JSON or Protobuf traces and converts them to OpenCensus Proto spans.
func (zr *ZipkinReceiver) v2ToTraceSpans(blob []byte, hdr http.Header) (reqs pdata.Traces, err error) {
	// This flag's reference is from:
	//      https://github.com/openzipkin/zipkin-go/blob/3793c981d4f621c0e3eb1457acffa2c1cc591384/proto/v2/zipkin.proto#L154
	debugWasSet := hdr.Get("X-B3-Flags") == "1"

	// By default, we'll assume using JSON
	unmarshaler := zr.jsonUnmarshaler

	// Zipkin can send protobuf via http
	if hdr.Get("Content-Type") == "application/x-protobuf" {
		// TODO: (@odeke-em) record the unique types of Content-Type uploads
		if debugWasSet {
			unmarshaler = zr.protobufDebugUnmarshaler
		} else {
			unmarshaler = zr.protobufUnmarshaler
		}
	}

	return unmarshaler.UnmarshalTraces(blob)
}

// Shutdown tells the receiver that should stop reception,
// giving it a chance to perform any necessary clean-up and shutting down
// its HTTP server.
func (zr *ZipkinReceiver) Shutdown(context.Context) error {
	err := zr.server.Close()
	zr.shutdownWG.Wait()
	return err
}

// processBodyIfNecessary checks the "Content-Encoding" HTTP header and if
// a compression such as "gzip", "deflate", "zlib", is found, the body will
// be uncompressed accordingly or return the body untouched if otherwise.
// Clients such as Zipkin-Java do this behavior e.g.
//    send "Content-Encoding":"gzip" of the JSON content.
func processBodyIfNecessary(req *http.Request) io.Reader {
	switch req.Header.Get("Content-Encoding") {
	default:
		return req.Body

	case "gzip":
		return gunzippedBodyIfPossible(req.Body)

	case "deflate", "zlib":
		return zlibUncompressedbody(req.Body)
	}
}

func gunzippedBodyIfPossible(r io.Reader) io.Reader {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		// Just return the old body as was
		return r
	}
	return gzr
}

func zlibUncompressedbody(r io.Reader) io.Reader {
	zr, err := zlib.NewReader(r)
	if err != nil {
		// Just return the old body as was
		return r
	}
	return zr
}

const (
	zipkinV1TagValue = "zipkinV1"
	zipkinV2TagValue = "zipkinV2"
)

// The ZipkinReceiver receives spans from endpoint /api/v2 as JSON,
// unmarshals them and sends them along to the nextConsumer.
func (zr *ZipkinReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if c, ok := client.FromHTTP(r); ok {
		ctx = client.NewContext(ctx, c)
	}

	// Now deserialize and process the spans.
	asZipkinv1 := r.URL != nil && strings.Contains(r.URL.Path, "api/v1/spans")

	transportTag := transportType(r, asZipkinv1)
	obsrecv := obsreport.NewReceiver(obsreport.ReceiverSettings{ReceiverID: zr.id, Transport: transportTag})
	ctx = obsrecv.StartTracesOp(ctx)

	pr := processBodyIfNecessary(r)
	slurp, _ := ioutil.ReadAll(pr)
	if c, ok := pr.(io.Closer); ok {
		_ = c.Close()
	}
	_ = r.Body.Close()

	var td pdata.Traces
	var err error
	if asZipkinv1 {
		td, err = zr.v1ToTraceSpans(slurp, r.Header)
	} else {
		td, err = zr.v2ToTraceSpans(slurp, r.Header)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	consumerErr := zr.nextConsumer.ConsumeTraces(ctx, td)

	receiverTagValue := zipkinV2TagValue
	if asZipkinv1 {
		receiverTagValue = zipkinV1TagValue
	}
	obsrecv.EndTracesOp(ctx, receiverTagValue, td.SpanCount(), consumerErr)

	if consumerErr != nil {
		// Transient error, due to some internal condition.
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errNextConsumerRespBody) // nolint:errcheck
		return
	}

	// Finally send back the response "Accepted" as
	// required at https://zipkin.io/zipkin-api/#/default/post_spans
	w.WriteHeader(http.StatusAccepted)
}

func transportType(r *http.Request, asZipkinv1 bool) string {
	if asZipkinv1 {
		if r.Header.Get("Content-Type") == "application/x-thrift" {
			return receiverTransportV1Thrift
		}
		return receiverTransportV1JSON
	}
	if r.Header.Get("Content-Type") == "application/x-protobuf" {
		return receiverTransportV2PROTO
	}
	return receiverTransportV2JSON
}
