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

package componenttest

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// NopHost mocks a receiver.ReceiverHost for test purposes.
type NopHost struct {
}

var _ component.Host = (*NopHost)(nil)

// NewNopHost returns a new instance of NopHost with proper defaults for most
// tests.
func NewNopHost() component.Host {
	return &NopHost{}
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (nh *NopHost) ReportFatalError(_ error) {
	// Do nothing for now.
}

// GetFactory of the specified kind. Returns the factory for a component type.
func (nh *NopHost) GetFactory(_ component.Kind, _ configmodels.Type) component.Factory {
	return nil
}

func (nh *NopHost) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

func (nh *NopHost) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	return nil
}
