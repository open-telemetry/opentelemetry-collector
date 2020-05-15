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
	"errors"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	otlpmetriccol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"
	otlptracecol "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/config/configgrpc"
)

type exporterImp struct {
	// Input configuration.
	config *Config

	// Prepared dial options.
	dialOpts []grpc.DialOption

	stopOnce sync.Once

	// gRPC clients and connection.
	traceExporter  otlptracecol.TraceServiceClient
	metricExporter otlpmetriccol.MetricsServiceClient
	grpcClientConn *grpc.ClientConn
}

var (
	errTimeout    = errors.New("timeout")
	errFatalError = errors.New("fatal error sending to server")
)

// Crete new exporter and start it. The exporter will begin connecting but
// this function may return before the connection is established.
func newExporter(config *Config) (*exporterImp, error) {
	e := &exporterImp{}
	e.config = config

	var err error
	e.dialOpts, err = configgrpc.GrpcSettingsToDialOptions(e.config.GRPCSettings)
	if err != nil {
		return nil, err
	}

	if e.grpcClientConn, err = grpc.Dial(e.config.GRPCSettings.Endpoint, e.dialOpts...); err != nil {
		return nil, err
	}
	e.traceExporter = otlptracecol.NewTraceServiceClient(e.grpcClientConn)
	e.metricExporter = otlpmetriccol.NewMetricsServiceClient(e.grpcClientConn)

	return e, nil
}

func (e *exporterImp) stop() error {
	var err error
	e.stopOnce.Do(func() {
		// Close the connection.
		err = e.grpcClientConn.Close()
	})
	return err
}

// Send a trace or metrics request to the server. "perform" function is expected to make
// the actual gRPC unary call that sends the request. This function implements the
// common OTLP logic around request handling such as retries and throttling.
func (e *exporterImp) exportRequest(ctx context.Context, perform func(ctx context.Context) error) error {

	expBackoff := backoff.NewExponentialBackOff()

	// Spend max 15 mins on this operation. This is just a reasonable number that
	// gives plenty of time for typical quick transient errors to resolve.
	expBackoff.MaxElapsedTime = time.Minute * 15

	for {
		// Send to server.
		err := perform(ctx)

		if err == nil {
			// Request is successful, we are done.
			return nil
		}

		// We have an error, check gRPC status code.

		status := status.Convert(err)

		statusCode := status.Code()
		if statusCode == codes.OK {
			// Not really an error, still success.
			return nil
		}

		// Now, this is this a real error.

		if !shouldRetry(statusCode) {
			// It is not a retryable error, we should not retry.
			return errFatalError
		}

		// Need to retry.

		// Check if server returned throttling information.
		waitDuration := getThrottleDuration(status)
		if waitDuration == 0 {
			// No explicit throttle duration. Use exponential backoff strategy.
			waitDuration = expBackoff.NextBackOff()
			if waitDuration == backoff.Stop {
				// We run out of max time allocated to this operation.
				return errTimeout
			}
		}

		// Wait until one of the conditions below triggers.
		select {
		case <-ctx.Done():
			// This request is cancelled or timed out.
			return errTimeout

		case <-time.After(waitDuration):
			// Time to try again.
		}
	}
}

func (e *exporterImp) exportTrace(ctx context.Context, request *otlptracecol.ExportTraceServiceRequest) error {
	return e.exportRequest(ctx, func(ctx context.Context) error {
		_, err := e.traceExporter.Export(ctx, request, grpc.WaitForReady(e.config.WaitForReady))
		return err
	})
}

func (e *exporterImp) exportMetrics(ctx context.Context, request *otlpmetriccol.ExportMetricsServiceRequest) error {
	return e.exportRequest(ctx, func(ctx context.Context) error {
		_, err := e.metricExporter.Export(ctx, request, grpc.WaitForReady(e.config.WaitForReady))
		return err
	})
}

func shouldRetry(code codes.Code) bool {
	switch code {
	case codes.OK:
		// Success. This function should not be called for this code, the best we
		// can do is tell the caller not to retry.
		return false

	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true

	case codes.Unknown,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.FailedPrecondition,
		codes.Unimplemented,
		codes.Internal:
		// These are fatal errors, don't retry.
		return false

	default:
		// Don't retry on unknown codes.
		return false
	}
}

func getThrottleDuration(status *status.Status) time.Duration {
	// See if throttling information is available.
	for _, detail := range status.Details() {
		switch t := detail.(type) {
		case *errdetails.RetryInfo:
			if t.RetryDelay.Seconds > 0 || t.RetryDelay.Nanos > 0 {
				// We are throttled. Wait before retrying as requested by the server.
				return time.Duration(t.RetryDelay.Seconds)*time.Second + time.Duration(t.RetryDelay.Nanos)*time.Nanosecond
			}
			return 0
		}
	}
	return 0
}
