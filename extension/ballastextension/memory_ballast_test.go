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

package ballastextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal/iruntime"
)

func TestMemoryBallast(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		getTotalMem func() (uint64, error)
		expect      int
	}{
		{
			name: "test_abs_ballast",
			config: &Config{
				SizeMiB: 13,
			},
			getTotalMem: iruntime.TotalMemory,
			expect:      13 * megaBytes,
		},
		{
			name: "test_abs_ballast_priority",
			config: &Config{
				SizeMiB:          13,
				SizeInPercentage: 20,
			},
			getTotalMem: iruntime.TotalMemory,
			expect:      13 * megaBytes,
		},
		{
			name:        "test_ballast_zero_val",
			config:      &Config{},
			getTotalMem: iruntime.TotalMemory,
			expect:      0,
		},
		{
			name: "test_ballast_in_percentage",
			config: &Config{
				SizeInPercentage: 20,
			},
			getTotalMem: mockTotalMem,
			expect:      20 * megaBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbExt := newMemoryBallast(tt.config, zap.NewNop(), tt.getTotalMem)
			require.NotNil(t, mbExt)
			assert.Nil(t, mbExt.ballast)

			assert.NoError(t, mbExt.Start(context.Background(), componenttest.NewNopHost()))
			assert.Equal(t, tt.expect, len(mbExt.ballast))

			assert.NoError(t, mbExt.Shutdown(context.Background()))
			assert.Nil(t, mbExt.ballast)
		})
	}
}

func mockTotalMem() (uint64, error) {
	return uint64(100 * megaBytes), nil
}
