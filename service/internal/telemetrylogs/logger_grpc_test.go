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

package telemetrylogs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"

	"go.opentelemetry.io/collector/config"
)

func TestGRPCLoggerInDevelopment(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  config.ServiceTelemetryLogs
		grpcLogMustBeEnabled bool
	}{
		{
			"dev_profile_should_enable_grpc_logs",
			config.ServiceTelemetryLogs{
				Development: true, // this must set the grpc logger to logger
				Level:       zapcore.InfoLevel,
				Encoding:    "console",
			},
			true,
		},
		{
			"prod_profile_should_not_enable_grpc_logs",
			config.ServiceTelemetryLogs{
				Development: false, // this must set the grpc logger to logger
				Level:       zapcore.InfoLevel,
				Encoding:    "console",
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// FIXME: this sets grpc logger for the entire test session.
			gLog := grpclog.NewLoggerV2(os.Stderr, os.Stderr, os.Stderr)
			grpclog.SetLoggerV2(gLog)

			called := false
			hook := zap.Hooks(func(entry zapcore.Entry) error {
				called = true
				return nil
			})

			_, err := NewLogger(test.cfg, []zap.Option{hook})
			assert.NoError(t, err)
			// write a grpc log
			grpclog.Info(test.name)
			// test whether the grpc log has been recorded from collector logger
			assert.Equal(t, called, test.grpcLogMustBeEnabled)
		})
	}
}
