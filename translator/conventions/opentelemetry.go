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

package conventions

// OpenTelemetry Semantic Convention values for Resource attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-resource-semantic-conventions.md
const (
	ServiceNameAttribute      = "service.name"
	ServiceNamespaceAttribute = "service.namespace"
	ServiceInstanceAttribute  = "service.instance.id"
	ServiceVersionAttribute   = "service.version" // proposed
	LibraryNameAttribute      = "library.name"
	LibraryLanguageAttribute  = "library.language"
	LibraryVersionAttribute   = "library.version"
	ContainerNameAttribute    = "container.name"
	ContainerImageAttribute   = "container.image.name"
	ContainerTagAttribute     = "container.image.tag"
	K8sClusterAttribute       = "k8s.cluster.name"
	K8sNamespaceAttribute     = "k8s.namespace.name"
	K8sPodAttribute           = "k8s.pod.name"
	K8sDeploymentAttribute    = "k8s.deployment.name"
	HostHostnameAttribute     = "host.hostname"
	HostIDAttribute           = "host.id"
	HostNameAttribute         = "host.name"
	HostTypeAttribute         = "host.type"
	CloudProviderAttribute    = "cloud.provider"
	CloudAccountAttribute     = "cloud.account.id"
	CloudRegionAttribute      = "cloud.region"
	CloudZoneAttribute        = "cloud.zone"
)

// OpenTelemetry Semantic Convention values for general Span attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-semantic-conventions.md
// Peer attributes currently defined in database conventions
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-database.md
const (
	ComponentAttribute    = "component"
	PeerAddressAttribute  = "peer.address"  // to be obsoleted
	PeerHostAttribute     = "peer.hostname" // to be obsoleted
	PeerIpv4Attribute     = "peer.ipv4"     // to be obsoleted
	PeerIpv6Attribute     = "peer.ipv6"     // to be obsoleted
	PeerPortAttribute     = "peer.port"     // to be obsoleted
	PeerServiceAttribute  = "peer.service"  // to be obsoleted
	NetTransportAttribute = "net.transport" // pull request #349
	TCPTransportType      = "IP.TCP"
	UDPTransportType      = "IP.UDP"
	IPTransportType       = "IP"
	UnixTransportType     = "Unix"
	PipeTransportType     = "pipe"
	InProcTransportType   = "inproc"
	OtherTransportType    = "other"
	NetPeerIPAttribute    = "net.peer.ip"   // pull request #349
	NetPeerPortAttribute  = "net.peer.port" // pull request #349
	NetPeerName           = "net.peer.name" // pull request #349
	NetHostIP             = "net.host.ip"   // pull request #349
	NetHostPort           = "net.host.port" // pull request #349
	NetHostName           = "net.host.name" // pull request #349
	IAMUserAttribute      = "iam.user"      // proposed in issue #362
	IAMRoleAttribute      = "iam.role"      // proposed in issue #362
)

// OpenTelemetry Semantic Convention values for component attribute values.
// Possibly being removed due to issue #336
const (
	HTTPComponentType    = "http"
	GrpcComponentType    = "grpc"
	DbComponentType      = "db"        // proposed in issue #362
	MsgComponentType     = "messaging" // proposed in issue #362
	ClusterComponentType = "cluster"   // proposed in issue #362
)

// OpenTelemetry Semantic Convention attribute names for HTTP related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-http.md
const (
	HTTPMethodAttribute     = "http.method"
	HTTPURLAttribute        = "http.url"
	HTTPTargetAttribute     = "http.target"
	HTTPHostAttribute       = "http.host"
	HTTPSchemeAttribute     = "http.scheme"
	HTTPStatusCodeAttribute = "http.status_code"
	HTTPStatusTextAttribute = "http.status_text"
	HTTPFlavorAttribute     = "http.flavor"
	HTTPServerNameAttribute = "http.server_name"
	HTTPPortAttribute       = "http.port"
	HTTPRouteAttribute      = "http.route"
	HTTPClientIPAttribute   = "http.client_ip"
	HTTPUserAgentAttribute  = "http.user_agent" // proposed in issue #362, used by OpenCensus and OpenTelemetry
)

// OpenTelemetry Semantic Convention attribute names for database related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-database.md
const (
	DBTypeAttribute      = "db.type"
	DBInstanceAttribute  = "db.instance"
	DBStatementAttribute = "db.statement"
	DBUserAttribute      = "db.user"
	DBURLAttribute       = "db.url" // pull request #349
)

// OpenTelemetry Semantic Convention attribute names for gRPC related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-rpc.md
const (
	MessageEventType                 = "message"
	MessageTypeAttribute             = "message.type"
	ReceivedMessageType              = "RECEIVED"
	SentMessageType                  = "SENT"
	MessageIDAttribute               = "message.id"
	MessageCompressedSizeAttribute   = "message.compressed_size"
	MessageUncompressedSizeAttribute = "message.uncompressed_size"
)

// Proposed OpenTelemetry Semantic Convention attribute names for error/fault/exception related event attributes
// Derived from OpenTracing: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
const (
	ErrorEventType        = "error"
	ErrorKindAttribute    = "error.kind"
	ErrorObjectAttribute  = "error.object"
	ErrorMessageAttribute = "error.message"
	ErrorStackAttribute   = "error.stack"
)

// Proposed OpenTelemetry Semantic Convention attribute names for asynchronous messaging related attributes
// Derived from OpenTracing: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
const (
	MessageDestinatioAttribute = "message_bus.destination" // proposed in issue #362
)
