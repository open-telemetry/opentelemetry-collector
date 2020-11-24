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

package jaegerexporter

import (
	"context"
	"net"
	"path"
	"sync"
	"testing"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	tracev1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestNew(t *testing.T) {
	type args struct {
		config Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "createExporter",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     nil,
						Endpoint:    "foo.bar",
						Compression: "",
						TLSSetting: configtls.TLSClientSetting{
							Insecure: true,
						},
						Keepalive: nil,
					},
				},
			},
		},
		{
			name: "createExporterWithHeaders",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     map[string]string{"extra-header": "header-value"},
						Endpoint:    "foo.bar",
						Compression: "",
						Keepalive:   nil,
					},
				},
			},
		},
		{
			name: "createBasicSecureExporter",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     nil,
						Endpoint:    "foo.bar",
						Compression: "",
						Keepalive:   nil,
					},
				},
			},
		},
		{
			name: "createSecureExporterWithClientTLS",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     nil,
						Endpoint:    "foo.bar",
						Compression: "",
						TLSSetting: configtls.TLSClientSetting{
							TLSSetting: configtls.TLSSetting{
								CAFile: "testdata/test_cert.pem",
							},
							Insecure: false,
						},
						Keepalive: nil,
					},
				},
			},
		},
		{
			name: "createSecureExporterWithKeepAlive",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     nil,
						Endpoint:    "foo.bar",
						Compression: "",
						TLSSetting: configtls.TLSClientSetting{
							TLSSetting: configtls.TLSSetting{
								CAFile: "testdata/test_cert.pem",
							},
							Insecure:   false,
							ServerName: "",
						},
						Keepalive: &configgrpc.KeepaliveClientConfig{
							Time:                0,
							Timeout:             0,
							PermitWithoutStream: false,
						},
					},
				},
			},
		},
		{
			name: "createSecureExporterWithMissingFile",
			args: args{
				config: Config{
					GRPCClientSettings: configgrpc.GRPCClientSettings{
						Headers:     nil,
						Endpoint:    "foo.bar",
						Compression: "",
						TLSSetting: configtls.TLSClientSetting{
							TLSSetting: configtls.TLSSetting{
								CAFile: "testdata/test_cert_missing.pem",
							},
							Insecure: false,
						},
						Keepalive: nil,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTraceExporter(&tt.args.config, zap.NewNop())
			if (err != nil) != tt.wantErr {
				t.Errorf("newTraceExporter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				return
			}

			// This is expected to fail.
			err = got.ConsumeTraces(context.Background(), testdata.GenerateTraceDataNoLibraries())
			assert.Error(t, err)
		})
	}
}

// CA key and cert
// openssl req -new -nodes -x509 -days 9650 -keyout ca.key -out ca.crt -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"
// Server key and cert
// openssl genrsa -des3 -out server.key 1024
// openssl req -new -key server.key -out server.csr -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"
// openssl x509 -req -days 9650 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out server.crt
// Client key and cert
// openssl genrsa -des3 -out client.key 1024
// openssl req -new -key client.key -out client.csr -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"
// openssl x509 -req -days 9650 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out client.crt
// Remove passphrase
// openssl rsa -in server.key -out temp.key && rm server.key && mv temp.key server.key
// openssl rsa -in client.key -out temp.key && rm client.key && mv temp.key client.key
func TestMutualTLS(t *testing.T) {
	caPath := path.Join(".", "testdata", "ca.crt")
	serverCertPath := path.Join(".", "testdata", "server.crt")
	serverKeyPath := path.Join(".", "testdata", "server.key")
	clientCertPath := path.Join(".", "testdata", "client.crt")
	clientKeyPath := path.Join(".", "testdata", "client.key")

	// start gRPC Jaeger server
	tlsCfgOpts := configtls.TLSServerSetting{
		TLSSetting: configtls.TLSSetting{
			CertFile: serverCertPath,
			KeyFile:  serverKeyPath,
		},
		ClientCAFile: caPath,
	}
	tlsCfg, err := tlsCfgOpts.LoadTLSConfig()
	require.NoError(t, err)
	spanHandler := &mockSpanHandler{}
	server, serverAddr := initializeGRPCTestServer(t, func(server *grpc.Server) {
		api_v2.RegisterCollectorServiceServer(server, spanHandler)
	}, grpc.Creds(credentials.NewTLS(tlsCfg)))
	defer server.GracefulStop()

	// Create gRPC trace exporter
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Disable queuing to ensure that we execute the request when calling ConsumeTraces
	// otherwise we will have to wait.
	cfg.QueueSettings.Enabled = false
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: serverAddr.String(),
		TLSSetting: configtls.TLSClientSetting{
			TLSSetting: configtls.TLSSetting{
				CAFile:   caPath,
				CertFile: clientCertPath,
				KeyFile:  clientKeyPath,
			},
			Insecure:   false,
			ServerName: "localhost",
		},
	}
	exporter, err := factory.CreateTracesExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	require.NoError(t, err)
	err = exporter.Start(context.Background(), nil)
	require.NoError(t, err)
	defer exporter.Shutdown(context.Background())

	traceID := data.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	spanID := data.NewSpanID([8]byte{0, 1, 2, 3, 4, 5, 6, 7})
	traces := pdata.TracesFromOtlp([]*tracev1.ResourceSpans{
		{InstrumentationLibrarySpans: []*tracev1.InstrumentationLibrarySpans{{Spans: []*tracev1.Span{{TraceId: traceID, SpanId: spanID}}}}},
	})
	err = exporter.ConsumeTraces(context.Background(), traces)
	require.NoError(t, err)
	requestes := spanHandler.getRequests()
	assert.Equal(t, 1, len(requestes))
	tidBytes := traceID.Bytes()
	jTraceID, err := model.TraceIDFromBytes(tidBytes[:])
	require.NoError(t, err)
	require.Len(t, requestes, 1)
	require.Len(t, requestes[0].GetBatch().Spans, 1)
	assert.Equal(t, jTraceID, requestes[0].GetBatch().Spans[0].TraceID)
}

func initializeGRPCTestServer(t *testing.T, beforeServe func(server *grpc.Server), opts ...grpc.ServerOption) (*grpc.Server, net.Addr) {
	server := grpc.NewServer(opts...)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	beforeServe(server)
	go func() {
		require.NoError(t, server.Serve(lis))
	}()
	return server, lis.Addr()
}

type mockSpanHandler struct {
	mux      sync.Mutex
	requests []*api_v2.PostSpansRequest
}

func (h *mockSpanHandler) getRequests() []*api_v2.PostSpansRequest {
	h.mux.Lock()
	defer h.mux.Unlock()
	return h.requests
}

func (h *mockSpanHandler) PostSpans(_ context.Context, r *api_v2.PostSpansRequest) (*api_v2.PostSpansResponse, error) {
	h.mux.Lock()
	defer h.mux.Unlock()
	h.requests = append(h.requests, r)
	return &api_v2.PostSpansResponse{}, nil
}
