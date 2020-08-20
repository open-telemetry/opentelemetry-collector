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

package routingprocessor

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

// Config defines configuration for the Routing processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// DefaultExporters contains the list of exporters to use when a more specific record can't be found in the routing table.
	// Optional.
	DefaultExporters []string `mapstructure:"default_exporters"`

	// FromAttribute contains the attribute name to look up the route value. This attribute should be part of the context propagated
	// down from the previous receivers and/or processors. If all the receivers and processors are propagating the entire context correctly,
	// this could be the HTTP/gRPC header from the original request/RPC. Typically, aggregation processors (batch, queued_retry, groupbytrace)
	// will create a new context, so, those should be avoided when using this processor.Although the HTTP spec allows headers to be repeated,
	// this processor will only use the first value.
	// Required.
	FromAttribute string `mapstructure:"from_attribute"`

	// Table contains the routing table for this processor.
	// Required.
	Table []RoutingTableItem `mapstructure:"table"`
}

// RoutingTableItem specifies how data should be routed to the different exporters
type RoutingTableItem struct {
	// Value represents a possible value for the field specified under FromAttribute. Required.
	Value string `mapstructure:"value"`

	// Exporters contains the list of exporters to use when the value from the FromAttribute field matches this table item.
	// When no exporters are specified, the ones specified under DefaultExporters are used, if any.
	// The routing processor will fail upon the first failure from these exporters.
	// Optional.
	Exporters []string `mapstructure:"exporters"`
}
