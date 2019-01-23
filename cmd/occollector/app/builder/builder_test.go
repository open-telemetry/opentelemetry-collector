// Copyright 2018, OpenCensus Authors
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

func TestReceiversEnabledByPresenceWithDefaultSettings(t *testing.T) {
	v, err := loadViperFromFile("./testdata/receivers_enabled.yaml")
	if err != nil {
		t.Fatalf("Failed to load viper from test file: %v", err)
	}

	jaegerEnabled, opencensusEnabled, zipkinEnabled := JaegerReceiverEnabled(v), OpenCensusReceiverEnabled(v), ZipkinReceiverEnabled(v)
	if !jaegerEnabled || !opencensusEnabled || !zipkinEnabled {
		t.Fatalf("Some of the expected receivers were not enabled j:%v oc:%v z:%v", jaegerEnabled, opencensusEnabled, zipkinEnabled)
	}

	wj := NewDefaultJaegerReceiverCfg()
	gj, err := wj.InitFromViper(v)
	if err != nil {
		t.Errorf("Failed to InitFromViper for Jaeger receiver: %v", err)
	} else if !reflect.DeepEqual(wj, gj) {
		t.Errorf("Incorrect config for Jaeger receiver, want %v got %v", wj, gj)
	}

	woc := NewDefaultOpenCensusReceiverCfg()
	goc, err := woc.InitFromViper(v)
	if err != nil {
		t.Errorf("Failed to InitFromViper for OpenCensus receiver: %v", err)
	} else if !reflect.DeepEqual(woc, goc) {
		t.Errorf("Incorrect config for OpenCensus receiver, want %v got %v", woc, goc)
	}

	wz := NewDefaultZipkinReceiverCfg()
	gz, err := wz.InitFromViper(v)
	if err != nil {
		t.Errorf("Failed to InitFromViper for Zipkin receiver: %v", err)
	} else if !reflect.DeepEqual(wz, gz) {
		t.Errorf("Incorrect config for Zipkin receiver, want %v got %v", wz, gz)
	}
}

func TestReceiversDisabledByPresenceWithDefaultSettings(t *testing.T) {
	v, err := loadViperFromFile("./testdata/receivers_disabled.yaml")
	if err != nil {
		t.Fatalf("Failed to load viper from test file: %v", err)
	}

	jaegerEnabled, opencensusEnabled, zipkinEnabled := JaegerReceiverEnabled(v), OpenCensusReceiverEnabled(v), ZipkinReceiverEnabled(v)
	if jaegerEnabled || opencensusEnabled || zipkinEnabled {
		t.Fatalf("Not all receivers were disabled j:%v oc:%v z:%v", jaegerEnabled, opencensusEnabled, zipkinEnabled)
	}
}

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
