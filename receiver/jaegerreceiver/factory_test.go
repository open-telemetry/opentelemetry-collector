// Copyright 2019, OpenCensus Authors
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

package jaegerreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/census-instrumentation/opencensus-service/internal/factories"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateReceiver(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.NoError(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(cfg, nil)
	assert.Equal(t, err, factories.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}

func TestCreateNoEndpoints(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*ConfigV2)

	rCfg.Protocols[protoThriftHTTP].Endpoint = ""
	rCfg.Protocols[protoThriftTChannel].Endpoint = ""
	_, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Error(t, err, "receiver creation with no endpoints must fail")
}

func TestCreateInvalidTChannelEndpoint(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*ConfigV2)

	rCfg.Protocols[protoThriftTChannel].Endpoint = ""
	_, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Error(t, err, "receiver creation with invalid tchannel endpoint must fail")
}

func TestCreateNoPort(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*ConfigV2)

	rCfg.Protocols[protoThriftHTTP].Endpoint = "127.0.0.1:"
	_, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Error(t, err, "receiver creation with no port number must fail")
}

func TestCreateLargePort(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*ConfigV2)

	rCfg.Protocols[protoThriftHTTP].Endpoint = "127.0.0.1:65536"
	_, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Error(t, err, "receiver creation with too large port number must fail")
}

func TestCreateNoProtocols(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*ConfigV2)

	delete(rCfg.Protocols, protoThriftHTTP)
	delete(rCfg.Protocols, protoThriftTChannel)
	_, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Error(t, err, "receiver creation with no protocols must fail")
}
