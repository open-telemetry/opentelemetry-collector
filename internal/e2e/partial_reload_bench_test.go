// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/telemetrytest"
)

// generatorConfig is the configuration for the generator receiver.
type generatorConfig struct {
	Label string `mapstructure:"label"`
	Rate  int    `mapstructure:"rate"`
}

var generatorType = component.MustNewType("generator")

// generatorReceiver produces log records at a configurable rate.
type generatorReceiver struct {
	nextConsumer consumer.Logs
	config       generatorConfig
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func (g *generatorReceiver) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel

	interval := time.Second / time.Duration(g.config.Rate)
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var counter atomic.Int64
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logs := plog.NewLogs()
				lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				lr.Body().SetStr(fmt.Sprintf("event-%d", counter.Add(1)))
				// Best-effort send; ignore errors during shutdown.
				_ = g.nextConsumer.ConsumeLogs(ctx, logs)
			}
		}
	}()
	return nil
}

func (g *generatorReceiver) Shutdown(context.Context) error {
	if g.cancel != nil {
		g.cancel()
	}
	g.wg.Wait()
	return nil
}

// newGeneratorReceiverFactory creates a receiver factory for the generator receiver.
func newGeneratorReceiverFactory() receiver.Factory {
	return receiver.NewFactory(
		generatorType,
		func() component.Config {
			return &generatorConfig{Rate: 1000}
		},
		receiver.WithLogs(func(_ context.Context, _ receiver.Settings, cfg component.Config, next consumer.Logs) (receiver.Logs, error) {
			c := cfg.(*generatorConfig)
			return &generatorReceiver{
				nextConsumer: next,
				config:       *c,
			}, nil
		}, component.StabilityLevelAlpha),
	)
}

// benchFakeProvider implements confmap.Provider for benchmark tests.
type benchFakeProvider struct {
	scheme string
	ret    func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)
}

func (f *benchFakeProvider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return f.ret(ctx, uri, watcher)
}

func (f *benchFakeProvider) Scheme() string {
	return f.scheme
}

func (f *benchFakeProvider) Shutdown(context.Context) error {
	return nil
}

func newBenchFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)) confmap.ProviderFactory {
	return confmap.NewProviderFactory(func(_ confmap.ProviderSettings) confmap.Provider {
		return &benchFakeProvider{
			scheme: scheme,
			ret:    ret,
		}
	})
}

// benchTelemetryConfig is a minimal telemetry config for benchmarks.
type benchTelemetryConfig struct{}

func (benchTelemetryConfig) Validate() error { return nil }

// tlsCerts holds PEM-encoded certificate material for TLS tests.
type tlsCerts struct {
	caPEM   string
	certPEM string
	keyPEM  string
}

// generateSelfSignedCert creates a self-signed CA and a server certificate
// for localhost, returning PEM-encoded strings suitable for in-memory TLS config.
func generateSelfSignedCert(tb testing.TB) tlsCerts {
	tb.Helper()

	// Generate CA key and cert.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(tb, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bench-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(tb, err)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(tb, err)

	// Generate server key and cert signed by CA.
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(tb, err)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(tb, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})

	keyDER, err := x509.MarshalECPrivateKey(serverKey)
	require.NoError(tb, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tlsCerts{
		caPEM:   string(caPEM),
		certPEM: string(certPEM),
		keyPEM:  string(keyPEM),
	}
}

// buildSourceConfig builds a confmap.Conf for the source collector.
// The generator label changes on each generation to trigger a receiver rebuild.
// TLS is configured with in-memory PEM certs so the exporter verifies the sink's
// certificate via the CA. On full reload the TLS handshake must be re-established.
func buildSourceConfig(sinkAddr string, generation int, certs tlsCerts) map[string]any {
	return map[string]any{
		"receivers": map[string]any{
			"generator": map[string]any{
				"label": fmt.Sprintf("gen-%d", generation),
				"rate":  1000,
			},
		},
		"exporters": map[string]any{
			"otlp": map[string]any{
				"endpoint": sinkAddr,
				"tls": map[string]any{
					"ca_pem":               certs.caPEM,
					"server_name_override": "localhost",
				},
				"sending_queue": map[string]any{
					"enabled": false,
				},
			},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{
					"receivers": []any{"generator"},
					"exporters": []any{"otlp"},
				},
			},
		},
	}
}

// BenchmarkPartialReloadThroughput measures how many events each reload mode
// can push through the pipeline in a fixed time window while reloads are
// happening. The generator receiver produces events at a steady rate; each
// reload briefly pauses generation while the receiver restarts. Partial reload
// restarts only the receiver, so the pause is shorter and more events get
// through. Full reload restarts the entire service, causing a longer pause.
//
// Run with: go test -bench BenchmarkPartialReloadThroughput -benchtime 1x -timeout 15m -run ^$ -v
func BenchmarkPartialReloadThroughput(b *testing.B) {
	b.Run("partial_reload", func(b *testing.B) {
		for range b.N {
			benchmarkReloadThroughput(b, true)
		}
	})
	b.Run("full_reload", func(b *testing.B) {
		for range b.N {
			benchmarkReloadThroughput(b, false)
		}
	})
}

func benchmarkReloadThroughput(b *testing.B, partialReloadEnabled bool) {
	const (
		benchDuration  = 5 * time.Minute
		reloadInterval = 5 * time.Second
	)

	// Set the master feature gate; the receiver phase gate is Beta (on by default).
	require.NoError(b, featuregate.GlobalRegistry().Set("service.partialReload", partialReloadEnabled))
	b.Cleanup(func() {
		require.NoError(b, featuregate.GlobalRegistry().Set("service.partialReload", false))
	})

	// --- Generate TLS certs for the link between exporter and sink ---
	certs := generateSelfSignedCert(b)

	// --- Sink: standalone OTLP gRPC receiver with TLS, connected to LogsSink ---
	sinkAddr := testutil.GetAvailableLocalAddress(b)
	sink := &consumertest.LogsSink{}

	sinkFactory := otlpreceiver.NewFactory()
	sinkCfg := sinkFactory.CreateDefaultConfig().(*otlpreceiver.Config)
	sinkCfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  sinkAddr,
			Transport: confignet.TransportTypeTCP,
		},
		TLS: configoptional.Some(configtls.ServerConfig{
			Config: configtls.Config{
				CertPem: configopaque.String(certs.certPEM),
				KeyPem:  configopaque.String(certs.keyPEM),
			},
		}),
	})
	sinkCfg.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	sinkRecv, err := sinkFactory.CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(sinkFactory.Type()),
		sinkCfg,
		sink,
	)
	require.NoError(b, err)
	require.NoError(b, sinkRecv.Start(context.Background(), componenttest.NewNopHost()))
	b.Cleanup(func() {
		require.NoError(b, sinkRecv.Shutdown(context.Background()))
	})

	// --- Source collector: generator receiver → OTLP gRPC exporter ---
	observerCore, observedLogs := observer.New(zapcore.InfoLevel)

	var configMu sync.Mutex
	generation := 0
	var watcher confmap.WatcherFunc

	fileProvider := newBenchFakeProvider("file", func(_ context.Context, _ string, w confmap.WatcherFunc) (*confmap.Retrieved, error) {
		watcher = w
		configMu.Lock()
		gen := generation
		configMu.Unlock()
		return confmap.NewRetrieved(buildSourceConfig(sinkAddr, gen, certs))
	})

	generatorFactory := newGeneratorReceiverFactory()
	otlpExpFactory := otlpexporter.NewFactory()

	receivers, err := otelcol.MakeFactoryMap[receiver.Factory](generatorFactory)
	require.NoError(b, err)
	exporters, err := otelcol.MakeFactoryMap[exporter.Factory](otlpExpFactory)
	require.NoError(b, err)
	connectors, err := otelcol.MakeFactoryMap(connectortest.NewNopFactory())
	require.NoError(b, err)
	extensions, err := otelcol.MakeFactoryMap(extensiontest.NewNopFactory())
	require.NoError(b, err)
	processors, err := otelcol.MakeFactoryMap(processortest.NewNopFactory())
	require.NoError(b, err)

	factories := otelcol.Factories{
		Receivers:  receivers,
		Exporters:  exporters,
		Connectors: connectors,
		Extensions: extensions,
		Processors: processors,
		ReceiverModules: map[component.Type]string{
			generatorType: "test/generator v0.0.0",
		},
		ExporterModules: map[component.Type]string{
			otlpExpFactory.Type(): "go.opentelemetry.io/collector/exporter/otlpexporter v0.0.0",
		},
		ConnectorModules: map[component.Type]string{
			connectortest.NewNopFactory().Type(): "test/connector v0.0.0",
		},
		ExtensionModules: map[component.Type]string{
			extensiontest.NewNopFactory().Type(): "test/extension v0.0.0",
		},
		ProcessorModules: map[component.Type]string{
			processortest.NewNopFactory().Type(): "test/processor v0.0.0",
		},
		Telemetry: telemetry.NewFactory(
			func() component.Config { return benchTelemetryConfig{} },
			telemetrytest.WithLogger(zap.New(observerCore), nil),
		),
	}

	col, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: func() (otelcol.Factories, error) { return factories, nil },
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{"file:bench-config"},
				ProviderFactories: []confmap.ProviderFactory{fileProvider},
			},
		},
	})
	require.NoError(b, err)

	ctx, cancel := context.WithCancel(context.Background())
	var colWg sync.WaitGroup
	colWg.Add(1)
	go func() {
		defer colWg.Done()
		_ = col.Run(ctx)
	}()

	// Wait for collector to be running.
	require.Eventually(b, func() bool {
		return otelcol.StateRunning == col.GetState()
	}, 10*time.Second, 10*time.Millisecond)

	// --- Reload loop ---
	deadline := time.Now().Add(benchDuration)
	reloads := 0

	for time.Now().Before(deadline) {
		time.Sleep(reloadInterval)
		if time.Now().After(deadline) {
			break
		}

		logCountBefore := observedLogs.Len()

		configMu.Lock()
		generation++
		configMu.Unlock()

		watcher(&confmap.ChangeEvent{})
		reloads++

		// Wait for reload to complete.
		reloadDeadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(reloadDeadline) {
			logs := observedLogs.All()
			for j := logCountBefore; j < len(logs); j++ {
				msg := logs[j].Message
				if partialReloadEnabled && msg == "Partial receiver reload completed successfully" {
					goto done
				}
				if !partialReloadEnabled && msg == "Config updated, restart service" {
					for col.GetState() != otelcol.StateRunning && time.Now().Before(reloadDeadline) {
						time.Sleep(1 * time.Millisecond)
					}
					goto done
				}
			}
			time.Sleep(100 * time.Microsecond)
		}
	done:
	}

	// Shutdown collector.
	cancel()
	colWg.Wait()

	// --- Report results ---
	// Allow a short grace period for in-flight events to arrive at the sink.
	time.Sleep(2 * time.Second)

	received := sink.LogRecordCount()
	throughput := float64(received) / benchDuration.Seconds()

	b.ReportMetric(float64(received), "events_pushed")
	b.ReportMetric(throughput, "events/sec")
	b.ReportMetric(float64(reloads), "reloads")
}
