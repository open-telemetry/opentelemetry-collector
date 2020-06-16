// Copyright 2020, OpenTelemetry Authors
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
	runner := NewInProcessCollector(factories)

	format := `
receivers:%v
exporters:%v
processors:
  batch:
  queued_retry:

extensions:

service:
  extensions:
  pipelines:
    traces:
      receivers: [%v]
      processors: [batch, queued_retry]
      exporters: [%v]
`
	sender := NewOTLPTraceDataSender(DefaultHost, GetAvailablePort(t))
	receiver := NewOTLPDataReceiver(DefaultOTLPPort)
	config := fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		sender.ProtocolName(),
		receiver.ProtocolName(),
	)
	cfgFilename, cfgErr := runner.PrepareConfig(config)
	assert.NoError(t, cfgErr)
	assert.Equal(t, "", cfgFilename)
	assert.NotNil(t, runner.config)
	args := StartParams{}
	defer runner.Stop()
	endpoint, startErr := runner.Start(args)
	assert.NoError(t, startErr)
	assert.Equal(t, DefaultHost, endpoint)
	assert.NotNil(t, runner.svc)
}
