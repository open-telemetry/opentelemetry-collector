// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stanzareceiver

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
)

func TestDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	params := component.ReceiverCreateParams{
		Logger: zap.NewNop(),
	}
	receiver, err := createLogsReceiver(context.Background(), params, createDefaultConfig(), &mockLogsConsumer{})
	require.NoError(t, err, "receiver creation failed")
	require.NotNil(t, receiver, "receiver creation failed")

	badCfg := createDefaultConfig().(*Config)
	badCfg.OffsetsFile = os.Args[0] // current executable cannot be opened
	receiver, err = createLogsReceiver(context.Background(), params, badCfg, &mockLogsConsumer{})
	require.Error(t, err, "receiver creation should fail if offsets file is invalid")
	require.Nil(t, receiver, "receiver creation should have failed due to invalid offsets file")
}
