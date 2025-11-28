// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

var _ plogotlp.GRPCServer = &logsServer{}

type logsServer struct {
	plogotlp.UnimplementedGRPCServer

	exportError error
}

func (r *logsServer) Export(_ context.Context, _ plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	return plogotlp.NewExportResponse(), r.exportError
}

func TestGRPCToGRPC(t *testing.T) {
	// gRPC supports 17 different status codes.
	// Source: https://github.com/grpc/grpc/blob/41788c90bc66caf29f28ef808d066db806389792/doc/statuscodes.md
	for i := range uint32(16) {
		s := status.New(codes.Code(i), "Testing error")
		t.Run("Code "+s.Code().String(), func(t *testing.T) {
			e := createGRPCExporter(t, s)
			assertOnGRPCCode(t, e, s)
		})
	}
}

func TestHTTPToGRPC(t *testing.T) {
	testCases := []struct {
		grpc codes.Code
		http int
	}{
		{codes.OK, http.StatusOK},
		{codes.Canceled, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusServiceUnavailable},
		{codes.Aborted, http.StatusServiceUnavailable},
		{codes.OutOfRange, http.StatusServiceUnavailable},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DataLoss, http.StatusServiceUnavailable},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.Unimplemented, http.StatusNotFound},
	}

	for _, tt := range testCases {
		s := status.New(tt.grpc, "Testing error")
		t.Run("Code "+s.Code().String(), func(t *testing.T) {
			e := createGRPCExporter(t, s)
			assertOnHTTPCode(t, e, tt.http)
		})
	}
}

func TestGRPCToHTTP(t *testing.T) {
	testCases := []struct {
		http int
		grpc codes.Code
	}{
		{http.StatusOK, codes.OK},
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusUnauthorized, codes.Unauthenticated},
		{http.StatusForbidden, codes.PermissionDenied},
		{http.StatusNotFound, codes.Unimplemented},
		{http.StatusTooManyRequests, codes.ResourceExhausted},
		{http.StatusBadGateway, codes.Unavailable},
		{http.StatusServiceUnavailable, codes.Unavailable},
		{http.StatusGatewayTimeout, codes.Unavailable},
	}

	for _, tt := range testCases {
		s := status.New(tt.grpc, "Testing error")
		t.Run("Code "+s.Code().String(), func(t *testing.T) {
			e := createHTTPExporter(t, tt.http)
			assertOnGRPCCode(t, e, s)
		})
	}
}

func TestHTTPToHTTP(t *testing.T) {
	testCases := []struct {
		code    int
		mapping int
	}{
		{code: http.StatusOK},
		{code: http.StatusServiceUnavailable},
		{code: http.StatusTooManyRequests},
		{code: http.StatusBadRequest},
		{code: http.StatusUnauthorized},
		{code: http.StatusForbidden},
		{code: http.StatusNotFound},
		{code: http.StatusInternalServerError},
		// Mappings won't be necessary once the OTLP/HTTP Exporter returns consumererror.Error types.
		{code: http.StatusBadGateway, mapping: http.StatusServiceUnavailable},
		{code: http.StatusGatewayTimeout, mapping: http.StatusServiceUnavailable},
		{code: http.StatusTeapot, mapping: http.StatusInternalServerError},
		{code: http.StatusConflict, mapping: http.StatusInternalServerError},
	}

	for _, tt := range testCases {
		t.Run("Code "+strconv.Itoa(tt.code), func(t *testing.T) {
			e := createHTTPExporter(t, tt.code)
			code := tt.code
			if tt.mapping != 0 {
				code = tt.mapping
			}
			assertOnHTTPCode(t, e, code)
		})
	}
}

func createGRPCExporter(t *testing.T, s *status.Status) consumer.Logs {
	t.Helper()

	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	srv := grpc.NewServer()
	rcv := &logsServer{
		exportError: s.Err(),
	}

	plogotlp.RegisterGRPCServer(srv, rcv)
	go func() {
		assert.NoError(t, srv.Serve(ln))
	}()
	t.Cleanup(func() {
		srv.Stop()
	})

	f := otlpexporter.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlpexporter.Config)
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	cfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: ln.Addr().String(),
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}
	e, err := f.CreateLogs(context.Background(), exportertest.NewNopSettings(component.MustNewType("otlp")), cfg)
	require.NoError(t, err)
	err = e.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, e.Shutdown(context.Background()))
	})

	return e
}

func createHTTPExporter(t *testing.T, code int) consumer.Logs {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(code)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
	})

	f := otlphttpexporter.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlphttpexporter.Config)
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	cfg.Encoding = otlphttpexporter.EncodingProto
	cfg.LogsEndpoint = srv.URL + "/v1/logs"
	e, err := f.CreateLogs(context.Background(), exportertest.NewNopSettings(component.MustNewType("otlphttp")), cfg)
	require.NoError(t, err)
	err = e.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, e.Shutdown(context.Background()))
	})

	return e
}

func assertOnGRPCCode(t *testing.T, l consumer.Logs, s *status.Status) {
	t.Helper()

	rf := otlpreceiver.NewFactory()
	rcfg := rf.CreateDefaultConfig().(*otlpreceiver.Config)
	rcfg.GRPC = configoptional.Some(
		configgrpc.ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  testutil.GetAvailableLocalAddress(t),
				Transport: confignet.TransportTypeTCP,
			},
		},
	)
	r, err := rf.CreateLogs(context.Background(), receiver.Settings{
		ID:                component.MustNewID("otlp"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, rcfg, l)
	require.NoError(t, err)
	err = r.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Shutdown(context.Background()))
	})

	conn, err := grpc.NewClient(rcfg.GRPC.Get().NetAddr.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	ld := testdata.GenerateLogs(2)

	acc := plogotlp.NewGRPCClient(conn)
	req := plogotlp.NewExportRequestFromLogs(ld)
	res, err := acc.Export(context.Background(), req)

	if s.Code() == codes.OK {
		require.NoError(t, err)
	} else {
		got := status.Convert(err).Code()
		require.Equal(t, s.Code(), got, "Expected code %s but got %s", s.Code().String(), got.String())
	}
	require.NotNil(t, res)
}

func assertOnHTTPCode(t *testing.T, l consumer.Logs, code int) {
	t.Helper()

	ld := testdata.GenerateLogs(2)
	protoMarshaler := &plog.ProtoMarshaler{}
	logProto, err := protoMarshaler.MarshalLogs(ld)
	require.NoError(t, err)

	rf := otlpreceiver.NewFactory()
	rcfg := rf.CreateDefaultConfig().(*otlpreceiver.Config)
	rcfg.HTTP = configoptional.Some(
		otlpreceiver.HTTPConfig{
			ServerConfig: confighttp.ServerConfig{
				Endpoint: testutil.GetAvailableLocalAddress(t),
			},
			LogsURLPath: "/v1/logs",
		},
	)
	r, err := rf.CreateLogs(context.Background(), receiver.Settings{
		ID:                component.MustNewID("otlp"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}, rcfg, l)
	require.NoError(t, err)
	err = r.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Shutdown(context.Background()))
	})

	doHTTPRequest(t, rcfg.HTTP.Get().ServerConfig.Endpoint+"/v1/logs", logProto, code)
}

func doHTTPRequest(
	t *testing.T,
	url string,
	data []byte,
	expectStatusCode int,
) []byte {
	req := createHTTPRequest(t, url, data)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.NoError(t, resp.Body.Close())

	if expectStatusCode == 0 {
		require.Equal(t, http.StatusOK, resp.StatusCode)
	} else {
		require.Equal(t, expectStatusCode, resp.StatusCode)
	}

	return respBytes
}

func createHTTPRequest(
	t *testing.T,
	url string,
	data []byte,
) *http.Request {
	buf := bytes.NewBuffer(data)

	req, err := http.NewRequest(http.MethodPost, "http://"+url, buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-protobuf")

	return req
}
