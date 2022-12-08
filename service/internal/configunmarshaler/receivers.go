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

package configunmarshaler // import "go.opentelemetry.io/collector/service/internal/configunmarshaler"

import (
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/receiver"
)

// receiversKeyName is the configuration key name for receivers section.
const receiversKeyName = "receivers"

type Receivers struct {
	recvs map[component.ID]component.Config

	factories map[component.Type]receiver.Factory
}

func NewReceivers(factories map[component.Type]receiver.Factory) *Receivers {
	return &Receivers{factories: factories}
}

func (r *Receivers) Unmarshal(conf *confmap.Conf) error {
	rawRecvs := make(map[component.ID]map[string]interface{})
	if err := conf.Unmarshal(&rawRecvs, confmap.WithErrorUnused()); err != nil {
		return err
	}

	// Prepare resulting map.
	r.recvs = make(map[component.ID]component.Config)

	// Iterate over input map and create a config for each.
	for id, value := range rawRecvs {
		// Find receiver factory based on "type" that we read from config source.
		factory := r.factories[id.Type()]
		if factory == nil {
			return errorUnknownType(receiversKeyName, id, reflect.ValueOf(r.factories).MapKeys())
		}

		// Create the default config for this receiver.
		receiverCfg := factory.CreateDefaultConfig()
		receiverCfg.SetIDName(id.Name()) //nolint:staticcheck

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := component.UnmarshalConfig(confmap.NewFromStringMap(value), receiverCfg); err != nil {
			return errorUnmarshalError(receiversKeyName, id, err)
		}

		r.recvs[id] = receiverCfg
	}

	return nil
}

func (r *Receivers) GetReceivers() map[component.ID]component.Config {
	return r.recvs
}
