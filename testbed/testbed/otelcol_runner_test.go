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

package testbed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func TestNewInProcessPipeline(t *testing.T) {
	factories, err := defaultcomponents.Components()
	assert.NoError(t, err)
	sender := NewOTLPTraceDataSender(DefaultHost, GetAvailablePort(t))
	receiver := NewOTLPDataReceiver(DefaultOTLPPort)
	runner := NewInProcessCollector(factories)

	format := `
receivers:%v
exporters:%v
processors:
  batch:

extensions:

service:
  extensions:
  pipelines:
    traces:
      receivers: [%v]
      processors: [batch]
      exporters: [%v]
`
	config := fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		sender.ProtocolName(),
		receiver.ProtocolName(),
	)
	configCleanup, cfgErr := runner.PrepareConfig(config)
	defer configCleanup()
	assert.NoError(t, cfgErr)
	assert.NotNil(t, configCleanup)
	assert.NotNil(t, runner.config)
	args := StartParams{}
	defer runner.Stop()
	assert.NoError(t, runner.Start(args))
	assert.NotNil(t, runner.svc)
}
