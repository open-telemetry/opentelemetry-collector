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

package configschema

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

const (
	receiver  = "receiver"
	extension = "extension"
	processor = "processor"
	exporter  = "exporter"
)

// CfgInfo contains a component config instance, as well as its group name and
// type.
type CfgInfo struct {
	// the name of the component group, e.g. "receiver"
	Group string
	// the component type, e.g. "otlpreceiver.Config"
	Type config.Type
	// an instance of the component's configuration struct
	CfgInstance interface{}
}

// GetAllCfgInfos accepts a Factories struct, then creates and returns a CfgInfo
// for each of its components.
func GetAllCfgInfos(components component.Factories) []CfgInfo {
	var out []CfgInfo
	for _, f := range components.Receivers {
		out = append(out, CfgInfo{
			Type:        f.Type(),
			Group:       receiver,
			CfgInstance: f.CreateDefaultConfig(),
		})
	}
	for _, f := range components.Extensions {
		out = append(out, CfgInfo{
			Type:        f.Type(),
			Group:       extension,
			CfgInstance: f.CreateDefaultConfig(),
		})
	}
	for _, f := range components.Processors {
		out = append(out, CfgInfo{
			Type:        f.Type(),
			Group:       processor,
			CfgInstance: f.CreateDefaultConfig(),
		})
	}
	for _, f := range components.Exporters {
		out = append(out, CfgInfo{
			Type:        f.Type(),
			Group:       exporter,
			CfgInstance: f.CreateDefaultConfig(),
		})
	}
	return out
}

// GetCfgInfo accepts a Factories struct, then creates and returns the default
// config for the component specified by the passed-in componentType and
// componentName.
func GetCfgInfo(components component.Factories, componentType, componentName string) (CfgInfo, error) {
	t := config.Type(componentName)
	switch componentType {
	case receiver:
		f := components.Receivers[t]
		if f == nil {
			return CfgInfo{}, fmt.Errorf("unknown %s name %q", componentType, componentName)
		}
		return CfgInfo{
			Type:        f.Type(),
			Group:       componentType,
			CfgInstance: f.CreateDefaultConfig(),
		}, nil
	case processor:
		f := components.Processors[t]
		if f == nil {
			return CfgInfo{}, fmt.Errorf("unknown %s name %q", componentType, componentName)
		}
		return CfgInfo{
			Type:        f.Type(),
			Group:       componentType,
			CfgInstance: f.CreateDefaultConfig(),
		}, nil
	case exporter:
		f := components.Exporters[t]
		if f == nil {
			return CfgInfo{}, fmt.Errorf("unknown %s name %q", componentType, componentName)
		}
		return CfgInfo{
			Type:        f.Type(),
			Group:       componentType,
			CfgInstance: f.CreateDefaultConfig(),
		}, nil
	case extension:
		f := components.Extensions[t]
		if f == nil {
			return CfgInfo{}, fmt.Errorf("unknown %s name %q", componentType, componentName)
		}
		return CfgInfo{
			Type:        f.Type(),
			Group:       componentType,
			CfgInstance: f.CreateDefaultConfig(),
		}, nil
	}
	return CfgInfo{}, fmt.Errorf("unknown component type %q", componentType)
}
