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

package prometheusremotewriteexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
)

// Tests whether or not the default Exporter factory can instantiate a properly interfaced Exporter with default conditions
func Test_createDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

// Tests whether or not a correct Metrics Exporter from the default Config parameters
func Test_createMetricsExporter(t *testing.T) {

	invalidConfig := createDefaultConfig().(*Config)
	invalidConfig.HTTPClientSettings = confighttp.HTTPClientSettings{}
	invalidTLSConfig := createDefaultConfig().(*Config)
	invalidTLSConfig.HTTPClientSettings.TLSSetting = configtls.TLSClientSetting{
		TLSSetting: configtls.TLSSetting{
			CAFile:   "non-existent file",
			CertFile: "",
			KeyFile:  "",
		},
		Insecure:   false,
		ServerName: "",
	}
	tests := []struct {
		name        string
		cfg         config.Exporter
		params      component.ExporterCreateParams
		returnError bool
		startError  bool
	}{
		{name: "success_case",
			cfg:    createDefaultConfig(),
			params: component.ExporterCreateParams{Logger: zap.NewNop()},
		},
		{name: "fail_case",
			params:      component.ExporterCreateParams{Logger: zap.NewNop()},
			returnError: true,
		},
		{name: "invalid_config_case",
			cfg:         invalidConfig,
			params:      component.ExporterCreateParams{Logger: zap.NewNop()},
			returnError: true,
		},
		{name: "invalid_tls_config_case",
			cfg:         invalidTLSConfig,
			params:      component.ExporterCreateParams{Logger: zap.NewNop()},
			returnError: false,
			startError:  true,
		},
	} // run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := createMetricsExporter(context.Background(), tt.params, tt.cfg)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			err = exp.Start(context.Background(), componenttest.NewNopHost())
			if tt.startError {
				assert.Error(t, err)
			}
		})
	}
}
