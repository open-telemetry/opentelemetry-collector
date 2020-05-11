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

package hwreceiver

import (
	"context"
	"net/http"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"

	"github.com/open-telemetry/opentelemetry-collector/pipelines/hwpipeline"
)

type Config struct {
	configmodels.ReceiverSettings `mapstructure:",squash"`
}

const (
	typeStr = "hw"
)

type Factory struct {
}

var _ hwpipeline.ReceiverFactory = (*Factory)(nil)

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func (f *Factory) CreateHWReceiver(nextConsumer hwpipeline.Consumer) component.Receiver {
	return &hwReceiver{nextConsumer}
}

type hwReceiver struct {
	next hwpipeline.Consumer
}

// start http server and listen for pings

func (p *hwReceiver) Start(_ context.Context, _ component.Host) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p.next.ConsumeHW(hwpipeline.HelloWorld{r.URL.Path})
	})
	go http.ListenAndServe("localhost:5000", nil)
	return nil
}

func (p *hwReceiver) Shutdown(context.Context) error {
	return nil
}
