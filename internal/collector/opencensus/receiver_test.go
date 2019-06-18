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

// Package ocreceiver wraps the functionality to start the end-point that
// receives data directly in the OpenCensus format.
package ocreceiver

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/cmd/occollector/app/builder"
	"github.com/open-telemetry/opentelemetry-service/processor/processortest"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
)

func TestStart(t *testing.T) {
	tests := []struct {
		name    string
		viperFn func() *viper.Viper
		wantErr bool
	}{
		{
			name: "default_config",
			viperFn: func() *viper.Viper {
				v := viper.New()
				v.Set("receivers.opencensus.{}", nil)
				return v
			},
		},
		{
			name: "invalid_port",
			viperFn: func() *viper.Viper {
				v := viper.New()
				v.Set("receivers.opencensus.port", -1)
				return v
			},
			wantErr: true,
		},
		{
			name: "missing_tls_files",
			viperFn: func() *viper.Viper {
				v := viper.New()
				v.Set("receivers.opencensus.tls_credentials.cert_file", "foo")
				return v
			},
			wantErr: true,
		},
		{
			name: "grpc_settings",
			viperFn: func() *viper.Viper {
				v := viper.New()
				v.Set("receivers.opencensus.port", 55678)
				v.Set("receivers.opencensus.max-recv-msg-size-mib", 32)
				v.Set("receivers.opencensus.max-concurrent-streams", 64)
				v.Set("receivers.opencensus.keepalive.server-parameters.max-connection-age", 180*time.Second)
				v.Set("receivers.opencensus.keepalive.server-parameters.max-connection-age-grace", 10*time.Second)
				v.Set("receivers.opencensus.keepalive.enforcement-policy.min-time", 60*time.Second)
				v.Set("receivers.opencensus.keepalive.enforcement-policy.permit-without-stream", true)
				return v
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Enforce that all configurations are actually recognized.
			v := tt.viperFn()
			rOpts := builder.OpenCensusReceiverCfg{}
			if err := v.Sub("receivers.opencensus").UnmarshalExact(&rOpts); err != nil {
				t.Errorf("UnmarshalExact error: %v", err)
				return
			}
			nopProcessor := processortest.NewNopTraceProcessor(nil)
			asyncErrChan := make(chan error, 1)
			got, err := Start(zap.NewNop(), v, nopProcessor, asyncErrChan)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				// TODO: (@pjanotti) current StopTraceReception, stop the whole receiver.
				// See https://github.com/census-instrumentation/opencensus-service/issues/559
				got.(*opencensusreceiver.Receiver).Stop()
			}
		})
	}
}
