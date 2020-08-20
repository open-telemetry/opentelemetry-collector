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

package zipkin

// These constants are the attribute keys used when translating from zipkin
// format to the internal collector data format.
const (
	LocalEndpointIPv4         = "ipv4"
	LocalEndpointIPv6         = "ipv6"
	LocalEndpointPort         = "port"
	LocalEndpointServiceName  = "serviceName"
	RemoteEndpointIPv4        = "zipkin.remoteEndpoint.ipv4"
	RemoteEndpointIPv6        = "zipkin.remoteEndpoint.ipv6"
	RemoteEndpointPort        = "zipkin.remoteEndpoint.port"
	RemoteEndpointServiceName = "zipkin.remoteEndpoint.serviceName"
	StartTimeAbsent           = "otel.zipkin.absentField.startTime"
)
