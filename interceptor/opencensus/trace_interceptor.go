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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/interceptor"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/spanreceiver"
)

// New will create a handle that runs an OCInterceptor at the provided address
func NewTraceInterceptor(port int, opts ...OCOption) (interceptor.TraceInterceptor, error) {
	return &ocInterceptorHandler{srvPort: port, interceptorOptions: opts}, nil
}

func NewTraceInterceptorOnDefaultPort(opts ...OCOption) (interceptor.TraceInterceptor, error) {
	return &ocInterceptorHandler{srvPort: defaultOCInterceptorPort, interceptorOptions: opts}, nil
}

type ocInterceptorHandler struct {
	mu      sync.RWMutex
	srvPort int

	interceptorOptions []OCOption

	ln         net.Listener
	grpcServer *grpc.Server

	stopOnce  sync.Once
	startOnce sync.Once
}

var _ interceptor.TraceInterceptor = (*ocInterceptorHandler)(nil)

var errAlreadyStarted = errors.New("already started")

// StartTraceInterception starts a gRPC server with an OpenCensus interceptor running
func (ocih *ocInterceptorHandler) StartTraceInterception(ctx context.Context, sr spanreceiver.SpanReceiver) error {
	var err error = errAlreadyStarted
	ocih.startOnce.Do(func() {
		err = ocih.startInternal(ctx, sr)
	})

	return err
}

const defaultOCInterceptorPort = 55678

func (ocih *ocInterceptorHandler) startInternal(ctx context.Context, sr spanreceiver.SpanReceiver) error {
	ocih.mu.Lock()
	defer ocih.mu.Unlock()

	oci, err := New(sr, ocih.interceptorOptions...)
	if err != nil {
		return err
	}

	port := ocih.srvPort
	if port <= 0 {
		port = defaultOCInterceptorPort
	}

	addr := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := internal.GRPCServerWithObservabilityEnabled()

	agenttracepb.RegisterTraceServiceServer(srv, oci)
	go func() {
		_ = srv.Serve(ln)
	}()
	ocih.ln = ln
	ocih.grpcServer = srv

	return err
}

var errAlreadyStopped = errors.New("already stopped")

func (ocih *ocInterceptorHandler) StopTraceInterception(ctx context.Context) error {
	var err error = errAlreadyStopped
	ocih.stopOnce.Do(func() {
		ocih.mu.Lock()
		defer ocih.mu.Unlock()

		// TODO: (@odeke-em) should we instead to a graceful stop instead of a sudden stop?
		// A graceful stop takes time to terminate.
		ocih.grpcServer.Stop()
		err = nil
	})
	return err
}
