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

package builder

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestMultiAndQueuedSpanProcessorConfig(t *testing.T) {
	v, err := loadViperFromFile("./testdata/queued_exporters.yaml")
	if err != nil {
		t.Fatalf("Failed to load viper from test file: %v", err)
	}

	fst := NewDefaultQueuedSpanProcessorCfg()
	fst.Name = "proc-tchannel"
	fst.NumWorkers = 13
	fst.QueueSize = 1300
	fst.SenderType = ThriftTChannelSenderType
	fst.SenderConfig = &JaegerThriftTChannelSenderCfg{
		CollectorHostPorts:        []string{":123", ":321"},
		DiscoveryMinPeers:         7,
		DiscoveryConnCheckTimeout: time.Second * 7,
	}
	fst.RawConfig = v.Sub(queuedExportersConfigKey).Sub("proc-tchannel")
	snd := NewDefaultQueuedSpanProcessorCfg()
	snd.Name = "proc-http"
	snd.RetryOnFailure = false
	snd.BackoffDelay = 3 * time.Second
	snd.SenderType = ThriftHTTPSenderType
	snd.SenderConfig = &JaegerThriftHTTPSenderCfg{
		CollectorEndpoint: "https://somedomain.com/api/traces",
		Headers:           map[string]string{"x-header-key": "00000000-0000-0000-0000-000000000001"},
		Timeout:           time.Second * 5,
	}
	snd.RawConfig = v.Sub(queuedExportersConfigKey).Sub("proc-http")

	wCfg := &MultiSpanProcessorCfg{
		Processors: []*QueuedSpanProcessorCfg{fst, snd},
	}

	gCfg := NewDefaultMultiSpanProcessorCfg().InitFromViper(v)

	// Viper sometimes gets these out of order, which is not an issue for production, but
	// a problem for tests. Enforce that the order matches the expected one.
	if gCfg.Processors[0].Name == snd.Name {
		gCfg.Processors[0], gCfg.Processors[1] = gCfg.Processors[1], gCfg.Processors[0]
	}

	for i := range wCfg.Processors {
		if !reflect.DeepEqual(*wCfg.Processors[i], *gCfg.Processors[i]) {
			t.Errorf("Wanted %+v but got %+v", *wCfg.Processors[i], *gCfg.Processors[i])
		}
	}
}

func loadViperFromFile(file string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return v, nil
}
