// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

type benchSender struct {
	mu            sync.Mutex
	latency       time.Duration
	latencyJitter time.Duration
	err           error
	errRate       float64
	rng           *rand.Rand
}

func newBenchSender() *benchSender {
	return &benchSender{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *benchSender) Send(ctx context.Context, req request.Request) error {
	s.mu.Lock()
	lat := s.latency
	jit := s.latencyJitter
	err := s.err
	rate := s.errRate
	s.mu.Unlock()

	if jit > 0 {
		lat += time.Duration(rand.Int63n(int64(jit)))
	}

	if lat > 0 {
		time.Sleep(lat)
	}

	if err != nil && rate > 0 {
		s.mu.Lock()
		roll := s.rng.Float64()
		s.mu.Unlock()
		if roll < rate {
			return err
		}
	}
	return nil
}

func (s *benchSender) SetConditions(latency time.Duration, jitter time.Duration, err error, rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = latency
	s.latencyJitter = jitter
	s.err = err
	s.errRate = rate
}

func BenchmarkQueueSender_ARC(b *testing.B) {
	tt := componenttest.NewTelemetry()
	defer tt.Shutdown(context.Background())

	qSet := queuebatch.AllSettings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: tt.NewTelemetrySettings(),
	}

	scenarios := []struct {
		name           string
		arcEnabled     bool
		numConsumers   int
		initialLimit   int
		maxConcurrency int
		latency        time.Duration
		jitter         time.Duration
		err            error
		errRate        float64
	}{
		// 1. Baseline / Regression
		{
			name:         "Baseline_Static_NoLatency",
			arcEnabled:   false,
			numConsumers: 10,
			latency:      0,
		},
		{
			name:         "Baseline_ARC_Disabled_NoLatency",
			arcEnabled:   false,
			numConsumers: 10,
			latency:      0,
		},
		{
			name:           "Baseline_ARC_Enabled_Steady",
			arcEnabled:     true,
			initialLimit:   10,
			maxConcurrency: 50,
			latency:        0,
		},

		// 2. CPU Stress / High Concurrency
		{
			name:           "Stress_ARC_200Concurrent_NoLatency",
			arcEnabled:     true,
			initialLimit:   200,
			maxConcurrency: 200,
			latency:        0,
		},

		// 3. Latency Compensation (Little's Law)
		{
			name:         "HighLatency_Static_10Conns",
			arcEnabled:   false,
			numConsumers: 10,
			latency:      10 * time.Millisecond,
		},
		{
			name:           "HighLatency_ARC_10to100Conns",
			arcEnabled:     true,
			initialLimit:   10,
			maxConcurrency: 100,
			latency:        10 * time.Millisecond,
		},

		// 4. Jitter / Noisy Network
		{
			name:           "Jittery_ARC_5-15ms",
			arcEnabled:     true,
			initialLimit:   20,
			maxConcurrency: 100,
			latency:        5 * time.Millisecond,
			jitter:         10 * time.Millisecond,
		},

		// 5. Backpressure / Safety Valve
		{
			name:         "Backpressure_Static_ErrorSpike",
			arcEnabled:   false,
			numConsumers: 10,
			latency:      1 * time.Millisecond,
			err:          errors.New("429 Too Many Requests"),
			errRate:      0.5,
		},
		{
			name:           "Backpressure_ARC_ErrorSpike",
			arcEnabled:     true,
			initialLimit:   10,
			maxConcurrency: 50,
			latency:        1 * time.Millisecond,
			err:            errors.New("429 Too Many Requests"),
			errRate:        0.5,
		},

		// 6. Recovery / Edge Cases
		{
			// Simulates a state where ARC had scaled up (to 100) due to previous load,
			// but latency has now dropped to 0. Checks if high limits hurt performance.
			name:           "Recovery_ARC_100ms_to_0ms",
			arcEnabled:     true,
			initialLimit:   100,
			maxConcurrency: 100,
			latency:        0,
		},
		{
			// Worst case contention: High concurrency static limits fighting over failing requests.
			name:         "WorstCase_Static_100Conns_ErrorSpike",
			arcEnabled:   false,
			numConsumers: 100,
			latency:      1 * time.Millisecond,
			err:          errors.New("503 Service Unavailable"),
			errRate:      0.5,
		},
		{
			// Worst case handling: ARC should shrink this limit and reduce contention.
			name:           "WorstCase_ARC_100Conns_ErrorSpike",
			arcEnabled:     true,
			initialLimit:   100,
			maxConcurrency: 100,
			latency:        1 * time.Millisecond,
			err:            errors.New("503 Service Unavailable"),
			errRate:        0.5,
		},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			qCfg := NewDefaultQueueConfig()
			qCfg.Enabled = true
			qCfg.QueueSize = 100_000
			qCfg.Batch = configoptional.Optional[queuebatch.BatchConfig]{}
			qCfg.NumConsumers = sc.numConsumers

			qCfg.Arc.Enabled = sc.arcEnabled
			if sc.arcEnabled {
				qCfg.Arc.InitialLimit = sc.initialLimit
				qCfg.Arc.MaxConcurrency = sc.maxConcurrency
			}

			bs := newBenchSender()
			bs.SetConditions(sc.latency, sc.jitter, sc.err, sc.errRate)

			qs, err := NewQueueSender(qSet, qCfg, "bench_exporter", sender.NewSender(bs.Send))
			require.NoError(b, err)
			require.NoError(b, qs.Start(context.Background(), componenttest.NewNopHost()))
			defer qs.Shutdown(context.Background())

			req := &requesttest.FakeRequest{Items: 10}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = qs.Send(context.Background(), req)
			}
		})
	}
}
