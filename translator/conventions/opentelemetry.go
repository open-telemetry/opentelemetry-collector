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

package conventions

// OpenTelemetry Semantic Convention values for Resource attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/README.md
const (
	AttributeCloudAccount          = "cloud.account.id"
	AttributeCloudProvider         = "cloud.provider"
	AttributeCloudRegion           = "cloud.region"
	AttributeCloudZone             = "cloud.zone"
	AttributeContainerID           = "container.id"
	AttributeContainerImage        = "container.image.name"
	AttributeContainerName         = "container.name"
	AttributeContainerTag          = "container.image.tag"
	AttributeFaasID                = "faas.id"
	AttributeFaasInstance          = "faas.instance"
	AttributeFaasName              = "faas.name"
	AttributeFaasVersion           = "faas.version"
	AttributeHostHostname          = "host.hostname"
	AttributeHostID                = "host.id"
	AttributeHostImageID           = "host.image.id"
	AttributeHostImageName         = "host.image.name"
	AttributeHostImageVersion      = "host.image.version"
	AttributeHostName              = "host.name"
	AttributeHostType              = "host.type"
	AttributeK8sCluster            = "k8s.cluster.name"
	AttributeK8sContainer          = "k8s.container.name"
	AttributeK8sCronJob            = "k8s.cronjob.name"
	AttributeK8sCronJobUID         = "k8s.cronjob.uid"
	AttributeK8sDaemonSet          = "k8s.daemonset.name"
	AttributeK8sDaemonSetUID       = "k8s.daemonset.uid"
	AttributeK8sDeployment         = "k8s.deployment.name"
	AttributeK8sDeploymentUID      = "k8s.deployment.uid"
	AttributeK8sJob                = "k8s.job.name"
	AttributeK8sJobUID             = "k8s.job.uid"
	AttributeK8sNamespace          = "k8s.namespace.name"
	AttributeK8sPod                = "k8s.pod.name"
	AttributeK8sPodUID             = "k8s.pod.uid"
	AttributeK8sReplicaSet         = "k8s.replicaset.name"
	AttributeK8sReplicaSetUID      = "k8s.replicaset.uid"
	AttributeK8sStatefulSet        = "k8s.statefulset.name"
	AttributeK8sStatefulSetUID     = "k8s.statefulset.uid"
	AttributeProcessCommand        = "process.command"
	AttributeProcessCommandLine    = "process.command_line"
	AttributeProcessExecutableName = "process.executable.name"
	AttributeProcessExecutablePath = "process.executable.path"
	AttributeProcessID             = "process.pid"
	AttributeProcessOwner          = "process.owner"
	AttributeServiceInstance       = "service.instance.id"
	AttributeServiceName           = "service.name"
	AttributeServiceNamespace      = "service.namespace"
	AttributeServiceVersion        = "service.version"
	AttributeTelemetryAutoVersion  = "telemetry.auto.version"
	AttributeTelemetrySDKLanguage  = "telemetry.sdk.language"
	AttributeTelemetrySDKName      = "telemetry.sdk.name"
	AttributeTelemetrySDKVersion   = "telemetry.sdk.version"
)

// GetResourceSemanticConventionAttributeNames a slice with all the Resource Semantic Conventions attribute names.
func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeCloudAccount,
		AttributeCloudProvider,
		AttributeCloudRegion,
		AttributeCloudZone,
		AttributeContainerImage,
		AttributeContainerName,
		AttributeContainerTag,
		AttributeFaasID,
		AttributeFaasInstance,
		AttributeFaasName,
		AttributeFaasVersion,
		AttributeHostHostname,
		AttributeHostID,
		AttributeHostImageID,
		AttributeHostImageName,
		AttributeHostImageVersion,
		AttributeHostName,
		AttributeHostType,
		AttributeK8sCluster,
		AttributeK8sDeployment,
		AttributeK8sNamespace,
		AttributeK8sPod,
		AttributeProcessCommand,
		AttributeProcessCommandLine,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessID,
		AttributeProcessOwner,
		AttributeServiceInstance,
		AttributeServiceName,
		AttributeServiceNamespace,
		AttributeServiceVersion,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKVersion,
	}
}

// OpenTelemetry Semantic Convention values for general Span attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/span-general.md
const (
	AttributeComponent    = "component"
	AttributeEnduserID    = "enduser.id"
	AttributeEnduserRole  = "enduser.role"
	AttributeEnduserScope = "enduser.scope"
	AttributeNetHostIP    = "net.host.ip"
	AttributeNetHostName  = "net.host.name"
	AttributeNetHostPort  = "net.host.port"
	AttributeNetPeerIP    = "net.peer.ip"
	AttributeNetPeerName  = "net.peer.name"
	AttributeNetPeerPort  = "net.peer.port"
	AttributeNetTransport = "net.transport"
	AttributePeerService  = "peer.service"
)

// OpenTelemetry Semantic Convention values for component attribute values.
// Possibly being removed due to issue #336
const (
	ComponentTypeHTTP = "http"
	ComponentTypeGRPC = "grpc"
)

// OpenTelemetry Semantic Convention attribute names for HTTP related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md
const (
	AttributeHTTPClientIP                          = "http.client_ip"
	AttributeHTTPFlavor                            = "http.flavor"
	AttributeHTTPHost                              = "http.host"
	AttributeHTTPHostName                          = "host.name"
	AttributeHTTPHostPort                          = "host.port"
	AttributeHTTPMethod                            = "http.method"
	AttributeHTTPRequestContentLength              = "http.request_content_length"
	AttributeHTTPRequestContentLengthUncompressed  = "http.request_content_length_uncompressed"
	AttributeHTTPResponseContentLength             = "http.response_content_length"
	AttributeHTTPResponseContentLengthUncompressed = "http.response_content_length_uncompressed"
	AttributeHTTPRoute                             = "http.route"
	AttributeHTTPScheme                            = "http.scheme"
	AttributeHTTPServerName                        = "http.server_name"
	AttributeHTTPStatusCode                        = "http.status_code"
	AttributeHTTPStatusText                        = "http.status_text"
	AttributeHTTPTarget                            = "http.target"
	AttributeHTTPURL                               = "http.url"
	AttributeHTTPUserAgent                         = "http.user_agent"
)

// OpenTelemetry Semantic Convention attribute names for database related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md
const (
	AttributeDBConnectionString = "db.connection_string"

	AttributeDBCassandraKeyspace   = "db.cassandra.keyspace"
	AttributeDBHBaseNamespace      = "db.hbase.namespace"
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"
	AttributeDBMongoDBCollection   = "db.mongodb.collection"
	AttributeDBMsSQLInstanceName   = "db.mssql.instance_name"

	AttributeDBName               = "db.name"
	AttributeDBOperation          = "db.operation"
	AttributeDBRedisDatabaseIndex = "db.redis.database_index"
	AttributeDBStatement          = "db.statement"
	AttributeDBSystem             = "db.system"
	AttributeDBUser               = "db.user"
)

// OpenTelemetry Semantic Convention attribute names for gRPC related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/rpc.md
const (
	AttributeMessageCompressedSize   = "message.compressed_size"
	AttributeMessageID               = "message.id"
	AttributeMessageType             = "message.type"
	AttributeMessageUncompressedSize = "message.uncompressed_size"
	AttributeRPCMethod               = "rpc.method"
	AttributeRPCService              = "rpc.service"
	AttributeRPCSystem               = "rpc.system"
	EventTypeMessage                 = "message"
	MessageTypeReceived              = "RECEIVED"
	MessageTypeSent                  = "SENT"
)

// OpenTelemetry Semantic Convention attribute names for FaaS related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/faas.md
const (
	AttributeFaaSCron               = "faas.cron"
	AttributeFaaSDocumentCollection = "faas.document.collection"
	AttributeFaaSDocumentName       = "faas.document.name"
	AttributeFaaSDocumentOperation  = "faas.document.operation"
	AttributeFaaSDocumentTime       = "faas.document.time"
	AttributeFaaSExecution          = "faas.execution"
	AttributeFaaSTime               = "faas.time"
	AttributeFaaSTrigger            = "faas.trigger"
	FaaSTriggerDataSource           = "datasource"
	FaaSTriggerHTTP                 = "http"
	FaaSTriggerOther                = "other"
	FaaSTriggerPubSub               = "pubsub"
	FaaSTriggerTimer                = "timer"
)

// OpenTelemetry Semantic Convention attribute names for messaging system related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/messaging.md
const (
	AttributeMessagingConversationID        = "messaging.conversation_id"
	AttributeMessagingDestination           = "messaging.destination"
	AttributeMessagingDestinationKind       = "messaging.destination_kind"
	AttributeMessagingMessageID             = "messaging.message_id"
	AttributeMessagingOperation             = "messaging.operation"
	AttributeMessagingPayloadCompressedSize = "messaging.message_payload_compressed_size_bytes"
	AttributeMessagingPayloadSize           = "messaging.message_payload_size_bytes"
	AttributeMessagingProtocol              = "messaging.protocol"
	AttributeMessagingProtocolVersion       = "messaging.protocol_version"
	AttributeMessagingSystem                = "messaging.system"
	AttributeMessagingTempDestination       = "messaging.temp_destination"
	AttributeMessagingURL                   = "messaging.url"
)

// OpenTelemetry Semantic Convention attribute names for exceptions
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/exceptions.md
const (
	AttributeExceptionEventName  = "exception"
	AttributeExceptionMessage    = "exception.message"
	AttributeExceptionStacktrace = "exception.stacktrace"
	AttributeExceptionType       = "exception.type"
)
