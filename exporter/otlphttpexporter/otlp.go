// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter // import "go.opentelemetry.io/collector/exporter/otlphttpexporter"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

type baseExporter struct {
	// Input configuration.
	config     *Config
	client     *http.Client
	tracesURL  string
	metricsURL string
	logsURL    string
	logger     *zap.Logger
	settings   component.TelemetrySettings
	// Default user-agent header.
	userAgent string
}

const (
	headerRetryAfter         = "Retry-After"
	maxHTTPResponseReadBytes = 64 * 1024
)

// Create new exporter.
func newExporter(cfg component.Config, set exporter.CreateSettings) (*baseExporter, error) {
	oCfg := cfg.(*Config)

	if oCfg.Endpoint != "" {
		_, err := url.Parse(oCfg.Endpoint)
		if err != nil {
			return nil, errors.New("endpoint must be a valid URL")
		}
	}

	userAgent := fmt.Sprintf("%s/%s (%s/%s)",
		set.BuildInfo.Description, set.BuildInfo.Version, runtime.GOOS, runtime.GOARCH)

	// client construction is deferred to start
	return &baseExporter{
		config:    oCfg,
		logger:    set.Logger,
		userAgent: userAgent,
		settings:  set.TelemetrySettings,
	}, nil
}

// start actually creates the HTTP client. The client construction is deferred till this point as this
// is the only place we get hold of Extensions which are required to construct auth round tripper.
func (e *baseExporter) start(_ context.Context, host component.Host) error {
	client, err := e.config.HTTPClientSettings.ToClient(host, e.settings)
	if err != nil {
		return err
	}
	e.client = client
	return nil
}

func (e *baseExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	tr := ptraceotlp.NewExportRequestFromTraces(td)
	request, err := tr.MarshalProto()
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	return e.export(ctx, e.tracesURL, request, tracesPartialSuccessHandler)
}

func (e *baseExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	tr := pmetricotlp.NewExportRequestFromMetrics(md)
	request, err := tr.MarshalProto()
	if err != nil {
		return consumererror.NewPermanent(err)
	}
	return e.export(ctx, e.metricsURL, request, metricsPartialSuccessHandler)
}

func (e *baseExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	tr := plogotlp.NewExportRequestFromLogs(ld)
	request, err := tr.MarshalProto()
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	return e.export(ctx, e.logsURL, request, logsPartialSuccessHandler)
}

func (e *baseExporter) export(ctx context.Context, url string, request []byte, partialSuccessHandler partialSuccessHandler) error {
	e.logger.Debug("Preparing to make HTTP request", zap.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(request))
	if err != nil {
		return consumererror.NewPermanent(err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make an HTTP request: %w", err)
	}

	defer func() {
		// Discard any remaining response body when we are done reading.
		io.CopyN(io.Discard, resp.Body, maxHTTPResponseReadBytes) // nolint:errcheck
		resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		if err := handlePartialSuccessResponse(resp, partialSuccessHandler); err != nil {
			return err
		}

		// Request is successful.
		return nil
	}

	respStatus := readResponseStatus(resp)

	// Format the error message. Use the status if it is present in the response.
	var formattedErr error
	if respStatus != nil {
		formattedErr = fmt.Errorf(
			"error exporting items, request to %s responded with HTTP Status Code %d, Message=%s, Details=%v",
			url, resp.StatusCode, respStatus.Message, respStatus.Details)
	} else {
		formattedErr = fmt.Errorf(
			"error exporting items, request to %s responded with HTTP Status Code %d",
			url, resp.StatusCode)
	}

	if isRetryableStatusCode(resp.StatusCode) {
		// A retry duration of 0 seconds will trigger the default backoff policy
		// of our caller (retry handler).
		retryAfter := 0

		// Check if the server is overwhelmed.
		// See spec https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#otlphttp-throttling
		isThrottleError := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable
		if val := resp.Header.Get(headerRetryAfter); isThrottleError && val != "" {
			if seconds, err2 := strconv.Atoi(val); err2 == nil {
				retryAfter = seconds
			}
		}

		return exporterhelper.NewThrottleRetry(formattedErr, time.Duration(retryAfter)*time.Second)
	}

	return consumererror.NewPermanent(formattedErr)
}

// Determine if the status code is retryable according to the specification.
// For more, see https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#failures-1
func isRetryableStatusCode(code int) bool {
	switch code {
	case http.StatusTooManyRequests:
		return true
	case http.StatusBadGateway:
		return true
	case http.StatusServiceUnavailable:
		return true
	case http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func readResponseBody(body io.ReadCloser) ([]byte, error) {
	// Read the maximum number of bytes allowed in a request. This avoids
	// issues with missing or invalid Content-Length headers.
	protoBytes := make([]byte, maxHTTPResponseReadBytes)
	n, err := io.ReadFull(body, protoBytes)

	// No bytes read and an EOF error indicates there is no body to read.
	if n == 0 && (err == nil || errors.Is(err, io.EOF)) {
		return nil, nil
	}

	// io.ReadFull will return io.ErrorUnexpectedEOF in most cases since there
	// will usually be a mismatch between the length of the byte slice and the
	// size of the body, so we ignore that error.
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}

	// The pdata unmarshaling methods check for the length of the slice
	// when unmarshaling it, so we have to trim down the length to the
	// actual size of the data.
	return protoBytes[:n], nil
}

// Read the response and decode the status.Status from the body.
// Returns nil if the response is empty or cannot be decoded.
func readResponseStatus(resp *http.Response) *status.Status {
	var respStatus *status.Status
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		// Request failed. Read the body. OTLP spec says:
		// "Response body for all HTTP 4xx and HTTP 5xx responses MUST be a
		// Protobuf-encoded Status message that describes the problem."
		respBytes, err := readResponseBody(resp.Body)

		if err != nil {
			return nil
		}

		// Decode it as Status struct. See https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#failures
		respStatus = &status.Status{}
		err = proto.Unmarshal(respBytes, respStatus)
		if err != nil {
			return nil
		}
	}

	return respStatus
}

func handlePartialSuccessResponse(resp *http.Response, partialSuccessHandler partialSuccessHandler) error {
	bodyBytes, err := readResponseBody(resp.Body)

	if err != nil {
		return err
	}

	return partialSuccessHandler(bodyBytes)
}

type partialSuccessHandler func(protoBytes []byte) error

func tracesPartialSuccessHandler(protoBytes []byte) error {
	exportResponse := ptraceotlp.NewExportResponse()
	err := exportResponse.UnmarshalProto(protoBytes)
	if err != nil {
		return err
	}
	partialSuccess := exportResponse.PartialSuccess()
	if !(partialSuccess.ErrorMessage() == "" && partialSuccess.RejectedSpans() == 0) {
		return consumererror.NewPermanent(fmt.Errorf("OTLP partial success: %s (%d rejected)", partialSuccess.ErrorMessage(), partialSuccess.RejectedSpans()))
	}
	return nil
}

func metricsPartialSuccessHandler(protoBytes []byte) error {
	exportResponse := pmetricotlp.NewExportResponse()
	err := exportResponse.UnmarshalProto(protoBytes)
	if err != nil {
		return err
	}
	partialSuccess := exportResponse.PartialSuccess()
	if !(partialSuccess.ErrorMessage() == "" && partialSuccess.RejectedDataPoints() == 0) {
		return consumererror.NewPermanent(fmt.Errorf("OTLP partial success: %s (%d rejected)", partialSuccess.ErrorMessage(), partialSuccess.RejectedDataPoints()))
	}
	return nil
}

func logsPartialSuccessHandler(protoBytes []byte) error {
	exportResponse := plogotlp.NewExportResponse()
	err := exportResponse.UnmarshalProto(protoBytes)
	if err != nil {
		return err
	}
	partialSuccess := exportResponse.PartialSuccess()
	if !(partialSuccess.ErrorMessage() == "" && partialSuccess.RejectedLogRecords() == 0) {
		return consumererror.NewPermanent(fmt.Errorf("OTLP partial success: %s (%d rejected)", partialSuccess.ErrorMessage(), partialSuccess.RejectedLogRecords()))
	}
	return nil
}
