// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type config struct {
	StringKey string                                           `mapstructure:"string_key"`
	HTTP      configoptional.Optional[confighttp.ServerConfig] `mapstructure:"http"`
	Grpc      configoptional.Optional[configgrpc.ServerConfig] `mapstructure:"grpc"`
}

func TestOptionalDefaults(t *testing.T) {
	testCases := []struct {
		suffix string
		err    string
	}{
		{
			suffix: "disabled",
		},
		{
			suffix: "enabled_error",
			err:    `grpc: netaddr: invalid transport type ""`,
		},
		{
			suffix: "enabled_noerror",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.suffix, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(fmt.Sprintf("testdata/default_%s.yaml", tt.suffix))
			require.NoError(t, err)

			cfg := config{
				HTTP: configoptional.Default(confighttp.ServerConfig{}),
				Grpc: configoptional.Default(configgrpc.ServerConfig{}),
			}

			err = conf.Unmarshal(&cfg)
			require.NoError(t, err)

			err = xconfmap.Validate(cfg)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
		})
	}
}
