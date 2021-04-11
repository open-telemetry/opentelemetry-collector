// Copyright 2020 Splunk, Inc.
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

package vaultconfigsource

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/experimental/configsource"
)

func TestVaultLoadConfig(t *testing.T) {
	fileName := path.Join("testdata", "config.yaml")
	v, err := config.NewParserFromFile(fileName)
	require.NoError(t, err)

	factories := map[config.Type]configsource.Factory{
		typeStr: NewFactory(),
	}

	actualSettings, err := configsource.Load(context.Background(), v, factories)
	require.NoError(t, err)

	expectedSettings := map[string]configsource.ConfigSettings{
		"vault": &Config{
			Settings: &configsource.Settings{
				TypeVal: "vault",
				NameVal: "vault",
			},
			Endpoint:     "http://localhost:8200",
			Token:        "dev_token",
			Path:         "secret/kv",
			PollInterval: 1 * time.Minute,
		},
		"vault/poll_interval": &Config{
			Settings: &configsource.Settings{
				TypeVal: "vault",
				NameVal: "vault/poll_interval",
			},
			Endpoint:     "https://localhost:8200",
			Token:        "other_token",
			Path:         "other/path/kv",
			PollInterval: 10 * time.Second,
		},
	}

	require.Equal(t, expectedSettings, actualSettings)

	params := configsource.CreateParams{
		Logger: zap.NewNop(),
	}
	_, err = configsource.Build(context.Background(), actualSettings, params, factories)
	require.NoError(t, err)
}
