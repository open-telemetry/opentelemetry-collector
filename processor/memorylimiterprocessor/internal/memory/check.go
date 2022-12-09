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
	"go.opentelemetry.io/collector/internal/iruntime"
)

const (
	mibBytes = 1024 * 1024
)

// Checker is used to compare the limits against the current reported
// runtime allocation size.
type Checker struct {
	allocLimit uint64
	spikeLimit uint64
}

// TotalFunc defines a means of loading current memory used
// that can be overridden within tests
type TotalFunc func() (uint64, error)

func (fn TotalFunc) Load() (uint64, error) {
	if fn != nil {
		return fn()
	}
	return iruntime.TotalMemory()
}

func (fn TotalFunc) NewMemChecker(limitMiB uint64, spikeMiB uint64, limitPercentage uint64, spikePercentage uint64) (*Checker, error) {
	if hardLimit := limitMiB * mibBytes; hardLimit != 0 {
		spikeLimit := spikeMiB * mibBytes
		if spikeLimit == 0 {
			spikeLimit = hardLimit / 5
		}
		return &Checker{
			allocLimit: hardLimit,
			spikeLimit: spikeLimit,
		}, nil
	}
	totalMem, err := fn.Load()
	if err != nil {
		return nil, err
	}
	return &Checker{
		allocLimit: limitPercentage * totalMem / 100,
		spikeLimit: spikePercentage * totalMem / 100,
	}, nil
}

func (c Checker) AboveSoftLimit(ms *Stats) bool {
	return ms.Alloc >= c.allocLimit-c.spikeLimit
}

func (c Checker) AboveHardLimit(ms *Stats) bool {
	return ms.Alloc >= c.allocLimit
}

func (c Checker) SoftLimitMiB() uint64 { return c.spikeLimit }
func (c Checker) HardLimitMiB() uint64 { return c.allocLimit }
