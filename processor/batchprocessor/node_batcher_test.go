// Copyright 2019, OpenTelemetry Authors
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

package batchprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type bucketIDTestInput struct {
	node     *commonpb.Node
	resource *resourcepb.Resource
	format   string
}

func BenchmarkGenBucketID(b *testing.B) {
	sender := newTestSender()
	batcher := NewBatcher("test", zap.NewNop(), sender).(*batcher)
	gens := map[string]func(*commonpb.Node, *resourcepb.Resource, string) string{
		"composite-md5": batcher.genBucketID,
	}

	inputSmall := bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "oc"}
	inputBig := bucketIDTestInput{
		&commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "svc-i-am-a-cat"},
			LibraryInfo: &commonpb.LibraryInfo{ExporterVersion: "v1.2.3", CoreLibraryVersion: "v1.2.4", Language: commonpb.LibraryInfo_GO_LANG},
		},
		&resourcepb.Resource{Labels: map[string]string{
			"asdfasdfasdfasdfasdf":  "bsdfasdfasdfasdfasdf",
			"asdfssssssssasdfasdf":  "bsdfasdfasdfasdfasdf",
			"skarisskarisskarisdf":  "bsdfasdfasdfasdfasdf",
			"iamacatiamacatiamacat": "bsdfasdfasdfasdfasdf",
		}},
		"oc",
	}

	for genName, gen := range gens {
		for name, input := range map[string]bucketIDTestInput{"smallInput": inputSmall, "bigInput": inputBig} {
			b.Run(genName+"-"+name, func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					gen(input.node, input.resource, input.format)
				}
			})
		}
	}
}

func TestGenBucketID(t *testing.T) {
	testCases := []struct {
		name   string
		match  bool
		input1 bucketIDTestInput
		input2 bucketIDTestInput
	}{
		{
			"different span formats",
			false,
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "oc"},
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "zipkin"},
		},
		{
			"identical but different node objects",
			true,
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "oc"},
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "oc"},
		},
		{
			"different nodes",
			false,
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}}, nil, "oc"},
			bucketIDTestInput{&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc2"}}, nil, "oc"},
		},
		{
			"different resources",
			false,
			bucketIDTestInput{
				&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}},
				&resourcepb.Resource{Labels: map[string]string{"a": "b"}},
				"oc",
			},
			bucketIDTestInput{
				&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}},
				&resourcepb.Resource{Labels: map[string]string{"a": "c"}},
				"oc",
			},
		},
		{
			"identical but different resources",
			true,
			bucketIDTestInput{
				&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}},
				&resourcepb.Resource{Labels: map[string]string{"a": "b"}},
				"oc",
			},
			bucketIDTestInput{
				&commonpb.Node{ServiceInfo: &commonpb.ServiceInfo{Name: "svc"}},
				&resourcepb.Resource{Labels: map[string]string{"a": "b"}},
				"oc",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := newTestSender()
			batcher := NewBatcher("test", zap.NewNop(), sender).(*batcher)

			key1 := batcher.genBucketID(tc.input1.node, tc.input1.resource, tc.input1.format)
			key2 := batcher.genBucketID(tc.input2.node, tc.input2.resource, tc.input2.format)

			if tc.match != (key1 == key2) {
				t.Errorf("Keys should be matching=%v but were matching=%v", tc.match, key1 == key2)
			}
		})
	}
}

func TestConcurrentNodeAdds(t *testing.T) {
	sender := newTestSender()
	batcher := NewBatcher("test", zap.NewNop(), sender).(*batcher)
	requestCount := 1000
	spansPerRequest := 100
	waitForCn := sender.waitFor(requestCount*spansPerRequest, 3*time.Second)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := make([]*tracepb.Span, 0, spansPerRequest)
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans = append(spans, &tracepb.Span{Name: getTestSpanName(requestNum, spanIndex)})
		}
		td := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: fmt.Sprintf("svc-%d", requestNum)},
			},
			Spans:        spans,
			SourceFormat: "oc_trace",
		}
		go batcher.ConsumeTraceData(context.Background(), td)
	}

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}
	if len(sender.spansReceivedByName) != requestCount*spansPerRequest {
		t.Errorf("Did not receive the correct number of spans. Got %d != expected %d.", len(sender.spansReceivedByName), requestCount*spansPerRequest)
		return
	}

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			name := getTestSpanName(requestNum, spanIndex).Value
			if sender.spansReceivedByName[name] == nil {
				t.Errorf("Did not receive span %s.", name)
				return
			}
		}
	}
}

func TestBucketRemove(t *testing.T) {
	sender := newTestSender()
	tickTime := 50 * time.Millisecond
	removeAfterTicks := 2
	batcher := NewBatcher(
		"test",
		zap.NewNop(),
		sender,
		WithTimeout(50*time.Millisecond),
		WithTickTime(tickTime),
		WithRemoveAfterTicks(removeAfterTicks),
	).(*batcher)
	spansPerRequest := 3
	waitForCn := sender.waitFor(spansPerRequest, 1*time.Second)
	spans := make([]*tracepb.Span, 0, spansPerRequest)
	for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
		spans = append(spans, &tracepb.Span{Name: getTestSpanName(0, spanIndex)})
	}
	request := consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "svc"},
		},
		Spans:        spans,
		SourceFormat: "oc_trace",
	}
	batcher.ConsumeTraceData(context.Background(), request)

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}

	if batcher.getBucket(batcher.genBucketID(request.Node, nil, "oc_trace")) == nil {
		t.Errorf("Bucket should exist but does not.")
	}

	// Doesn't seem to be a great way to test this without waiting
	<-time.After(2 * time.Duration(removeAfterTicks) * tickTime)

	if batcher.getBucket(batcher.genBucketID(request.Node, nil, "oc_trace")) != nil {
		t.Errorf("Bucket should be deleted but is not.")
	}
}

func TestBucketTickerStop(t *testing.T) {
	sender := newTestSender()
	tickTime := 50 * time.Millisecond
	removeAfterTicks := 2
	batcher := NewBatcher(
		"test",
		zap.NewNop(),
		sender,
		WithTimeout(50*time.Millisecond),
		WithTickTime(tickTime),
		WithRemoveAfterTicks(removeAfterTicks),
	).(*batcher)

	// Stop all the tickers which should prevent the node batches from getting removed and the spans from timing
	// out
	for _, ticker := range batcher.tickers {
		ticker.stop()
	}

	spansPerRequest := 3
	waitForCn := sender.waitFor(spansPerRequest, 3*time.Duration(removeAfterTicks)*tickTime)
	spans := make([]*tracepb.Span, 0, spansPerRequest)
	for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
		spans = append(spans, &tracepb.Span{Name: getTestSpanName(0, spanIndex)})
	}
	request := consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "svc"},
		},
		Spans:        spans,
		SourceFormat: "oc_trace",
	}
	batcher.ConsumeTraceData(context.Background(), request)

	err := <-waitForCn
	if err == nil {
		t.Errorf("Unexpectedly received spans")
	}

	if batcher.getBucket(batcher.genBucketID(request.Node, nil, "oc_trace")) == nil {
		t.Errorf("Bucket should not be deleted but is.")
	}
}

func TestConcurrentBatchAdds(t *testing.T) {
	sender := newTestSender()
	batcher := NewBatcher("test", zap.NewNop(), sender, WithSendBatchSize(128)).(*batcher)
	requestCount := 1000
	spansPerRequest := 100
	waitForCn := sender.waitFor(requestCount*spansPerRequest, 5*time.Second)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := make([]*tracepb.Span, 0, spansPerRequest)
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans = append(spans, &tracepb.Span{Name: getTestSpanName(requestNum, spanIndex)})
		}
		request := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: "svc"},
			},
			Spans:        spans,
			SourceFormat: "oc_trace",
		}
		go batcher.ConsumeTraceData(context.Background(), request)
	}

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}
	if len(sender.spansReceivedByName) != requestCount*spansPerRequest {
		t.Errorf("Did not receive the correct number of spans. %d != %d", len(sender.spansReceivedByName), requestCount*spansPerRequest)
	}

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			if name := sender.spansReceivedByName[getTestSpanName(requestNum, spanIndex).Value]; name == nil {
				t.Errorf("Did not receive span %s.", name)
			}
		}
	}
}

func BenchmarkConcurrentBatchAdds(b *testing.B) {
	sender1 := newNopSender()
	batcher := NewBatcher("test", zap.NewNop(), sender1).(*batcher)
	spansPerRequest := 1000
	var requests []consumerdata.TraceData
	spans := make([]*tracepb.Span, 0, spansPerRequest)
	for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
		spans = append(spans, &tracepb.Span{Name: getTestSpanName(0, spanIndex)})
	}
	request := consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "svc"},
		},
		Spans:        spans,
		SourceFormat: "oc_trace",
	}
	requests = append(requests, request)

	b.Run("v1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, td := range requests {
				_ = batcher.ConsumeTraceData(context.Background(), td)
			}
		}
	})
}

func getTestSpanName(requestNum, index int) *tracepb.TruncatableString {
	return &tracepb.TruncatableString{
		Value: fmt.Sprintf("test-span-%d-%d", requestNum, index),
	}
}

type nopSender struct{}

func newNopSender() *nopSender {
	return &nopSender{}
}

func (ts *nopSender) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return nil
}

type testSender struct {
	reqChan             chan consumerdata.TraceData
	batchesReceived     int
	spansReceived       int
	spansReceivedByName map[string]*tracepb.Span
}

func newTestSender() *testSender {
	return &testSender{
		reqChan:             make(chan consumerdata.TraceData, 100),
		spansReceivedByName: make(map[string]*tracepb.Span),
	}
}

func (ts *testSender) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ts.reqChan <- td
	return nil
}

func (ts *testSender) waitFor(spans int, timeout time.Duration) chan error {
	errorCn := make(chan error)
	go func() {
		for {
			select {
			case request := <-ts.reqChan:
				for _, span := range request.Spans {
					ts.spansReceivedByName[span.Name.Value] = span
				}
				ts.batchesReceived = ts.batchesReceived + 1
				ts.spansReceived = ts.spansReceived + len(request.Spans)
				if ts.spansReceived == spans {
					errorCn <- nil
				}
			case <-time.After(timeout):
				errorCn <- fmt.Errorf("timed out waiting for spans")
			}
		}
	}()
	return errorCn
}
