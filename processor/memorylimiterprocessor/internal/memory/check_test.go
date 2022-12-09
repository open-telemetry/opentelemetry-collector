// Copyright The OpenTelemetry Authors
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

package memory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTotal(t *testing.T) {
	t.Parallel()

	errFailed := errors.New("failed")

	for _, tc := range []struct {
		name string
		fn   TotalFunc
		val  uint64
		err  error
	}{
		{
			name: "default invocation",
		},
		{
			name: "check value without error",
			fn: func() (uint64, error) {
				return 100, nil
			},
			val: 100,
			err: nil,
		},
		{
			name: "check value with error",
			fn: func() (uint64, error) {
				return 0, errFailed
			},
			val: 0,
			err: errFailed,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			total, err := tc.fn.Load()
			if tc.fn == nil {
				// exiting here due to allow coverage of default
				// method but since it can vary to os / arch
				// the values are not validated
				return
			}
			assert.Equal(t, tc.val, total, "Must match expected value")
			assert.ErrorIs(t, err, tc.err, "Must match the expected error")
		})
	}
}

func TestMemoryChecker(t *testing.T) {
	t.Parallel()

	errFailed := errors.New("failed")

	for _, tc := range []struct {
		name  string
		total TotalFunc
		usage StatsFunc

		limitMiB     uint64
		spikeMiB     uint64
		limitPercent uint64
		spikePercent uint64

		limiter *Checker

		aboveSoft bool
		aboveHard bool
		err       error
	}{
		{
			name: "no limits are breached using fixed limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 10 * mibBytes
			},
			limitMiB: 20,
			spikeMiB: 10,

			limiter: &Checker{
				allocLimit: 20 * mibBytes,
				spikeLimit: 10 * mibBytes,
			},

			aboveSoft: false,
			aboveHard: false,
		},
		{
			name: "no limits are breached using fixed limits no spike defined",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 10 * mibBytes
			},
			limitMiB: 20,

			limiter: &Checker{
				allocLimit: 20 * mibBytes,
				spikeLimit: 20 * mibBytes / 5,
			},

			aboveSoft: false,
			aboveHard: false,
		},
		{
			name: "limits are breached using fixed limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 30 * mibBytes
			},
			limitMiB: 20,
			spikeMiB: 10,

			limiter: &Checker{
				allocLimit: 20 * mibBytes,
				spikeLimit: 10 * mibBytes,
			},

			aboveSoft: true,
			aboveHard: true,
		},
		{
			name: "soft limits are breached using fixed limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 20 * mibBytes
			},
			limitMiB: 20,
			spikeMiB: 10,

			limiter: &Checker{
				allocLimit: 20 * mibBytes,
				spikeLimit: 10 * mibBytes,
			},

			aboveSoft: true,
			aboveHard: false,
		},
		{
			name: "no limits are breached using percentage limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 10 * mibBytes
			},
			limitPercent: 80,
			spikePercent: 60,

			limiter: &Checker{
				allocLimit: 100 * mibBytes * 80 / 100,
				spikeLimit: 100 * mibBytes * 60 / 100,
			},

			aboveSoft: false,
			aboveHard: false,
		},
		{
			name: "soft limits are breached using percentage limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 62 * mibBytes
			},
			limitPercent: 80,
			spikePercent: 60,

			limiter: &Checker{
				allocLimit: 100 * mibBytes * 80 / 100,
				spikeLimit: 100 * mibBytes * 60 / 100,
			},

			aboveSoft: true,
			aboveHard: false,
		},
		{
			name: "both limits are breached using percentage limits",
			total: func() (uint64, error) {
				return 100 * mibBytes, nil
			},
			usage: func(s *Stats) {
				s.Alloc = 88 * mibBytes
			},
			limitPercent: 80,
			spikePercent: 60,

			limiter: &Checker{
				allocLimit: 100 * mibBytes * 80 / 100,
				spikeLimit: 100 * mibBytes * 60 / 100,
			},

			aboveSoft: true,
			aboveHard: true,
		},
		{
			name: "total function errors",
			total: func() (uint64, error) {
				return 0, errFailed
			},
			err: errFailed,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			check, err := tc.total.NewMemChecker(
				tc.limitMiB,
				tc.spikeMiB,
				tc.limitPercent,
				tc.spikePercent,
			)
			require.ErrorIs(t, err, tc.err)
			if tc.err != nil {
				return
			}
			assert.Equal(t, tc.limiter.SoftLimitMiB(), check.SoftLimitMiB())
			assert.Equal(t, tc.limiter.HardLimitMiB(), check.HardLimitMiB())

			s, _ := tc.usage.Load(1000)

			assert.Equal(t, tc.aboveSoft, check.AboveSoftLimit(s))
			assert.Equal(t, tc.aboveHard, check.AboveHardLimit(s))
		})
	}
}
