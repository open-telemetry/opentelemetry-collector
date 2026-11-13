// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"
)

type fatalTestExt struct {
	component.StartFunc
	component.ShutdownFunc
}

func newFatalTestCollector(t *testing.T, newExt func() fatalTestExt) *Collector {
	t.Helper()
	factories, err := nopFactories()
	require.NoError(t, err)
	f := extension.NewFactory(
		component.MustNewType("nop"),
		func() component.Config { return &struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return newExt(), nil
		},
		component.StabilityLevelStable,
	)
	factories.Extensions[f.Type()] = f
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		SkipSettingGRPCLogger:  true,
		Factories:              func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	})
	require.NoError(t, err)
	return col
}

// runWithTimeout turns a hang into a test failure instead of a stuck test run.
func runWithTimeout(t *testing.T, col *Collector) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- col.Run(context.Background()) }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("collector hung")
		return nil
	}
}

// Report FatalError, then return the error from Start.
func TestCollectorFatalErrorInStartFailsStartup(t *testing.T) {
	errBoom := errors.New("boom")
	col := newFatalTestCollector(t, func() fatalTestExt {
		return fatalTestExt{StartFunc: func(_ context.Context, host component.Host) error {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errBoom))
			return errBoom
		}}
	})

	require.ErrorContains(t, runWithTimeout(t, col), "boom")
	assert.Equal(t, StateClosed, col.GetState())
}

// A background goroutine reports FatalError while startup is still running.
// Start returns nil, so only the status reporter's lock can block the Run goroutine.
func TestCollectorFatalErrorFromBackgroundGoroutineDuringStartup(t *testing.T) {
	col := newFatalTestCollector(t, func() fatalTestExt {
		return fatalTestExt{StartFunc: func(_ context.Context, host component.Host) error {
			// Using reported channel to signal report status complete
			reported := make(chan struct{})
			go func() {
				componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errors.New("boom")))
				close(reported)
			}()

			// waiting on report status completion to avoid sleeping excessively in case of no deadlocks.
			select {
			case <-reported:
			case <-time.After(100 * time.Millisecond):
			}
			return nil
		}}
	})

	require.NoError(t, runWithTimeout(t, col)) // startup completes, then the buffered error shuts it down
	assert.Equal(t, StateClosed, col.GetState())
}

// A fatal error reported while the retiring config shuts down must not kill its replacement.
func TestCollectorReloadIgnoresFatalErrorFromRetiredConfig(t *testing.T) {
	var generation atomic.Int32
	col := newFatalTestCollector(t, func() fatalTestExt {
		gen := generation.Add(1)
		var host component.Host
		return fatalTestExt{
			StartFunc: func(_ context.Context, h component.Host) error { host = h; return nil },
			ShutdownFunc: func(context.Context) error {
				if gen == 1 {
					componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errors.New("stale")))
				}
				return nil
			},
		}
	})

	wg := startCollector(context.Background(), t, col)
	require.Eventually(t, func() bool { return col.GetState() == StateRunning }, 2*time.Second, 10*time.Millisecond)

	col.signalsChannel <- SIGHUP // full reload: gen 1 shuts down (reports fatal), gen 2 starts
	require.Eventually(t, func() bool {
		return generation.Load() == 2 && col.GetState() == StateRunning
	}, 2*time.Second, 10*time.Millisecond)
	require.Never(t, func() bool { return col.GetState() == StateClosed }, 300*time.Millisecond, 20*time.Millisecond)

	col.Shutdown()
	wg.Wait()
}

// A fatal error reported while the collector shuts down must not hang it
func TestCollectorFatalErrorDuringShutdown(t *testing.T) {
	col := newFatalTestCollector(t, func() fatalTestExt {
		var host component.Host
		return fatalTestExt{
			StartFunc: func(_ context.Context, h component.Host) error { host = h; return nil },
			ShutdownFunc: func(context.Context) error {
				componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errors.New("boom")))
				return nil
			},
		}
	})

	done := make(chan error, 1)
	go func() { done <- col.Run(context.Background()) }()
	require.Eventually(t, func() bool { return col.GetState() == StateRunning }, 2*time.Second, 10*time.Millisecond)

	shutdownDone := make(chan struct{})

	go func() {
		col.Shutdown() // blocks until Run has finished its cleanup
		close(shutdownDone)
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("collector hung during shutdown")
	}
	select {
	case <-shutdownDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not return")
	}
	assert.Equal(t, StateClosed, col.GetState())
}
