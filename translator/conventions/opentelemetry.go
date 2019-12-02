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
	AttributeServiceName      = "service.name"
	AttributeServiceNamespace = "service.namespace"
	AttributeServiceInstance  = "service.instance.id"
	AttributeServiceVersion   = "service.version" // proposed
	AttributeLibraryName      = "library.name"
	AttributeLibraryLanguage  = "library.language"
	AttributeLibraryVersion   = "library.version"
	AttributeContainerName    = "container.name"
	AttributeContainerImage   = "container.image.name"
	AttributeContainerTag     = "container.image.tag"
	AttributeK8sCluster       = "k8s.cluster.name"
	AttributeK8sNamespace     = "k8s.namespace.name"
	AttributeK8sPod           = "k8s.pod.name"
	AttributeK8sDeployment    = "k8s.deployment.name"
	AttributeHostHostname     = "host.hostname"
	AttributeHostID           = "host.id"
	AttributeHostName         = "host.name"
	AttributeHostType         = "host.type"
	AttributeCloudProvider    = "cloud.provider"
	AttributeCloudAccount     = "cloud.account.id"
	AttributeCloudRegion      = "cloud.region"
	AttributeCloudZone        = "cloud.zone"
)

// OpenTelemetry Semantic Convention values for general Span attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-semantic-conventions.md
// Peer attributes currently defined in database conventions
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-database.md
const (
	AttributeComponent    = "component"
	AttributePeerAddress  = "peer.address"  // to be obsoleted
	AttributePeerHost     = "peer.hostname" // to be obsoleted
	AttributePeerIpv4     = "peer.ipv4"     // to be obsoleted
	AttributePeerIpv6     = "peer.ipv6"     // to be obsoleted
	AttributePeerPort     = "peer.port"     // to be obsoleted
	AttributePeerService  = "peer.service"  // to be obsoleted
	AttributeNetTransport = "net.transport" // pull request #349
	TransportTypeTCP      = "IP.TCP"
	TransportTypeUDP      = "IP.UDP"
	TransportTypeIP       = "IP"
	TransportTypeUnix     = "Unix"
	TransportTypePipe     = "pipe"
	TransportTypeInProc   = "inproc"
	TransportTypeOther    = "other"
	AttributeNetPeerIP    = "net.peer.ip"   // pull request #349
	AttributeNetPeerPort  = "net.peer.port" // pull request #349
	AttributeNetPeerName  = "net.peer.name" // pull request #349
	AttributeNetHostIP    = "net.host.ip"   // pull request #349
	AttributeNetHostPort  = "net.host.port" // pull request #349
	AttributeNetHostName  = "net.host.name" // pull request #349
	AttributeIAMUser      = "iam.user"      // proposed in issue #362
	AttributeIAMRole      = "iam.role"      // proposed in issue #362
)

// OpenTelemetry Semantic Convention values for component attribute values.
// Possibly being removed due to issue #336
const (
	ComponentTypeHTTP    = "http"
	ComponentTypeGRPC    = "grpc"
	ComponentTypeDB      = "db"        // proposed in issue #362
	ComponentTypeMsg     = "messaging" // proposed in issue #362
	ComponentTypeCluster = "cluster"   // proposed in issue #362
)

// OpenTelemetry Semantic Convention attribute names for HTTP related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-http.md
const (
	AttributeHTTPMethod     = "http.method"
	AttributeHTTPURL        = "http.url"
	AttributeHTTPTarget     = "http.target"
	AttributeHTTPHost       = "http.host"
	AttributeHTTPScheme     = "http.scheme"
	AttributeHTTPStatusCode = "http.status_code"
	AttributeHTTPStatusText = "http.status_text"
	AttributeHTTPFlavor     = "http.flavor"
	AttributeHTTPServerName = "http.server_name"
	AttributeHTTPPort       = "http.port"
	AttributeHTTPRoute      = "http.route"
	AttributeHTTPClientIP   = "http.client_ip"
	AttributeHTTPUserAgent  = "http.user_agent" // proposed in issue #362, used by OpenCensus and OpenTelemetry
)

// OpenTelemetry Semantic Convention attribute names for database related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-database.md
const (
	AttributeDBType      = "db.type"
	AttributeDBInstance  = "db.instance"
	AttributeDBStatement = "db.statement"
	AttributeDBUser      = "db.user"
	AttributeDBURL       = "db.url" // pull request #349
)

// OpenTelemetry Semantic Convention attribute names for gRPC related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/data-rpc.md
const (
	EventTypeMessage                 = "message"
	AttributeMessageType             = "message.type"
	MessageTypeReceived              = "RECEIVED"
	MessageTypeSent                  = "SENT"
	AttributeMessageID               = "message.id"
	AttributeMessageCompressedSize   = "message.compressed_size"
	AttributeMessageUncompressedSize = "message.uncompressed_size"
	AttributeMessageContent          = "message.content" // proposed in issue #362
)

// Proposed OpenTelemetry Semantic Convention attribute names for error/fault/exception related event attributes
// Derived from OpenTracing: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
const (
	EventTypeError        = "error"
	AttributeErrorKind    = "error.kind"
	AttributeErrorObject  = "error.object"
	AttributeErrorMessage = "error.message"
	AttributeErrorStack   = "error.stack"
)

// Proposed OpenTelemetry Semantic Convention attribute names for asynchronous messaging related attributes
// Derived from OpenTracing: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
const (
	AttributeMessageDestination = "message_bus.destination" // proposed in issue #362
)
