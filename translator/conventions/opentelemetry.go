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
	AttributeServiceName           = "service.name"
	AttributeServiceNamespace      = "service.namespace"
	AttributeServiceInstance       = "service.instance.id"
	AttributeServiceVersion        = "service.version"
	AttributeTelemetrySDKName      = "telemetry.sdk.name"
	AttributeTelemetrySDKLanguage  = "telemetry.sdk.language"
	AttributeTelemetrySDKVersion   = "telemetry.sdk.version"
	AttributeTelemetryAutoVersion  = "telemetry.auto.version"
	AttributeContainerName         = "container.name"
	AttributeContainerID           = "container.id"
	AttributeContainerImage        = "container.image.name"
	AttributeContainerTag          = "container.image.tag"
	AttributeFaasName              = "faas.name"
	AttributeFaasID                = "faas.id"
	AttributeFaasVersion           = "faas.version"
	AttributeFaasInstance          = "faas.instance"
	AttributeK8sCluster            = "k8s.cluster.name"
	AttributeK8sNamespace          = "k8s.namespace.name"
	AttributeK8sContainer          = "k8s.container.name"
	AttributeK8sPod                = "k8s.pod.name"
	AttributeK8sPodUID             = "k8s.pod.uid"
	AttributeK8sCronJob            = "k8s.cronjob.name"
	AttributeK8sCronJobUID         = "k8s.cronjob.uid"
	AttributeK8sJob                = "k8s.job.name"
	AttributeK8sJobUID             = "k8s.job.uid"
	AttributeK8sDeployment         = "k8s.deployment.name"
	AttributeK8sDeploymentUID      = "k8s.deployment.uid"
	AttributeK8sDaemonSet          = "k8s.daemonset.name"
	AttributeK8sDaemonSetUID       = "k8s.daemonset.uid"
	AttributeK8sReplicaSet         = "k8s.replicaset.name"
	AttributeK8sReplicaSetUID      = "k8s.replicaset.uid"
	AttributeK8sStatefulSet        = "k8s.statefulset.name"
	AttributeK8sStatefulSetUID     = "k8s.statefulset.uid"
	AttributeHostHostname          = "host.hostname"
	AttributeHostID                = "host.id"
	AttributeHostName              = "host.name"
	AttributeHostType              = "host.type"
	AttributeHostImageName         = "host.image.name"
	AttributeHostImageID           = "host.image.id"
	AttributeHostImageVersion      = "host.image.version"
	AttributeProcessID             = "process.pid"
	AttributeProcessExecutableName = "process.executable.name"
	AttributeProcessExecutablePath = "process.executable.path"
	AttributeProcessCommand        = "process.command"
	AttributeProcessCommandLine    = "process.command_line"
	AttributeProcessOwner          = "process.owner"
	AttributeCloudProvider         = "cloud.provider"
	AttributeCloudAccount          = "cloud.account.id"
	AttributeCloudRegion           = "cloud.region"
	AttributeCloudZone             = "cloud.zone"
)

// GetResourceSemanticConventionAttributeNames a slice with all the Resource Semantic Conventions attribute names.
func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeServiceName,
		AttributeServiceNamespace,
		AttributeServiceInstance,
		AttributeServiceVersion,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKVersion,
		AttributeContainerName,
		AttributeContainerImage,
		AttributeContainerTag,
		AttributeFaasName,
		AttributeFaasID,
		AttributeFaasVersion,
		AttributeFaasInstance,
		AttributeK8sCluster,
		AttributeK8sNamespace,
		AttributeK8sPod,
		AttributeK8sDeployment,
		AttributeHostHostname,
		AttributeHostID,
		AttributeHostName,
		AttributeHostType,
		AttributeHostImageName,
		AttributeHostImageID,
		AttributeHostImageVersion,
		AttributeProcessID,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessCommand,
		AttributeProcessCommandLine,
		AttributeProcessOwner,
		AttributeCloudProvider,
		AttributeCloudAccount,
		AttributeCloudRegion,
		AttributeCloudZone,
	}
}

// OpenTelemetry Semantic Convention values for general Span attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/span-general.md
const (
	AttributeComponent    = "component"
	AttributeNetTransport = "net.transport"
	AttributeNetPeerIP    = "net.peer.ip"
	AttributeNetPeerPort  = "net.peer.port"
	AttributeNetPeerName  = "net.peer.name"
	AttributeNetHostIP    = "net.host.ip"
	AttributeNetHostPort  = "net.host.port"
	AttributeNetHostName  = "net.host.name"
	AttributeEnduserID    = "enduser.id"
	AttributeEnduserRole  = "enduser.role"
	AttributeEnduserScope = "enduser.scope"
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
	AttributeHTTPMethod                            = "http.method"
	AttributeHTTPURL                               = "http.url"
	AttributeHTTPTarget                            = "http.target"
	AttributeHTTPHost                              = "http.host"
	AttributeHTTPScheme                            = "http.scheme"
	AttributeHTTPStatusCode                        = "http.status_code"
	AttributeHTTPStatusText                        = "http.status_text"
	AttributeHTTPFlavor                            = "http.flavor"
	AttributeHTTPServerName                        = "http.server_name"
	AttributeHTTPHostName                          = "host.name"
	AttributeHTTPHostPort                          = "host.port"
	AttributeHTTPRoute                             = "http.route"
	AttributeHTTPClientIP                          = "http.client_ip"
	AttributeHTTPUserAgent                         = "http.user_agent"
	AttributeHTTPRequestContentLength              = "http.request_content_length"
	AttributeHTTPRequestContentLengthUncompressed  = "http.request_content_length_uncompressed"
	AttributeHTTPResponseContentLength             = "http.response_content_length"
	AttributeHTTPResponseContentLengthUncompressed = "http.response_content_length_uncompressed"
)

// OpenTelemetry Semantic Convention attribute names for database related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/database.md
const (
	AttributeDBSystem           = "db.system"
	AttributeDBConnectionString = "db.connection_string"
	AttributeDBUser             = "db.user"

	AttributeDBName      = "db.name"
	AttributeDBStatement = "db.statement"
	AttributeDBOperation = "db.operation"

	AttributeDBMsSQLInstanceName   = "db.mssql.instance_name"
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"
	AttributeDBCassandraKeyspace   = "db.cassandra.keyspace"
	AttributeDBHBaseNamespace      = "db.hbase.namespace"
	AttributeDBRedisDatabaseIndex  = "db.redis.database_index"
	AttributeDBMongoDBCollection   = "db.mongodb.collection"
)

// OpenTelemetry Semantic Convention attribute names for gRPC related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/rpc.md
const (
	AttributeRPCSystem               = "rpc.system"
	AttributeRPCService              = "rpc.service"
	AttributeRPCMethod               = "rpc.method"
	EventTypeMessage                 = "message"
	AttributeMessageType             = "message.type"
	MessageTypeReceived              = "RECEIVED"
	MessageTypeSent                  = "SENT"
	AttributeMessageID               = "message.id"
	AttributeMessageCompressedSize   = "message.compressed_size"
	AttributeMessageUncompressedSize = "message.uncompressed_size"
)

// OpenTelemetry Semantic Convention attribute names for FaaS related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/faas.md
const (
	AttributeFaaSTrigger            = "faas.trigger"
	AttributeFaaSExecution          = "faas.execution"
	AttributeFaaSDocumentCollection = "faas.document.collection"
	AttributeFaaSDocumentOperation  = "faas.document.operation"
	AttributeFaaSDocumentTime       = "faas.document.time"
	AttributeFaaSDocumentName       = "faas.document.name"
	AttributeFaaSTime               = "faas.time"
	AttributeFaaSCron               = "faas.cron"
	FaaSTriggerDataSource           = "datasource"
	FaaSTriggerHTTP                 = "http"
	FaaSTriggerPubSub               = "pubsub"
	FaaSTriggerTimer                = "timer"
	FaaSTriggerOther                = "other"
)

// OpenTelemetry Semantic Convention attribute names for messaging system related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/messaging.md
const (
	AttributeMessagingSystem                = "messaging.system"
	AttributeMessagingDestination           = "messaging.destination"
	AttributeMessagingDestinationKind       = "messaging.destination_kind"
	AttributeMessagingTempDestination       = "messaging.temp_destination"
	AttributeMessagingProtocol              = "messaging.protocol"
	AttributeMessagingProtocolVersion       = "messaging.protocol_version"
	AttributeMessagingURL                   = "messaging.url"
	AttributeMessagingMessageID             = "messaging.message_id"
	AttributeMessagingConversationID        = "messaging.conversation_id"
	AttributeMessagingPayloadSize           = "messaging.message_payload_size_bytes"
	AttributeMessagingPayloadCompressedSize = "messaging.message_payload_compressed_size_bytes"
	AttributeMessagingOperation             = "messaging.operation"
)

// OpenTelemetry Semantic Convention attribute names for exceptions
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/exceptions.md
const (
	AttributeExceptionEventName  = "exception"
	AttributeExceptionType       = "exception.type"
	AttributeExceptionMessage    = "exception.message"
	AttributeExceptionStacktrace = "exception.stacktrace"
)
