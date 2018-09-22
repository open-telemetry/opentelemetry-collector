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

package ocinterceptor

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/api/support/bundler"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
)

type OCInterceptor struct {
	spanSink         spanreceiver.SpanReceiver
	spanBufferPeriod time.Duration
	spanBufferCount  int
}

func New(sr spanreceiver.SpanReceiver, opts ...OCOption) (*OCInterceptor, error) {
	if sr == nil {
		return nil, errors.New("needs a non-nil spanReceiver")
	}
	oci := &OCInterceptor{spanSink: sr}
	for _, opt := range opts {
		opt.WithOCInterceptor(oci)
	}
	return oci, nil
}

var _ agenttracepb.TraceServiceServer = (*OCInterceptor)(nil)

var errUnimplemented = errors.New("unimplemented")

func (oci *OCInterceptor) Config(tcs agenttracepb.TraceService_ConfigServer) error {
	// TODO: Implement when we define the config receiver/sender.
	return errUnimplemented
}

// Export is the gRPC method that receives streamed traces from
// OpenCensus-traceproto compatible libraries/applications.
func (oci *OCInterceptor) Export(tes agenttracepb.TraceService_ExportServer) error {
	// Firstly we MUST receive the node identifier to initiate
	// the service and start accepting exported spans.
	const maxTraceInitRetries = 15 // Arbitrary value

	var node *commonpb.Node
	for i := 0; i < maxTraceInitRetries; i++ {
		recv, err := tes.Recv()
		if err != nil {
			return err
		}

		if nd := recv.Node; nd != nil {
			node = nd
			break
		}
	}

	if node == nil {
		return fmt.Errorf("failed to receive a non-nil Node even after %d retries", maxTraceInitRetries)
	}

	// Now that we've got the node, we can start to receive streamed up spans.
	// The bundler will receive batches of spans i.e. []*tracepb.Span
	traceBundler := bundler.NewBundler(([]*tracepb.Span)(nil), func(payload interface{}) {
		listOfSpanLists := payload.([][]*tracepb.Span)
		flattenedListOfSpans := make([]*tracepb.Span, 0, len(listOfSpanLists)) // In the best case, each spanList has 1 span
		for _, listOfSpans := range listOfSpanLists {
			flattenedListOfSpans = append(flattenedListOfSpans, listOfSpans...)
		}
		oci.spanSink.ReceiveSpans(node, flattenedListOfSpans...)
	})
	spanBufferPeriod := oci.spanBufferPeriod
	if spanBufferPeriod <= 0 {
		spanBufferPeriod = 2 * time.Second // Arbitrary value
	}
	spanBufferCount := oci.spanBufferCount
	if spanBufferCount <= 0 {
		// TODO: (@odeke-em) provide an option to disable any buffering
		spanBufferCount = 50 // Arbitrary value
	}

	traceBundler.DelayThreshold = spanBufferPeriod
	traceBundler.BundleCountThreshold = spanBufferCount

	for {
		recv, err := tes.Recv()
		if err != nil {
			return err
		}

		// Otherwise add these spans to the bundler
		traceBundler.Add(recv.Spans, len(recv.Spans))
	}
}
