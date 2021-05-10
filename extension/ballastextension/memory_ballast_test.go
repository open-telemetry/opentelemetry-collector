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
)

func TestMemoryBallast(t *testing.T) {
	config := &Config{
		SizeMiB: 13,
	}

	mbExt := newMemoryBallast(config, zap.NewNop())
	require.NotNil(t, mbExt)
	assert.Nil(t, mbExt.ballast)

	assert.NoError(t, mbExt.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 13*megaBytes, len(mbExt.ballast))

	assert.NoError(t, mbExt.Shutdown(context.Background()))
	assert.Nil(t, mbExt.ballast)
}

func TestMemoryBallast_ZeroSize(t *testing.T) {
	config := &Config{}

	mbExt := newMemoryBallast(config, zap.NewNop())
	require.NotNil(t, mbExt)
	assert.Nil(t, mbExt.ballast)

	assert.NoError(t, mbExt.Start(context.Background(), componenttest.NewNopHost()))
	assert.Nil(t, mbExt.ballast)

	assert.NoError(t, mbExt.Shutdown(context.Background()))
	assert.Nil(t, mbExt.ballast)
}
