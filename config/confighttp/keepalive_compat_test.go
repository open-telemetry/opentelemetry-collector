// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
)

// This file encodes compatibility requirements for the keepalive config
// migration, derived from how opentelemetry-collector-contrib components use
// the deprecated fields today. Each test asserts the behavior the pre-migration
// code (main) produces for the same inputs; a failure here means a real,
// user-visible behavior change in the named downstream components.

func compatClientTransport(t *testing.T, cc ClientConfig) *http.Transport {
	t.Helper()
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil
	client, err := cc.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok, "transport is %T", client.Transport)
	return transport
}

// compatServeAndGet builds the server from sc, serves a single request against
// it, and returns the response so callers can observe connection handling.
func compatServeAndGet(t *testing.T, sc ServerConfig) *http.Response {
	t.Helper()
	srv, err := sc.ToServer(t.Context(), nil, componenttest.NewNopTelemetrySettings(),
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	require.NoError(t, err)

	ln, err := sc.ToListener(t.Context())
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ln)
	}()
	t.Cleanup(func() {
		assert.NoError(t, srv.Close())
		<-done
	})

	resp, err := http.Get(fmt.Sprintf("http://%s/", ln.Addr()))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp
}

// A user setting a subset of the deprecated fields must keep the defaults for
// the settings they did not mention. On main, `idle_conn_timeout: 30s` alone
// leaves MaxIdleConns at its default of 100.
func TestKeepaliveCompatPartialDeprecatedFields(t *testing.T) {
	cfg := NewDefaultClientConfig()
	conf := confmap.NewFromStringMap(map[string]any{"idle_conn_timeout": "30s"})
	require.NoError(t, conf.Unmarshal(&cfg))

	transport := compatClientTransport(t, cfg)
	assert.Equal(t, 30*time.Second, transport.IdleConnTimeout)
	assert.Equal(t, 100, transport.MaxIdleConns)
}

// Factories zero the deprecated fields to mean "unlimited idle connections, no
// idle timeout": prometheusreceiver, nsxtreceiver, httpcheckreceiver,
// huaweicloudcesreceiver, haproxyreceiver, awsecscontainermetricsreceiver, and
// awscontainerinsightreceiver in contrib. Zero must remain meaningful.
func TestKeepaliveCompatFactoryZeroedDeprecatedFields(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.MaxIdleConns = 0
	cfg.IdleConnTimeout = 0
	require.NoError(t, confmap.NewFromStringMap(map[string]any{}).Unmarshal(&cfg))

	transport := compatClientTransport(t, cfg)
	assert.Equal(t, 0, transport.MaxIdleConns)
	assert.Equal(t, time.Duration(0), transport.IdleConnTimeout)
}

// Factories also set non-zero values on the deprecated fields, e.g.
// signalfxexporter sets MaxIdleConns and MaxIdleConnsPerHost to 30000.
func TestKeepaliveCompatFactoryCustomDeprecatedFields(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.MaxIdleConns = 30000
	cfg.MaxIdleConnsPerHost = 30000
	require.NoError(t, confmap.NewFromStringMap(map[string]any{}).Unmarshal(&cfg))

	transport := compatClientTransport(t, cfg)
	assert.Equal(t, 30000, transport.MaxIdleConns)
	assert.Equal(t, 30000, transport.MaxIdleConnsPerHost)
}

// Deprecated-field values set programmatically by a factory are not "the user
// set both": a user must be able to adopt the keepalive section on a component
// whose factory customizes the deprecated fields without hitting the
// mixed-config error.
func TestKeepaliveCompatFactoryFieldsDoNotConflictWithNewSection(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.MaxIdleConns = 30000
	cfg.MaxIdleConnsPerHost = 30000
	conf := confmap.NewFromStringMap(map[string]any{
		"keepalive": map[string]any{"idle_conn_timeout": "5m"},
	})
	require.NoError(t, conf.Unmarshal(&cfg))
}

// Deprecation warnings must be triggered only by configuration the user wrote,
// never by factory-set values: users of a component whose factory customizes
// the deprecated fields (signalfxexporter) can't act on the warning.
func TestKeepaliveCompatNoWarningsForFactoryFields(t *testing.T) {
	cfg := NewDefaultClientConfig()
	cfg.MaxIdleConns = 30000
	require.NoError(t, confmap.NewFromStringMap(map[string]any{}).Unmarshal(&cfg))

	core, observed := observer.New(zapcore.WarnLevel)
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = nil
	settings.TracerProvider = nil
	settings.Logger = zap.New(core)
	_, err := cfg.ToClient(t.Context(), nil, settings)
	require.NoError(t, err)
	assert.Empty(t, observed.All())
}

// A zero-value ClientConfig used programmatically (without NewDefaultClientConfig)
// has keep-alives enabled on main. Test helpers downstream construct bare
// ClientConfig literals and must not silently switch to one-connection-per-request.
func TestKeepaliveCompatZeroValueClientConfig(t *testing.T) {
	cfg := ClientConfig{Endpoint: "http://localhost"}
	transport := compatClientTransport(t, cfg)
	assert.False(t, transport.DisableKeepAlives)
}

// Factories disable server keep-alives programmatically: splunkhecreceiver,
// remotetapprocessor, githubreceiver, gitlabreceiver, libhoneyreceiver,
// collectdreceiver, azurefunctionsreceiver, and prometheusremotewritereceiver
// in contrib all set KeepAlivesEnabled = false in createDefaultConfig. The
// built server must send `Connection: close`.
func TestKeepaliveCompatServerFactoryDisabledKeepAlives(t *testing.T) {
	cfg := NewDefaultServerConfig()
	cfg.KeepAlivesEnabled = false
	conf := confmap.NewFromStringMap(map[string]any{"endpoint": "localhost:0"})
	require.NoError(t, conf.Unmarshal(&cfg))

	resp := compatServeAndGet(t, cfg)
	assert.True(t, resp.Close, "server must disable keep-alives")
}

// The same via user configuration: `keep_alives_enabled: false` in yaml.
func TestKeepaliveCompatServerDeprecatedDisableViaConfig(t *testing.T) {
	cfg := NewDefaultServerConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":            "localhost:0",
		"keep_alives_enabled": false,
	})
	require.NoError(t, conf.Unmarshal(&cfg))

	resp := compatServeAndGet(t, cfg)
	assert.True(t, resp.Close, "server must disable keep-alives")
}

// A config loaded from deprecated fields must survive a marshal/unmarshal
// roundtrip: `print-initial-config` output is valid collector configuration,
// and tooling (e.g. the OpAMP supervisor) re-marshals effective configs.
func TestKeepaliveCompatClientMarshalRoundtrip(t *testing.T) {
	cfg := NewDefaultClientConfig()
	require.NoError(t, confmap.NewFromStringMap(map[string]any{"idle_conn_timeout": "60s"}).Unmarshal(&cfg))

	marshaled := confmap.New()
	require.NoError(t, marshaled.Marshal(cfg))

	cfg2 := NewDefaultClientConfig()
	require.NoError(t, marshaled.Unmarshal(&cfg2), "marshaled output must load without a mixed-config error")

	transport := compatClientTransport(t, cfg2)
	assert.Equal(t, 60*time.Second, transport.IdleConnTimeout)
	assert.Equal(t, 100, transport.MaxIdleConns)
}

func TestKeepaliveCompatServerMarshalRoundtrip(t *testing.T) {
	cfg := NewDefaultServerConfig()
	require.NoError(t, confmap.NewFromStringMap(map[string]any{"idle_timeout": "2m"}).Unmarshal(&cfg))

	marshaled := confmap.New()
	require.NoError(t, marshaled.Marshal(cfg))

	cfg2 := NewDefaultServerConfig()
	require.NoError(t, marshaled.Unmarshal(&cfg2), "marshaled output must load without a mixed-config error")
}

// Strict unmarshaling rejects unknown keys on main, but implementing
// confmap.Unmarshaler on ClientConfig/ServerConfig forfeits this for every
// embedding component: for squash embedding, confmap's embedded-structs hook
// decodes sibling fields with unused keys ignored, and for named sections the
// Unmarshal implementation must use WithIgnoreUnused to support the squash
// case. These tests document that accepted trade-off. If they start failing,
// strict checking has been restored (e.g. the Unmarshaler was removed or
// confmap learned to track unused keys across the hook) and they should be
// flipped back to asserting an error.
func TestKeepaliveCompatUnknownKeyIgnoredSquash(t *testing.T) {
	cfg := namedSquashClientConfig{ClientConfig: NewDefaultClientConfig()}
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":  "http://localhost:4318",
		"bogus_key": 1,
	})
	assert.NoError(t, conf.Unmarshal(&cfg), "unknown keys are ignored for configs embedding ClientConfig")
}

func TestKeepaliveCompatUnknownKeyIgnoredNamedSection(t *testing.T) {
	type outerConfig struct {
		Egress ClientConfig `mapstructure:"egress"`
	}
	cfg := outerConfig{Egress: NewDefaultClientConfig()}
	conf := confmap.NewFromStringMap(map[string]any{
		"egress": map[string]any{
			"endpoint":  "http://localhost:4318",
			"bogus_key": 1,
		},
	})
	assert.NoError(t, conf.Unmarshal(&cfg), "unknown keys are ignored inside a ClientConfig section")
}
