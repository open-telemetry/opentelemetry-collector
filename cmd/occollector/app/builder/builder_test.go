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

	"github.com/spf13/viper"
)

func TestReceiversEnabledByPresenceWithDefaultSettings(t *testing.T) {
	v, err := loadViperFromFile("./testdata/receivers_enabled.yaml")
	if err != nil {
		t.Fatalf("Failed to load viper from test file: %v", err)
	}

	jaegerEnabled, opencensusEnabled, zipkinEnabled := JaegerReceiverEnabled(v, "j"), OpenCensusReceiverEnabled(v, "oc"), ZipkinReceiverEnabled(v, "z")
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

	jaegerEnabled, opencensusEnabled, zipkinEnabled := JaegerReceiverEnabled(v, "j"), OpenCensusReceiverEnabled(v, "oc"), ZipkinReceiverEnabled(v, "z")
	if jaegerEnabled || opencensusEnabled || zipkinEnabled {
		t.Fatalf("Not all receivers were disabled j:%v oc:%v z:%v", jaegerEnabled, opencensusEnabled, zipkinEnabled)
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
