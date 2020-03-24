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

package otlpexporter

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"
	"unsafe"

	otlpmetriccol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"
	otlptracecol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Connection implementation is borrowed and adapted from OpenTelemetry Go SDK implementation:
// https://github.com/open-telemetry/opentelemetry-go/blob/e0406dd3eb7aa826aba19287e8a14ca016bf8578/exporters/otlp/connection.go#L15

func (e *exporterImp) saveLastConnectError(err error) {
	var errPtr *error
	if err != nil {
		errPtr = &err
	}
	atomic.StorePointer(&e.lastConnectErrPtr, unsafe.Pointer(errPtr))
}

func (e *exporterImp) setStateDisconnected(err error) {
	e.saveLastConnectError(err)
	select {
	case e.disconnectedCh <- true:
	default:
	}
}

func (e *exporterImp) setStateConnected() {
	e.saveLastConnectError(nil)
}

const defaultConnReattemptPeriod = 10 * time.Second

func (e *exporterImp) indefiniteBackgroundConnection() {
	defer func() {
		e.backgroundConnectionDoneCh <- true
	}()

	connReattemptPeriod := e.config.ReconnectionDelay
	if connReattemptPeriod <= 0 {
		connReattemptPeriod = defaultConnReattemptPeriod
	}

	// No strong seeding required, nano time can
	// already help with pseudo uniqueness.
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63n(1024)))

	// maxJitterNanos: 70% of the connectionReattemptPeriod
	maxJitterNanos := int64(0.7 * float64(connReattemptPeriod))

	for {
		// Otherwise these will be the normal scenarios to enable
		// reconnection if we trip out.
		// 1. If we've stopped, return entirely
		// 2. Otherwise block until we are disconnected, and
		//    then retry connecting
		select {
		case <-e.stopCh:
			return

		case <-e.disconnectedCh:
			// Normal scenario that we'll wait for
		}

		if err := e.connect(); err == nil {
			e.setStateConnected()
		} else {
			e.setStateDisconnected(err)
		}

		// Apply some jitter to avoid lockstep retrials of other
		// collector-exporterImps. Lockstep retrials could result in an
		// innocent DDOS, by clogging the machine's resources and network.
		jitter := time.Duration(rng.Int63n(maxJitterNanos))
		select {
		case <-e.stopCh:
			return
		case <-time.After(connReattemptPeriod + jitter):
		}
	}
}

func (e *exporterImp) connect() error {
	cc, err := e.dialToCollector()
	if err != nil {
		return err
	}
	return e.enableConnections(cc)
}

func (e *exporterImp) enableConnections(cc *grpc.ClientConn) error {
	e.mutex.RLock()
	started := e.started
	e.mutex.RUnlock()

	if !started {
		return errNotStarted
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()
	// If previous clientConn is same as the current then just return.
	// This doesn't happen right now as this func is only called with new ClientConn.
	// It is more about future-proofing.
	if e.grpcClientConn == cc {
		return nil
	}
	// If the previous clientConn was non-nil, close it
	if e.grpcClientConn != nil {
		_ = e.grpcClientConn.Close()
	}
	e.grpcClientConn = cc
	e.traceExporter = otlptracecol.NewTraceServiceClient(cc)
	e.metricExporter = otlpmetriccol.NewMetricsServiceClient(cc)

	return nil
}
func (e *exporterImp) dialToCollector() (*grpc.ClientConn, error) {
	ctx := context.Background()
	if len(e.config.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(e.config.Headers))
	}
	return grpc.DialContext(ctx, e.config.Endpoint, e.dialOpts...)
}
