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

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"errors"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
)

type exporter struct {
	// Input configuration.
	config *Config

	// gRPC clients and connection.
	traceExporter  otlpgrpc.TracesClient
	metricExporter otlpgrpc.MetricsClient
	logExporter    otlpgrpc.LogsClient
	clientConn     *grpc.ClientConn
	metadata       metadata.MD
	callOptions    []grpc.CallOption

	settings component.TelemetrySettings
}

// Crete new exporter and start it. The exporter will begin connecting but
// this function may return before the connection is established.
func newExporter(cfg config.Exporter, settings component.TelemetrySettings) (*exporter, error) {
	oCfg := cfg.(*Config)

	if oCfg.Endpoint == "" {
		return nil, errors.New("OTLP exporter config requires an Endpoint")
	}

	return &exporter{config: oCfg, settings: settings}, nil
}

// start actually creates the gRPC connection. The client construction is deferred till this point as this
// is the only place we get hold of Extensions which are required to construct auth round tripper.
func (e *exporter) start(_ context.Context, host component.Host) (err error) {
	dialOpts, err := e.config.GRPCClientSettings.ToDialOptions(host, e.settings)
	if err != nil {
		return err
	}

	if e.clientConn, err = grpc.Dial(e.config.GRPCClientSettings.SanitizedEndpoint(), dialOpts...); err != nil {
		return err
	}

	e.traceExporter = otlpgrpc.NewTracesClient(e.clientConn)
	e.metricExporter = otlpgrpc.NewMetricsClient(e.clientConn)
	e.logExporter = otlpgrpc.NewLogsClient(e.clientConn)
	e.metadata = metadata.New(e.config.GRPCClientSettings.Headers)
	e.callOptions = []grpc.CallOption{
		grpc.WaitForReady(e.config.GRPCClientSettings.WaitForReady),
	}

	return
}

func (e *exporter) shutdown(context.Context) error {
	return e.clientConn.Close()
}

func (e *exporter) pushTraces(ctx context.Context, td pdata.Traces) error {
	req := otlpgrpc.NewTracesRequest()
	req.SetTraces(td)
	_, err := e.traceExporter.Export(e.enhanceContext(ctx), req, e.callOptions...)
	return processError(err)
}

func (e *exporter) pushMetrics(ctx context.Context, md pdata.Metrics) error {
	req := otlpgrpc.NewMetricsRequest()
	req.SetMetrics(md)
	_, err := e.metricExporter.Export(e.enhanceContext(ctx), req, e.callOptions...)
	return processError(err)
}

func (e *exporter) pushLogs(ctx context.Context, ld pdata.Logs) error {
	req := otlpgrpc.NewLogsRequest()
	req.SetLogs(ld)
	_, err := e.logExporter.Export(e.enhanceContext(ctx), req, e.callOptions...)
	return processError(err)
}

func (e *exporter) enhanceContext(ctx context.Context) context.Context {
	if e.metadata.Len() > 0 {
		return metadata.NewOutgoingContext(ctx, e.metadata)
	}
	return ctx
}

// Send a trace or metrics request to the server. "perform" function is expected to make
// the actual gRPC unary call that sends the request. This function implements the
// common OTLP logic around request handling such as retries and throttling.
func processError(err error) error {
	if err == nil {
		// Request is successful, we are done.
		return nil
	}

	// We have an error, check gRPC status code.

	st := status.Convert(err)
	if st.Code() == codes.OK {
		// Not really an error, still success.
		return nil
	}

	// Now, this is this a real error.

	if !shouldRetry(st.Code()) {
		// It is not a retryable error, we should not retry.
		return consumererror.NewPermanent(err)
	}

	// Need to retry.

	// Check if server returned throttling information.
	throttleDuration := getThrottleDuration(st)
	if throttleDuration != 0 {
		return exporterhelper.NewThrottleRetry(err, throttleDuration)
	}

	return err
}

func shouldRetry(code codes.Code) bool {
	switch code {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		// These are retryable errors.
		return true
	}
	// Don't retry on any other code.
	return false
}

func getThrottleDuration(status *status.Status) time.Duration {
	// See if throttling information is available.
	for _, detail := range status.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			if t.RetryDelay.Seconds > 0 || t.RetryDelay.Nanos > 0 {
				// We are throttled. Wait before retrying as requested by the server.
				return time.Duration(t.RetryDelay.Seconds)*time.Second + time.Duration(t.RetryDelay.Nanos)*time.Nanosecond
			}
			return 0
		}
	}
	return 0
}
