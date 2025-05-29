// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tinybirdexporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewExporter(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Endpoint:   "http://localhost:8080",
				Token:      "test-token",
				DataSource: "test-datasource",
			},
			wantErr: false,
		},
		{
			name: "invalid endpoint",
			config: &Config{
				Endpoint:   "invalid-url",
				Token:      "test-token",
				DataSource: "test-datasource",
			},
			wantErr: true,
		},
		{
			name: "missing token",
			config: &Config{
				Endpoint:   "http://localhost:8080",
				DataSource: "test-datasource",
			},
			wantErr: true,
		},
		{
			name: "missing datasource",
			config: &Config{
				Endpoint: "http://localhost:8080",
				Token:    "test-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := newExporter(tt.config, exportertest.NewNopSettings(component.MustNewType("tinybird")))
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exp)
			}
		})
	}
}

func TestExportTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v0/events?name=traces", r.URL.Path)
		assert.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Endpoint:   server.URL,
		Token:      "test-token",
		DataSource: "test-datasource",
	}

	exp, err := newExporter(config, exportertest.NewNopSettings(component.MustNewType("tinybird")))
	require.NoError(t, err)
	require.NoError(t, exp.start(context.Background(), componenttest.NewNopHost()))

	traces := ptrace.NewTraces()
	require.NoError(t, exp.pushTraces(context.Background(), traces))
}

func TestExportMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v0/events?name=metrics", r.URL.Path)
		assert.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Endpoint:   server.URL,
		Token:      "test-token",
		DataSource: "test-datasource",
	}

	exp, err := newExporter(config, exportertest.NewNopSettings(component.MustNewType("tinybird")))
	require.NoError(t, err)
	require.NoError(t, exp.start(context.Background(), componenttest.NewNopHost()))

	metrics := pmetric.NewMetrics()
	require.NoError(t, exp.pushMetrics(context.Background(), metrics))
}

func TestExportLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v0/events?name=logs", r.URL.Path)
		assert.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Endpoint:   server.URL,
		Token:      "test-token",
		DataSource: "test-datasource",
	}

	exp, err := newExporter(config, exportertest.NewNopSettings(component.MustNewType("tinybird")))
	require.NoError(t, err)
	require.NoError(t, exp.start(context.Background(), componenttest.NewNopHost()))

	logs := plog.NewLogs()
	require.NoError(t, exp.pushLogs(context.Background(), logs))
}

func TestExportErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		headers        map[string]string
		wantErr        bool
	}{
		{
			name:           "success",
			responseStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "throttled",
			responseStatus: http.StatusTooManyRequests,
			headers:        map[string]string{"Retry-After": "30"},
			wantErr:        true,
		},
		{
			name:           "service unavailable",
			responseStatus: http.StatusServiceUnavailable,
			wantErr:        true,
		},
		{
			name:           "permanent error",
			responseStatus: http.StatusBadRequest,
			responseBody:   "invalid request",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != "" {
					w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			// Wait for server to be ready
			_, err := http.Get(server.URL)
			require.NoError(t, err)

			config := &Config{
				Endpoint:   server.URL,
				Token:      "test-token",
				DataSource: "test-datasource",
			}

			exp, err := newExporter(config, exportertest.NewNopSettings(component.MustNewType("tinybird")))
			require.NoError(t, err)
			require.NoError(t, exp.start(context.Background(), componenttest.NewNopHost()))

			traces := ptrace.NewTraces()
			rs := traces.ResourceSpans().AppendEmpty()
			ss := rs.ScopeSpans().AppendEmpty()
			span := ss.Spans().AppendEmpty()
			span.SetName("test-span")
			err = exp.pushTraces(context.Background(), traces)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
