// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// Attributes for Events represented using Log Records.
const (
	// Identifies the class / type of event.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'browser.mouse.click', 'device.app.lifecycle'
	// Note: Event names are subject to the same rules as attribute names. Notably,
	// event names are namespaced to avoid collisions and provide a clean separation
	// of semantics for events in separate domains like browser, mobile, and
	// kubernetes.
	AttributeEventName = "event.name"
)

// The attributes described in this section are rather generic. They may be
// used in any Log Record they apply to.
const (
	// A unique identifier for the Log Record.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '01ARZ3NDEKTSV4RRFFQ69G5FAV'
	// Note: If an id is provided, other log records with the same id will be
	// considered duplicates and can be removed safely. This means, that two
	// distinguishable log records MUST have different values.
	// The id MAY be an Universally Unique Lexicographically Sortable Identifier
	// (ULID), but other identifiers (e.g. UUID) may be used as needed.
	AttributeLogRecordUID = "log.record.uid"
)

// Describes Log attributes
const (
	// The stream associated with the log. See below for a list of well-known values.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeLogIostream = "log.iostream"
)

const (
	// Logs from stdout stream
	AttributeLogIostreamStdout = "stdout"
	// Events from stderr stream
	AttributeLogIostreamStderr = "stderr"
)

// A file to which log was emitted.
const (
	// The basename of the file.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'audit.log'
	AttributeLogFileName = "log.file.name"
	// The basename of the file, with symlinks resolved.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'uuid.log'
	AttributeLogFileNameResolved = "log.file.name_resolved"
	// The full path to the file.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/var/log/mysql/audit.log'
	AttributeLogFilePath = "log.file.path"
	// The full path to the file, with symlinks resolved.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/var/lib/docker/uuid.log'
	AttributeLogFilePathResolved = "log.file.path_resolved"
)

// Describes Database attributes
const (
	// The name of the connection pool; unique within the instrumented application. In
	// case the connection pool implementation doesn't provide a name, instrumentation
	// should use a combination of server.address and server.port attributes formatted
	// as server.address:server.port.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'myDataSource'
	AttributePoolName = "pool.name"
	// The state of a connection in the pool
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'idle'
	AttributeState = "state"
)

const (
	// idle
	AttributeStateIdle = "idle"
	// used
	AttributeStateUsed = "used"
)

// ASP.NET Core attributes
const (
	// Rate-limiting result, shows whether the lease was acquired or contains a
	// rejection reason
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: stable
	// Examples: 'acquired', 'request_canceled'
	AttributeAspnetcoreRateLimitingResult = "aspnetcore.rate_limiting.result"
	// Full type name of the IExceptionHandler implementation that handled the
	// exception.
	//
	// Type: string
	// Requirement Level: Conditionally Required - if and only if the exception was
	// handled by this handler.
	// Stability: stable
	// Examples: 'Contoso.MyHandler'
	AttributeAspnetcoreDiagnosticsHandlerType = "aspnetcore.diagnostics.handler.type"
	// Rate limiting policy name.
	//
	// Type: string
	// Requirement Level: Conditionally Required - if the matched endpoint for the
	// request had a rate-limiting policy.
	// Stability: stable
	// Examples: 'fixed', 'sliding', 'token'
	AttributeAspnetcoreRateLimitingPolicy = "aspnetcore.rate_limiting.policy"
	// Flag indicating if request was handled by the application pipeline.
	//
	// Type: boolean
	// Requirement Level: Conditionally Required - if and only if the request was not
	// handled.
	// Stability: stable
	// Examples: True
	AttributeAspnetcoreRequestIsUnhandled = "aspnetcore.request.is_unhandled"
	// A value that indicates whether the matched route is a fallback route.
	//
	// Type: boolean
	// Requirement Level: Conditionally Required - If and only if a route was
	// successfully matched.
	// Stability: stable
	// Examples: True
	AttributeAspnetcoreRoutingIsFallback = "aspnetcore.routing.is_fallback"
)

const (
	// Lease was acquired
	AttributeAspnetcoreRateLimitingResultAcquired = "acquired"
	// Lease request was rejected by the endpoint limiter
	AttributeAspnetcoreRateLimitingResultEndpointLimiter = "endpoint_limiter"
	// Lease request was rejected by the global limiter
	AttributeAspnetcoreRateLimitingResultGlobalLimiter = "global_limiter"
	// Lease request was canceled
	AttributeAspnetcoreRateLimitingResultRequestCanceled = "request_canceled"
)

// SignalR attributes
const (
	// SignalR HTTP connection closure status.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'app_shutdown', 'timeout'
	AttributeSignalrConnectionStatus = "signalr.connection.status"
	// SignalR transport type
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'web_sockets', 'long_polling'
	AttributeSignalrTransport = "signalr.transport"
)

const (
	// The connection was closed normally
	AttributeSignalrConnectionStatusNormalClosure = "normal_closure"
	// The connection was closed due to a timeout
	AttributeSignalrConnectionStatusTimeout = "timeout"
	// The connection was closed because the app is shutting down
	AttributeSignalrConnectionStatusAppShutdown = "app_shutdown"
)

const (
	// ServerSentEvents protocol
	AttributeSignalrTransportServerSentEvents = "server_sent_events"
	// LongPolling protocol
	AttributeSignalrTransportLongPolling = "long_polling"
	// WebSockets protocol
	AttributeSignalrTransportWebSockets = "web_sockets"
)

// Describes JVM buffer metric attributes.
const (
	// Name of the buffer pool.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'mapped', 'direct'
	// Note: Pool names are generally obtained via BufferPoolMXBean#getName().
	AttributeJvmBufferPoolName = "jvm.buffer.pool.name"
)

// Describes JVM memory metric attributes.
const (
	// Name of the memory pool.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: stable
	// Examples: 'G1 Old Gen', 'G1 Eden space', 'G1 Survivor Space'
	// Note: Pool names are generally obtained via MemoryPoolMXBean#getName().
	AttributeJvmMemoryPoolName = "jvm.memory.pool.name"
	// The type of memory.
	//
	// Type: Enum
	// Requirement Level: Recommended
	// Stability: stable
	// Examples: 'heap', 'non_heap'
	AttributeJvmMemoryType = "jvm.memory.type"
)

const (
	// Heap memory
	AttributeJvmMemoryTypeHeap = "heap"
	// Non-heap memory
	AttributeJvmMemoryTypeNonHeap = "non_heap"
)

// Attributes for process CPU metrics.
const (
	// The CPU state for this data point. A process SHOULD be characterized either by
	// data points with no state labels, or only data points with state labels.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeProcessCPUState = "process.cpu.state"
)

const (
	// system
	AttributeProcessCPUStateSystem = "system"
	// user
	AttributeProcessCPUStateUser = "user"
	// wait
	AttributeProcessCPUStateWait = "wait"
)

// Describes System metric attributes
const (
	// The device identifier
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '(identifier)'
	AttributeSystemDevice = "system.device"
)

// Describes System CPU metric attributes
const (
	// The logical CPU number [0..n-1]
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1
	AttributeSystemCPULogicalNumber = "system.cpu.logical_number"
	// The CPU state for this data point. A system's CPU SHOULD be characterized
	// either by data points with no state labels, or only data points with state
	// labels.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'idle', 'interrupt'
	AttributeSystemCPUState = "system.cpu.state"
)

const (
	// user
	AttributeSystemCPUStateUser = "user"
	// system
	AttributeSystemCPUStateSystem = "system"
	// nice
	AttributeSystemCPUStateNice = "nice"
	// idle
	AttributeSystemCPUStateIdle = "idle"
	// iowait
	AttributeSystemCPUStateIowait = "iowait"
	// interrupt
	AttributeSystemCPUStateInterrupt = "interrupt"
	// steal
	AttributeSystemCPUStateSteal = "steal"
)

// Describes System Memory metric attributes
const (
	// The memory state
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'free', 'cached'
	AttributeSystemMemoryState = "system.memory.state"
)

const (
	// used
	AttributeSystemMemoryStateUsed = "used"
	// free
	AttributeSystemMemoryStateFree = "free"
	// shared
	AttributeSystemMemoryStateShared = "shared"
	// buffers
	AttributeSystemMemoryStateBuffers = "buffers"
	// cached
	AttributeSystemMemoryStateCached = "cached"
)

// Describes System Memory Paging metric attributes
const (
	// The paging access direction
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'in'
	AttributeSystemPagingDirection = "system.paging.direction"
	// The memory paging state
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'free'
	AttributeSystemPagingState = "system.paging.state"
	// The memory paging type
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'minor'
	AttributeSystemPagingType = "system.paging.type"
)

const (
	// in
	AttributeSystemPagingDirectionIn = "in"
	// out
	AttributeSystemPagingDirectionOut = "out"
)

const (
	// used
	AttributeSystemPagingStateUsed = "used"
	// free
	AttributeSystemPagingStateFree = "free"
)

const (
	// major
	AttributeSystemPagingTypeMajor = "major"
	// minor
	AttributeSystemPagingTypeMinor = "minor"
)

// Describes Filesystem metric attributes
const (
	// The filesystem mode
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'rw, ro'
	AttributeSystemFilesystemMode = "system.filesystem.mode"
	// The filesystem mount path
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/mnt/data'
	AttributeSystemFilesystemMountpoint = "system.filesystem.mountpoint"
	// The filesystem state
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'used'
	AttributeSystemFilesystemState = "system.filesystem.state"
	// The filesystem type
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ext4'
	AttributeSystemFilesystemType = "system.filesystem.type"
)

const (
	// used
	AttributeSystemFilesystemStateUsed = "used"
	// free
	AttributeSystemFilesystemStateFree = "free"
	// reserved
	AttributeSystemFilesystemStateReserved = "reserved"
)

const (
	// fat32
	AttributeSystemFilesystemTypeFat32 = "fat32"
	// exfat
	AttributeSystemFilesystemTypeExfat = "exfat"
	// ntfs
	AttributeSystemFilesystemTypeNtfs = "ntfs"
	// refs
	AttributeSystemFilesystemTypeRefs = "refs"
	// hfsplus
	AttributeSystemFilesystemTypeHfsplus = "hfsplus"
	// ext4
	AttributeSystemFilesystemTypeExt4 = "ext4"
)

// Describes Network metric attributes
const (
	// A stateless protocol MUST NOT set this attribute
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'close_wait'
	AttributeSystemNetworkState = "system.network.state"
)

const (
	// close
	AttributeSystemNetworkStateClose = "close"
	// close_wait
	AttributeSystemNetworkStateCloseWait = "close_wait"
	// closing
	AttributeSystemNetworkStateClosing = "closing"
	// delete
	AttributeSystemNetworkStateDelete = "delete"
	// established
	AttributeSystemNetworkStateEstablished = "established"
	// fin_wait_1
	AttributeSystemNetworkStateFinWait1 = "fin_wait_1"
	// fin_wait_2
	AttributeSystemNetworkStateFinWait2 = "fin_wait_2"
	// last_ack
	AttributeSystemNetworkStateLastAck = "last_ack"
	// listen
	AttributeSystemNetworkStateListen = "listen"
	// syn_recv
	AttributeSystemNetworkStateSynRecv = "syn_recv"
	// syn_sent
	AttributeSystemNetworkStateSynSent = "syn_sent"
	// time_wait
	AttributeSystemNetworkStateTimeWait = "time_wait"
)

// Describes System Process metric attributes
const (
	// The process state, e.g., Linux Process State Codes
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'running'
	AttributeSystemProcessStatus = "system.process.status"
)

const (
	// running
	AttributeSystemProcessStatusRunning = "running"
	// sleeping
	AttributeSystemProcessStatusSleeping = "sleeping"
	// stopped
	AttributeSystemProcessStatusStopped = "stopped"
	// defunct
	AttributeSystemProcessStatusDefunct = "defunct"
)

// The Android platform on which the Android application is running.
const (
	// Uniquely identifies the framework API revision offered by a version
	// (os.version) of the android operating system. More information can be found
	// here.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '33', '32'
	AttributeAndroidOSAPILevel = "android.os.api_level"
)

// Attributes for AWS DynamoDB.
const (
	// The JSON-serialized value of each item in the AttributeDefinitions request
	// field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "AttributeName": "string", "AttributeType": "string" }'
	AttributeAWSDynamoDBAttributeDefinitions = "aws.dynamodb.attribute_definitions"
	// The value of the AttributesToGet request parameter.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'lives', 'id'
	AttributeAWSDynamoDBAttributesToGet = "aws.dynamodb.attributes_to_get"
	// The value of the ConsistentRead request parameter.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeAWSDynamoDBConsistentRead = "aws.dynamodb.consistent_read"
	// The JSON-serialized value of each item in the ConsumedCapacity response field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "CapacityUnits": number, "GlobalSecondaryIndexes": { "string" : {
	// "CapacityUnits": number, "ReadCapacityUnits": number, "WriteCapacityUnits":
	// number } }, "LocalSecondaryIndexes": { "string" : { "CapacityUnits": number,
	// "ReadCapacityUnits": number, "WriteCapacityUnits": number } },
	// "ReadCapacityUnits": number, "Table": { "CapacityUnits": number,
	// "ReadCapacityUnits": number, "WriteCapacityUnits": number }, "TableName":
	// "string", "WriteCapacityUnits": number }'
	AttributeAWSDynamoDBConsumedCapacity = "aws.dynamodb.consumed_capacity"
	// The value of the Count response parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 10
	AttributeAWSDynamoDBCount = "aws.dynamodb.count"
	// The value of the ExclusiveStartTableName request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Users', 'CatsTable'
	AttributeAWSDynamoDBExclusiveStartTable = "aws.dynamodb.exclusive_start_table"
	// The JSON-serialized value of each item in the GlobalSecondaryIndexUpdates
	// request field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "Create": { "IndexName": "string", "KeySchema": [ {
	// "AttributeName": "string", "KeyType": "string" } ], "Projection": {
	// "NonKeyAttributes": [ "string" ], "ProjectionType": "string" },
	// "ProvisionedThroughput": { "ReadCapacityUnits": number, "WriteCapacityUnits":
	// number } }'
	AttributeAWSDynamoDBGlobalSecondaryIndexUpdates = "aws.dynamodb.global_secondary_index_updates"
	// The JSON-serialized value of each item of the GlobalSecondaryIndexes request
	// field
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "IndexName": "string", "KeySchema": [ { "AttributeName": "string",
	// "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ],
	// "ProjectionType": "string" }, "ProvisionedThroughput": { "ReadCapacityUnits":
	// number, "WriteCapacityUnits": number } }'
	AttributeAWSDynamoDBGlobalSecondaryIndexes = "aws.dynamodb.global_secondary_indexes"
	// The value of the IndexName request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'name_to_group'
	AttributeAWSDynamoDBIndexName = "aws.dynamodb.index_name"
	// The JSON-serialized value of the ItemCollectionMetrics response field.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "string" : [ { "ItemCollectionKey": { "string" : { "B": blob,
	// "BOOL": boolean, "BS": [ blob ], "L": [ "AttributeValue" ], "M": { "string" :
	// "AttributeValue" }, "N": "string", "NS": [ "string" ], "NULL": boolean, "S":
	// "string", "SS": [ "string" ] } }, "SizeEstimateRangeGB": [ number ] } ] }'
	AttributeAWSDynamoDBItemCollectionMetrics = "aws.dynamodb.item_collection_metrics"
	// The value of the Limit request parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 10
	AttributeAWSDynamoDBLimit = "aws.dynamodb.limit"
	// The JSON-serialized value of each item of the LocalSecondaryIndexes request
	// field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "IndexARN": "string", "IndexName": "string", "IndexSizeBytes":
	// number, "ItemCount": number, "KeySchema": [ { "AttributeName": "string",
	// "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ],
	// "ProjectionType": "string" } }'
	AttributeAWSDynamoDBLocalSecondaryIndexes = "aws.dynamodb.local_secondary_indexes"
	// The value of the ProjectionExpression request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Title', 'Title, Price, Color', 'Title, Description, RelatedItems,
	// ProductReviews'
	AttributeAWSDynamoDBProjection = "aws.dynamodb.projection"
	// The value of the ProvisionedThroughput.ReadCapacityUnits request parameter.
	//
	// Type: double
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1.0, 2.0
	AttributeAWSDynamoDBProvisionedReadCapacity = "aws.dynamodb.provisioned_read_capacity"
	// The value of the ProvisionedThroughput.WriteCapacityUnits request parameter.
	//
	// Type: double
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1.0, 2.0
	AttributeAWSDynamoDBProvisionedWriteCapacity = "aws.dynamodb.provisioned_write_capacity"
	// The value of the ScanIndexForward request parameter.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeAWSDynamoDBScanForward = "aws.dynamodb.scan_forward"
	// The value of the ScannedCount response parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 50
	AttributeAWSDynamoDBScannedCount = "aws.dynamodb.scanned_count"
	// The value of the Segment request parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 10
	AttributeAWSDynamoDBSegment = "aws.dynamodb.segment"
	// The value of the Select request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ALL_ATTRIBUTES', 'COUNT'
	AttributeAWSDynamoDBSelect = "aws.dynamodb.select"
	// The number of items in the TableNames response parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 20
	AttributeAWSDynamoDBTableCount = "aws.dynamodb.table_count"
	// The keys in the RequestItems object field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Users', 'Cats'
	AttributeAWSDynamoDBTableNames = "aws.dynamodb.table_names"
	// The value of the TotalSegments request parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 100
	AttributeAWSDynamoDBTotalSegments = "aws.dynamodb.total_segments"
)

// The web browser attributes
const (
	// Array of brand name and version separated by a space
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: ' Not A;Brand 99', 'Chromium 99', 'Chrome 99'
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.brands).
	AttributeBrowserBrands = "browser.brands"
	// Preferred language of the user using the browser
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'en', 'en-US', 'fr', 'fr-FR'
	// Note: This value is intended to be taken from the Navigator API
	// navigator.language.
	AttributeBrowserLanguage = "browser.language"
	// A boolean that is true if the browser is running on a mobile device
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.mobile). If unavailable, this attribute SHOULD be left
	// unset.
	AttributeBrowserMobile = "browser.mobile"
	// The platform on which the browser is running
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Windows', 'macOS', 'Android'
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.platform). If unavailable, the legacy
	// navigator.platform API SHOULD NOT be used instead and this attribute SHOULD be
	// left unset in order for the values to be consistent.
	// The list of possible values is defined in the W3C User-Agent Client Hints
	// specification. Note that some (but not all) of these values can overlap with
	// values in the os.type and os.name attributes. However, for consistency, the
	// values in the browser.platform attribute should capture the exact value that
	// the user agent provides.
	AttributeBrowserPlatform = "browser.platform"
)

// These attributes may be used to describe the client in a connection-based
// network interaction where there is one side that initiates the connection
// (the client is the side that initiates the connection). This covers all TCP
// network interactions since TCP is connection-based and one side initiates
// the connection (an exception is made for peer-to-peer communication over TCP
// where the "user-facing" surface of the protocol / API doesn't expose a clear
// notion of client and server). This also covers UDP network interactions
// where one side initiates the interaction, e.g. QUIC (HTTP/3) and DNS.
const (
	// Client address - domain name if available without reverse DNS lookup;
	// otherwise, IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'client.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.address SHOULD represent the client address behind any
	// intermediaries,  for example proxies, if it's available.
	AttributeClientAddress = "client.address"
	// Client port number.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 65123
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.port SHOULD represent the client port behind any
	// intermediaries,  for example proxies, if it's available.
	AttributeClientPort = "client.port"
)

// A cloud environment (e.g. GCP, Azure, AWS).
const (
	// The cloud account ID the resource is assigned to.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '111111111111', 'opentelemetry'
	AttributeCloudAccountID = "cloud.account.id"
	// Cloud regions often have multiple, isolated locations known as zones to
	// increase availability. Availability zone represents the zone where the resource
	// is running.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'us-east-1c'
	// Note: Availability zones are called &quot;zones&quot; on Alibaba Cloud and
	// Google Cloud.
	AttributeCloudAvailabilityZone = "cloud.availability_zone"
	// The cloud platform in use.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: The prefix of the service SHOULD match the one specified in
	// cloud.provider.
	AttributeCloudPlatform = "cloud.platform"
	// Name of the cloud provider.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeCloudProvider = "cloud.provider"
	// The geographical region the resource is running.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'us-central1', 'us-east-1'
	// Note: Refer to your provider's docs to see the available regions, for example
	// Alibaba Cloud regions, AWS regions, Azure regions, Google Cloud regions, or
	// Tencent Cloud regions.
	AttributeCloudRegion = "cloud.region"
	// Cloud provider-specific native identifier of the monitored cloud resource (e.g.
	// an ARN on AWS, a fully qualified resource ID on Azure, a full resource name on
	// GCP)
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:lambda:REGION:ACCOUNT_ID:function:my-function', '//run.googl
	// eapis.com/projects/PROJECT_ID/locations/LOCATION_ID/services/SERVICE_ID', '/sub
	// scriptions/<SUBSCIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/sites
	// /<FUNCAPP>/functions/<FUNC>'
	// Note: On some cloud providers, it may not be possible to determine the full ID
	// at startup,
	// so it may be necessary to set cloud.resource_id as a span attribute instead.The
	// exact value to use for cloud.resource_id depends on the cloud provider.
	// The following well-known definitions MUST be used if you set this attribute and
	// they apply:<ul>
	// <li>AWS Lambda: The function ARN.
	// Take care not to use the &quot;invoked ARN&quot; directly but replace any
	// alias suffix
	// with the resolved function version, as the same runtime instance may be
	// invokable with
	// multiple different aliases.</li>
	// <li>GCP: The URI of the resource</li>
	// <li>Azure: The Fully Qualified Resource ID of the invoked function,
	// not the function app, having the form
	// /subscriptions/<SUBSCIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/s
	// ites/<FUNCAPP>/functions/<FUNC>.
	// This means that a span attribute MUST be used, as an Azure function app can
	// host multiple functions that would usually share
	// a TracerProvider.</li>
	// </ul>
	AttributeCloudResourceID = "cloud.resource_id"
)

const (
	// Alibaba Cloud Elastic Compute Service
	AttributeCloudPlatformAlibabaCloudECS = "alibaba_cloud_ecs"
	// Alibaba Cloud Function Compute
	AttributeCloudPlatformAlibabaCloudFc = "alibaba_cloud_fc"
	// Red Hat OpenShift on Alibaba Cloud
	AttributeCloudPlatformAlibabaCloudOpenshift = "alibaba_cloud_openshift"
	// AWS Elastic Compute Cloud
	AttributeCloudPlatformAWSEC2 = "aws_ec2"
	// AWS Elastic Container Service
	AttributeCloudPlatformAWSECS = "aws_ecs"
	// AWS Elastic Kubernetes Service
	AttributeCloudPlatformAWSEKS = "aws_eks"
	// AWS Lambda
	AttributeCloudPlatformAWSLambda = "aws_lambda"
	// AWS Elastic Beanstalk
	AttributeCloudPlatformAWSElasticBeanstalk = "aws_elastic_beanstalk"
	// AWS App Runner
	AttributeCloudPlatformAWSAppRunner = "aws_app_runner"
	// Red Hat OpenShift on AWS (ROSA)
	AttributeCloudPlatformAWSOpenshift = "aws_openshift"
	// Azure Virtual Machines
	AttributeCloudPlatformAzureVM = "azure_vm"
	// Azure Container Apps
	AttributeCloudPlatformAzureContainerApps = "azure_container_apps"
	// Azure Container Instances
	AttributeCloudPlatformAzureContainerInstances = "azure_container_instances"
	// Azure Kubernetes Service
	AttributeCloudPlatformAzureAKS = "azure_aks"
	// Azure Functions
	AttributeCloudPlatformAzureFunctions = "azure_functions"
	// Azure App Service
	AttributeCloudPlatformAzureAppService = "azure_app_service"
	// Azure Red Hat OpenShift
	AttributeCloudPlatformAzureOpenshift = "azure_openshift"
	// Google Bare Metal Solution (BMS)
	AttributeCloudPlatformGCPBareMetalSolution = "gcp_bare_metal_solution"
	// Google Cloud Compute Engine (GCE)
	AttributeCloudPlatformGCPComputeEngine = "gcp_compute_engine"
	// Google Cloud Run
	AttributeCloudPlatformGCPCloudRun = "gcp_cloud_run"
	// Google Cloud Kubernetes Engine (GKE)
	AttributeCloudPlatformGCPKubernetesEngine = "gcp_kubernetes_engine"
	// Google Cloud Functions (GCF)
	AttributeCloudPlatformGCPCloudFunctions = "gcp_cloud_functions"
	// Google Cloud App Engine (GAE)
	AttributeCloudPlatformGCPAppEngine = "gcp_app_engine"
	// Red Hat OpenShift on Google Cloud
	AttributeCloudPlatformGCPOpenshift = "gcp_openshift"
	// Red Hat OpenShift on IBM Cloud
	AttributeCloudPlatformIbmCloudOpenshift = "ibm_cloud_openshift"
	// Tencent Cloud Cloud Virtual Machine (CVM)
	AttributeCloudPlatformTencentCloudCvm = "tencent_cloud_cvm"
	// Tencent Cloud Elastic Kubernetes Service (EKS)
	AttributeCloudPlatformTencentCloudEKS = "tencent_cloud_eks"
	// Tencent Cloud Serverless Cloud Function (SCF)
	AttributeCloudPlatformTencentCloudScf = "tencent_cloud_scf"
)

const (
	// Alibaba Cloud
	AttributeCloudProviderAlibabaCloud = "alibaba_cloud"
	// Amazon Web Services
	AttributeCloudProviderAWS = "aws"
	// Microsoft Azure
	AttributeCloudProviderAzure = "azure"
	// Google Cloud Platform
	AttributeCloudProviderGCP = "gcp"
	// Heroku Platform as a Service
	AttributeCloudProviderHeroku = "heroku"
	// IBM Cloud
	AttributeCloudProviderIbmCloud = "ibm_cloud"
	// Tencent Cloud
	AttributeCloudProviderTencentCloud = "tencent_cloud"
)

// Attributes for CloudEvents.
const (
	// The event_id uniquely identifies the event.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '123e4567-e89b-12d3-a456-426614174000', '0001'
	AttributeCloudeventsEventID = "cloudevents.event_id"
	// The source identifies the context in which an event happened.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'https://github.com/cloudevents', '/cloudevents/spec/pull/123', 'my-
	// service'
	AttributeCloudeventsEventSource = "cloudevents.event_source"
	// The version of the CloudEvents specification which the event uses.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1.0'
	AttributeCloudeventsEventSpecVersion = "cloudevents.event_spec_version"
	// The subject of the event in the context of the event producer (identified by
	// source).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'mynewfile.jpg'
	AttributeCloudeventsEventSubject = "cloudevents.event_subject"
	// The event_type contains a value describing the type of event related to the
	// originating occurrence.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'com.github.pull_request.opened', 'com.example.object.deleted.v2'
	AttributeCloudeventsEventType = "cloudevents.event_type"
)

// These attributes allow to report this unit of code and therefore to provide
// more context about the span.
const (
	// The column number in code.filepath best representing the operation. It SHOULD
	// point within the code unit named in code.function.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 16
	AttributeCodeColumn = "code.column"
	// The source code file name that identifies the code unit as uniquely as possible
	// (preferably an absolute file path).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/usr/local/MyApplication/content_root/app/index.php'
	AttributeCodeFilepath = "code.filepath"
	// The method or function name, or equivalent (usually rightmost part of the code
	// unit's name).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'serveRequest'
	AttributeCodeFunction = "code.function"
	// The line number in code.filepath best representing the operation. It SHOULD
	// point within the code unit named in code.function.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 42
	AttributeCodeLineNumber = "code.lineno"
	// The &quot;namespace&quot; within which code.function is defined. Usually the
	// qualified class or module name, such that code.namespace + some separator +
	// code.function form a unique identifier for the code unit.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'com.example.MyHTTPService'
	AttributeCodeNamespace = "code.namespace"
	// A stacktrace as a string in the natural representation for the language
	// runtime. The representation is to be determined and documented by each language
	// SIG.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at '
	//  'com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at '
	//  'com.example.GenerateTrace.main(GenerateTrace.java:5)'
	AttributeCodeStacktrace = "code.stacktrace"
)

// A container instance.
const (
	// The command used to run the container (i.e. the command name).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol'
	// Note: If using embedded credentials or sensitive data, it is recommended to
	// remove them to prevent potential leakage.
	AttributeContainerCommand = "container.command"
	// All the command arguments (including the command/executable itself) run by the
	// container. [2]
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol, --config, config.yaml'
	AttributeContainerCommandArgs = "container.command_args"
	// The full command run by the container as a single string representing the full
	// command. [2]
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol --config config.yaml'
	AttributeContainerCommandLine = "container.command_line"
	// The CPU state for this data point.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'user', 'kernel'
	AttributeContainerCPUState = "container.cpu.state"
	// Container ID. Usually a UUID, as for example used to identify Docker
	// containers. The UUID might be abbreviated.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'a3bf90e006b2'
	AttributeContainerID = "container.id"
	// Runtime specific image identifier. Usually a hash algorithm followed by a UUID.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples:
	// 'sha256:19c92d0a00d1b66d897bceaa7319bee0dd38a10a851c60bcec9474aa3f01e50f'
	// Note: Docker defines a sha256 of the image id; container.image.id corresponds
	// to the Image field from the Docker container inspect API endpoint.
	// K8S defines a link to the container registry repository with digest "imageID":
	// "registry.azurecr.io /namespace/service/dockerfile@sha256:bdeabd40c3a8a492eaf9e
	// 8e44d0ebbb84bac7ee25ac0cf8a7159d25f62555625".
	// The ID is assinged by the container runtime and can vary in different
	// environments. Consider using oci.manifest.digest if it is important to identify
	// the same image in different environments/runtimes.
	AttributeContainerImageID = "container.image.id"
	// Name of the image the container was built on.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'gcr.io/opentelemetry/operator'
	AttributeContainerImageName = "container.image.name"
	// Repo digests of the container image as provided by the container runtime.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'example@sha256:afcc7f1ac1b49db317a7196c902e61c6c3c4607d63599ee1a82d7
	// 02d249a0ccb', 'internal.registry.example.com:5000/example@sha256:b69959407d21e8
	// a062e0416bf13405bb2b71ed7a84dde4158ebafacfa06f5578'
	// Note: Docker and CRI report those under the RepoDigests field.
	AttributeContainerImageRepoDigests = "container.image.repo_digests"
	// Container image tags. An example can be found in Docker Image Inspect. Should
	// be only the <tag> section of the full name for example from
	// registry.example.com/my-org/my-image:<tag>.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'v1.27.1', '3.5.7-0'
	AttributeContainerImageTags = "container.image.tags"
	// Container name used by container runtime.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-autoconf'
	AttributeContainerName = "container.name"
	// The container runtime managing this container.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'docker', 'containerd', 'rkt'
	AttributeContainerRuntime = "container.runtime"
)

const (
	// When tasks of the cgroup are in user mode (Linux). When all container processes are in user mode (Windows)
	AttributeContainerCPUStateUser = "user"
	// When CPU is used by the system (host OS)
	AttributeContainerCPUStateSystem = "system"
	// When tasks of the cgroup are in kernel mode (Linux). When all container processes are in kernel mode (Windows)
	AttributeContainerCPUStateKernel = "kernel"
)

// The attributes used to describe telemetry in the context of databases.
const (
	// The consistency level of the query. Based on consistency values from CQL.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBCassandraConsistencyLevel = "db.cassandra.consistency_level"
	// The data center of the coordinating node for a query.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'us-west-2'
	AttributeDBCassandraCoordinatorDC = "db.cassandra.coordinator.dc"
	// The ID of the coordinating node for a query.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'be13faa2-8574-4d71-926d-27f16cf8a7af'
	AttributeDBCassandraCoordinatorID = "db.cassandra.coordinator.id"
	// Whether or not the query is idempotent.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBCassandraIdempotence = "db.cassandra.idempotence"
	// The fetch size used for paging, i.e. how many rows will be returned at once.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 5000
	AttributeDBCassandraPageSize = "db.cassandra.page_size"
	// The number of times a query was speculatively executed. Not set or 0 if the
	// query was not executed speculatively.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 2
	AttributeDBCassandraSpeculativeExecutionCount = "db.cassandra.speculative_execution_count"
	// The name of the primary Cassandra table that the operation is acting upon,
	// including the keyspace name (if applicable).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'mytable'
	// Note: This mirrors the db.sql.table attribute but references cassandra rather
	// than sql. It is not recommended to attempt any client-side parsing of
	// db.statement just to get this property, but it should be set if it is provided
	// by the library being instrumented. If the operation is acting upon an anonymous
	// table, or more than one table, this value MUST NOT be set.
	AttributeDBCassandraTable = "db.cassandra.table"
	// Unique Cosmos client instance id.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '3ba4827d-4422-483f-b59f-85b74211c11d'
	AttributeDBCosmosDBClientID = "db.cosmosdb.client_id"
	// Cosmos client connection mode.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBCosmosDBConnectionMode = "db.cosmosdb.connection_mode"
	// Cosmos DB container name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'anystring'
	AttributeDBCosmosDBContainer = "db.cosmosdb.container"
	// CosmosDB Operation Type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBCosmosDBOperationType = "db.cosmosdb.operation_type"
	// RU consumed for that operation
	//
	// Type: double
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 46.18, 1.0
	AttributeDBCosmosDBRequestCharge = "db.cosmosdb.request_charge"
	// Request payload size in bytes
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBCosmosDBRequestContentLength = "db.cosmosdb.request_content_length"
	// Cosmos DB status code.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 200, 201
	AttributeDBCosmosDBStatusCode = "db.cosmosdb.status_code"
	// Cosmos DB sub status code.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1000, 1002
	AttributeDBCosmosDBSubStatusCode = "db.cosmosdb.sub_status_code"
	// Represents the identifier of an Elasticsearch cluster.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'e9106fc68e3044f0b1475b04bf4ffd5f'
	AttributeDBElasticsearchClusterName = "db.elasticsearch.cluster.name"
	// An identifier (address, unique name, or any other identifier) of the database
	// instance that is executing queries or mutations on the current connection. This
	// is useful in cases where the database is running in a clustered environment and
	// the instrumentation is able to record the node executing the query. The client
	// may obtain this value in databases like MySQL using queries like select
	// @@hostname.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'mysql-e26b99z.example.com'
	AttributeDBInstanceID = "db.instance.id"
	// The MongoDB collection being accessed within the database stated in db.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'customers', 'products'
	AttributeDBMongoDBCollection = "db.mongodb.collection"
	// The Microsoft SQL Server instance name connecting to. This name is used to
	// determine the port of a named instance.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MSSQLSERVER'
	// Note: If setting a db.mssql.instance_name, server.port is no longer required
	// (but still recommended if non-standard).
	AttributeDBMSSQLInstanceName = "db.mssql.instance_name"
	// This attribute is used to report the name of the database being accessed. For
	// commands that switch the database, this should be set to the target database
	// (even if the command fails).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'customers', 'main'
	// Note: In some SQL databases, the database name to be used is called
	// &quot;schema name&quot;. In case there are multiple layers that could be
	// considered for database name (e.g. Oracle instance name and schema name), the
	// database name to be used is the more specific layer (e.g. Oracle schema name).
	AttributeDBName = "db.name"
	// The name of the operation being executed, e.g. the MongoDB command name such as
	// findAndModify, or the SQL keyword.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'findAndModify', 'HMSET', 'SELECT'
	// Note: When setting this to an SQL keyword, it is not recommended to attempt any
	// client-side parsing of db.statement just to get this property, but it should be
	// set if the operation name is provided by the library being instrumented. If the
	// SQL statement has an ambiguous operation, or performs more than one operation,
	// this value may be omitted.
	AttributeDBOperation = "db.operation"
	// The index of the database being accessed as used in the SELECT command,
	// provided as an integer. To be used instead of the generic db.name attribute.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 1, 15
	AttributeDBRedisDBIndex = "db.redis.database_index"
	// The name of the primary table that the operation is acting upon, including the
	// database name (if applicable).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'public.users', 'customers'
	// Note: It is not recommended to attempt any client-side parsing of db.statement
	// just to get this property, but it should be set if it is provided by the
	// library being instrumented. If the operation is acting upon an anonymous table,
	// or more than one table, this value MUST NOT be set.
	AttributeDBSQLTable = "db.sql.table"
	// The database statement being executed.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'SELECT * FROM wuser_table', 'SET mykey "WuValue"'
	AttributeDBStatement = "db.statement"
	// An identifier for the database management system (DBMS) product being used. See
	// below for a list of well-known identifiers.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeDBSystem = "db.system"
	// Username for accessing the database.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'readonly_user', 'reporting_user'
	AttributeDBUser = "db.user"
)

const (
	// all
	AttributeDBCassandraConsistencyLevelAll = "all"
	// each_quorum
	AttributeDBCassandraConsistencyLevelEachQuorum = "each_quorum"
	// quorum
	AttributeDBCassandraConsistencyLevelQuorum = "quorum"
	// local_quorum
	AttributeDBCassandraConsistencyLevelLocalQuorum = "local_quorum"
	// one
	AttributeDBCassandraConsistencyLevelOne = "one"
	// two
	AttributeDBCassandraConsistencyLevelTwo = "two"
	// three
	AttributeDBCassandraConsistencyLevelThree = "three"
	// local_one
	AttributeDBCassandraConsistencyLevelLocalOne = "local_one"
	// any
	AttributeDBCassandraConsistencyLevelAny = "any"
	// serial
	AttributeDBCassandraConsistencyLevelSerial = "serial"
	// local_serial
	AttributeDBCassandraConsistencyLevelLocalSerial = "local_serial"
)

const (
	// Gateway (HTTP) connections mode
	AttributeDBCosmosDBConnectionModeGateway = "gateway"
	// Direct connection
	AttributeDBCosmosDBConnectionModeDirect = "direct"
)

const (
	// invalid
	AttributeDBCosmosDBOperationTypeInvalid = "Invalid"
	// create
	AttributeDBCosmosDBOperationTypeCreate = "Create"
	// patch
	AttributeDBCosmosDBOperationTypePatch = "Patch"
	// read
	AttributeDBCosmosDBOperationTypeRead = "Read"
	// read_feed
	AttributeDBCosmosDBOperationTypeReadFeed = "ReadFeed"
	// delete
	AttributeDBCosmosDBOperationTypeDelete = "Delete"
	// replace
	AttributeDBCosmosDBOperationTypeReplace = "Replace"
	// execute
	AttributeDBCosmosDBOperationTypeExecute = "Execute"
	// query
	AttributeDBCosmosDBOperationTypeQuery = "Query"
	// head
	AttributeDBCosmosDBOperationTypeHead = "Head"
	// head_feed
	AttributeDBCosmosDBOperationTypeHeadFeed = "HeadFeed"
	// upsert
	AttributeDBCosmosDBOperationTypeUpsert = "Upsert"
	// batch
	AttributeDBCosmosDBOperationTypeBatch = "Batch"
	// query_plan
	AttributeDBCosmosDBOperationTypeQueryPlan = "QueryPlan"
	// execute_javascript
	AttributeDBCosmosDBOperationTypeExecuteJavascript = "ExecuteJavaScript"
)

const (
	// Some other SQL database. Fallback only. See notes
	AttributeDBSystemOtherSQL = "other_sql"
	// Microsoft SQL Server
	AttributeDBSystemMSSQL = "mssql"
	// Microsoft SQL Server Compact
	AttributeDBSystemMssqlcompact = "mssqlcompact"
	// MySQL
	AttributeDBSystemMySQL = "mysql"
	// Oracle Database
	AttributeDBSystemOracle = "oracle"
	// IBM DB2
	AttributeDBSystemDB2 = "db2"
	// PostgreSQL
	AttributeDBSystemPostgreSQL = "postgresql"
	// Amazon Redshift
	AttributeDBSystemRedshift = "redshift"
	// Apache Hive
	AttributeDBSystemHive = "hive"
	// Cloudscape
	AttributeDBSystemCloudscape = "cloudscape"
	// HyperSQL DataBase
	AttributeDBSystemHSQLDB = "hsqldb"
	// Progress Database
	AttributeDBSystemProgress = "progress"
	// SAP MaxDB
	AttributeDBSystemMaxDB = "maxdb"
	// SAP HANA
	AttributeDBSystemHanaDB = "hanadb"
	// Ingres
	AttributeDBSystemIngres = "ingres"
	// FirstSQL
	AttributeDBSystemFirstSQL = "firstsql"
	// EnterpriseDB
	AttributeDBSystemEDB = "edb"
	// InterSystems Caché
	AttributeDBSystemCache = "cache"
	// Adabas (Adaptable Database System)
	AttributeDBSystemAdabas = "adabas"
	// Firebird
	AttributeDBSystemFirebird = "firebird"
	// Apache Derby
	AttributeDBSystemDerby = "derby"
	// FileMaker
	AttributeDBSystemFilemaker = "filemaker"
	// Informix
	AttributeDBSystemInformix = "informix"
	// InstantDB
	AttributeDBSystemInstantDB = "instantdb"
	// InterBase
	AttributeDBSystemInterbase = "interbase"
	// MariaDB
	AttributeDBSystemMariaDB = "mariadb"
	// Netezza
	AttributeDBSystemNetezza = "netezza"
	// Pervasive PSQL
	AttributeDBSystemPervasive = "pervasive"
	// PointBase
	AttributeDBSystemPointbase = "pointbase"
	// SQLite
	AttributeDBSystemSqlite = "sqlite"
	// Sybase
	AttributeDBSystemSybase = "sybase"
	// Teradata
	AttributeDBSystemTeradata = "teradata"
	// Vertica
	AttributeDBSystemVertica = "vertica"
	// H2
	AttributeDBSystemH2 = "h2"
	// ColdFusion IMQ
	AttributeDBSystemColdfusion = "coldfusion"
	// Apache Cassandra
	AttributeDBSystemCassandra = "cassandra"
	// Apache HBase
	AttributeDBSystemHBase = "hbase"
	// MongoDB
	AttributeDBSystemMongoDB = "mongodb"
	// Redis
	AttributeDBSystemRedis = "redis"
	// Couchbase
	AttributeDBSystemCouchbase = "couchbase"
	// CouchDB
	AttributeDBSystemCouchDB = "couchdb"
	// Microsoft Azure Cosmos DB
	AttributeDBSystemCosmosDB = "cosmosdb"
	// Amazon DynamoDB
	AttributeDBSystemDynamoDB = "dynamodb"
	// Neo4j
	AttributeDBSystemNeo4j = "neo4j"
	// Apache Geode
	AttributeDBSystemGeode = "geode"
	// Elasticsearch
	AttributeDBSystemElasticsearch = "elasticsearch"
	// Memcached
	AttributeDBSystemMemcached = "memcached"
	// CockroachDB
	AttributeDBSystemCockroachdb = "cockroachdb"
	// OpenSearch
	AttributeDBSystemOpensearch = "opensearch"
	// ClickHouse
	AttributeDBSystemClickhouse = "clickhouse"
	// Cloud Spanner
	AttributeDBSystemSpanner = "spanner"
	// Trino
	AttributeDBSystemTrino = "trino"
)

// Attributes for software deployments.
const (
	// Name of the deployment environment (aka deployment tier).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'staging', 'production'
	// Note: deployment.environment does not affect the uniqueness constraints defined
	// through
	// the service.namespace, service.name and service.instance.id resource
	// attributes.
	// This implies that resources carrying the following attribute combinations MUST
	// be
	// considered to be identifying the same service:<ul>
	// <li>service.name=frontend, deployment.environment=production</li>
	// <li>service.name=frontend, deployment.environment=staging.</li>
	// </ul>
	AttributeDeploymentEnvironment = "deployment.environment"
)

// "Describes deprecated db attributes."
const (
	// Deprecated, use server.address, server.port attributes instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: "Replaced by `server.address` and `server.port`."
	// Examples: 'Server=(localdb)\\v11.0;Integrated Security=true;'
	AttributeDBConnectionString = "db.connection_string"
	// Deprecated, use db.instance.id instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `db.instance.id`.
	// Examples: 'instance-0000000001'
	AttributeDBElasticsearchNodeName = "db.elasticsearch.node.name"
	// Removed, no replacement at this time.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Removed as not used.
	// Examples: 'org.postgresql.Driver',
	// 'com.microsoft.sqlserver.jdbc.SQLServerDriver'
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"
)

// Describes deprecated HTTP attributes.
const (
	// Deprecated, use network.protocol.name instead.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.protocol.name`.
	AttributeHTTPFlavor = "http.flavor"
	// Deprecated, use http.request.method instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `http.request.method`.
	// Examples: 'GET', 'POST', 'HEAD'
	AttributeHTTPMethod = "http.method"
	// Deprecated, use http.request.header.content-length instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `http.request.header.content-length`.
	// Examples: 3495
	AttributeHTTPRequestContentLength = "http.request_content_length"
	// Deprecated, use http.response.header.content-length instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `http.response.header.content-length`.
	// Examples: 3495
	AttributeHTTPResponseContentLength = "http.response_content_length"
	// Deprecated, use url.scheme instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `url.scheme` instead.
	// Examples: 'http', 'https'
	AttributeHTTPScheme = "http.scheme"
	// Deprecated, use http.response.status_code instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `http.response.status_code`.
	// Examples: 200
	AttributeHTTPStatusCode = "http.status_code"
	// Deprecated, use url.path and url.query instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Split to `url.path` and `url.query.
	// Examples: '/search?q=OpenTelemetry#SemConv'
	AttributeHTTPTarget = "http.target"
	// Deprecated, use url.full instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `url.full`.
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv'
	AttributeHTTPURL = "http.url"
	// Deprecated, use user_agent.original instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `user_agent.original`.
	// Examples: 'CERN-LineMode/2.15 libwww/2.17b3', 'Mozilla/5.0 (iPhone; CPU iPhone
	// OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko)
	// Version/14.1.2 Mobile/15E148 Safari/604.1'
	AttributeHTTPUserAgent = "http.user_agent"
)

const (
	// HTTP/1.0
	AttributeHTTPFlavorHTTP10 = "1.0"
	// HTTP/1.1
	AttributeHTTPFlavorHTTP11 = "1.1"
	// HTTP/2
	AttributeHTTPFlavorHTTP20 = "2.0"
	// HTTP/3
	AttributeHTTPFlavorHTTP30 = "3.0"
	// SPDY protocol
	AttributeHTTPFlavorSPDY = "SPDY"
	// QUIC protocol
	AttributeHTTPFlavorQUIC = "QUIC"
)

// Describes deprecated messaging attributes.
const (
	// &quot;Deprecated, use messaging.destination.partition.id instead.&quot;
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `messaging.destination.partition.id`.
	// Examples: 2
	AttributeMessagingKafkaDestinationPartition = "messaging.kafka.destination.partition"
)

// These attributes may be used for any network related operation.
const (
	// Deprecated, use server.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `server.address`.
	// Examples: 'example.com'
	AttributeNetHostName = "net.host.name"
	// Deprecated, use server.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `server.port`.
	// Examples: 8080
	AttributeNetHostPort = "net.host.port"
	// Deprecated, use server.address on client spans and client.address on server
	// spans.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `server.address` on client spans and `client.address`
	// on server spans.
	// Examples: 'example.com'
	AttributeNetPeerName = "net.peer.name"
	// Deprecated, use server.port on client spans and client.port on server spans.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `server.port` on client spans and `client.port` on
	// server spans.
	// Examples: 8080
	AttributeNetPeerPort = "net.peer.port"
	// Deprecated, use network.protocol.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.protocol.name`.
	// Examples: 'amqp', 'http', 'mqtt'
	AttributeNetProtocolName = "net.protocol.name"
	// Deprecated, use network.protocol.version.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.protocol.version`.
	// Examples: '3.1.1'
	AttributeNetProtocolVersion = "net.protocol.version"
	// Deprecated, use network.transport and network.type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Split to `network.transport` and `network.type`.
	AttributeNetSockFamily = "net.sock.family"
	// Deprecated, use network.local.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.local.address`.
	// Examples: '/var/my.sock'
	AttributeNetSockHostAddr = "net.sock.host.addr"
	// Deprecated, use network.local.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.local.port`.
	// Examples: 8080
	AttributeNetSockHostPort = "net.sock.host.port"
	// Deprecated, use network.peer.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.peer.address`.
	// Examples: '192.168.0.1'
	AttributeNetSockPeerAddr = "net.sock.peer.addr"
	// Deprecated, no replacement at this time.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Removed.
	// Examples: '/var/my.sock'
	AttributeNetSockPeerName = "net.sock.peer.name"
	// Deprecated, use network.peer.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.peer.port`.
	// Examples: 65531
	AttributeNetSockPeerPort = "net.sock.peer.port"
	// Deprecated, use network.transport.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `network.transport`.
	AttributeNetTransport = "net.transport"
)

const (
	// IPv4 address
	AttributeNetSockFamilyInet = "inet"
	// IPv6 address
	AttributeNetSockFamilyInet6 = "inet6"
	// Unix domain socket path
	AttributeNetSockFamilyUnix = "unix"
)

const (
	// ip_tcp
	AttributeNetTransportTCP = "ip_tcp"
	// ip_udp
	AttributeNetTransportUDP = "ip_udp"
	// Named or anonymous pipe
	AttributeNetTransportPipe = "pipe"
	// In-process communication
	AttributeNetTransportInProc = "inproc"
	// Something else (non IP-based)
	AttributeNetTransportOther = "other"
)

// Deprecated system attributes.
const (
	// Deprecated, use system.process.status instead.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: Replaced by `system.process.status`.
	// Examples: 'running'
	AttributeSystemProcessesStatus = "system.processes.status"
)

const (
	// running
	AttributeSystemProcessesStatusRunning = "running"
	// sleeping
	AttributeSystemProcessesStatusSleeping = "sleeping"
	// stopped
	AttributeSystemProcessesStatusStopped = "stopped"
	// defunct
	AttributeSystemProcessesStatusDefunct = "defunct"
)

// These attributes may be used to describe the receiver of a network
// exchange/packet. These should be used when there is no client/server
// relationship between the two sides, or when that relationship is unknown.
// This covers low-level network interactions (e.g. packet tracing) where you
// don't know if there was a connection or which side initiated it. This also
// covers unidirectional UDP flows and peer-to-peer communication where the
// "user-facing" surface of the protocol / API doesn't expose a clear notion of
// client and server.
const (
	// Destination address - domain name if available without reverse DNS lookup;
	// otherwise, IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'destination.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the source side, and when communicating through an
	// intermediary, destination.address SHOULD represent the destination address
	// behind any intermediaries, for example proxies, if it's available.
	AttributeDestinationAddress = "destination.address"
	// Destination port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeDestinationPort = "destination.port"
)

// Describes device attributes.
const (
	// A unique identifier representing the device
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2ab2916d-a51f-4ac8-80ee-45ac31a28092'
	// Note: The device identifier MUST only be defined using the values outlined
	// below. This value is not an advertising identifier and MUST NOT be used as
	// such. On iOS (Swift or Objective-C), this value MUST be equal to the vendor
	// identifier. On Android (Java or Kotlin), this value MUST be equal to the
	// Firebase Installation ID or a globally unique UUID which is persisted across
	// sessions in your application. More information can be found here on best
	// practices and exact implementation details. Caution should be taken when
	// storing personal data or anything which can identify a user. GDPR and data
	// protection laws may apply, ensure you do your own due diligence.
	AttributeDeviceID = "device.id"
	// The name of the device manufacturer
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Apple', 'Samsung'
	// Note: The Android OS provides this field via Build. iOS apps SHOULD hardcode
	// the value Apple.
	AttributeDeviceManufacturer = "device.manufacturer"
	// The model identifier for the device
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iPhone3,4', 'SM-G920F'
	// Note: It's recommended this value represents a machine-readable version of the
	// model identifier rather than the market or consumer-friendly name of the
	// device.
	AttributeDeviceModelIdentifier = "device.model.identifier"
	// The marketing name for the device model
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iPhone 6s Plus', 'Samsung Galaxy S6'
	// Note: It's recommended this value represents a human-readable version of the
	// device model rather than a machine-readable alternative.
	AttributeDeviceModelName = "device.model.name"
)

// These attributes may be used for any disk related operation.
const (
	// The disk IO operation direction.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'read'
	AttributeDiskIoDirection = "disk.io.direction"
)

const (
	// read
	AttributeDiskIoDirectionRead = "read"
	// write
	AttributeDiskIoDirectionWrite = "write"
)

// The shared attributes used to report a DNS query.
const (
	// The name being queried.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'www.example.com', 'opentelemetry.io'
	// Note: If the name field contains non-printable characters (below 32 or above
	// 126), those characters should be represented as escaped base 10 integers
	// (\DDD). Back slashes and quotes should be escaped. Tabs, carriage returns, and
	// line feeds should be converted to \t, \r, and \n respectively.
	AttributeDNSQuestionName = "dns.question.name"
)

// Attributes for operations with an authenticated and/or authorized enduser.
const (
	// Username or client_id extracted from the access token or Authorization header
	// in the inbound request from outside the system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'username'
	AttributeEnduserID = "enduser.id"
	// Actual/assumed role the client is making the request under extracted from token
	// or application security context.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'admin'
	AttributeEnduserRole = "enduser.role"
	// Scopes or granted authorities the client currently possesses extracted from
	// token or application security context. The value would come from the scope
	// associated with an OAuth 2.0 Access Token or an attribute value in a SAML 2.0
	// Assertion.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'read:message, write:files'
	AttributeEnduserScope = "enduser.scope"
)

// The shared attributes used to report an error.
const (
	// Describes a class of error the operation ended with.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'timeout', 'java.net.UnknownHostException',
	// 'server_certificate_invalid', '500'
	// Note: The error.type SHOULD be predictable and SHOULD have low cardinality.
	// Instrumentations SHOULD document the list of errors they report.The cardinality
	// of error.type within one instrumentation library SHOULD be low.
	// Telemetry consumers that aggregate data from multiple instrumentation libraries
	// and applications
	// should be prepared for error.type to have high cardinality at query time when
	// no
	// additional filters are applied.If the operation has completed successfully,
	// instrumentations SHOULD NOT set error.type.If a specific domain defines its own
	// set of error identifiers (such as HTTP or gRPC status codes),
	// it's RECOMMENDED to:<ul>
	// <li>Use a domain-specific attribute</li>
	// <li>Set error.type to capture all errors, regardless of whether they are
	// defined within the domain-specific set or not.</li>
	// </ul>
	AttributeErrorType = "error.type"
)

const (
	// A fallback error value to be used when the instrumentation doesn't define a custom value
	AttributeErrorTypeOther = "_OTHER"
)

// The shared attributes used to report a single exception associated with a
// span or log.
const (
	// SHOULD be set to true if the exception event is recorded at a point where it is
	// known that the exception is escaping the scope of the span.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: stable
	// Note: An exception is considered to have escaped (or left) the scope of a span,
	// if that span is ended while the exception is still logically &quot;in
	// flight&quot;.
	// This may be actually &quot;in flight&quot; in some languages (e.g. if the
	// exception
	// is passed to a Context manager's __exit__ method in Python) but will
	// usually be caught at the point of recording the exception in most languages.It
	// is usually not possible to determine at the point where an exception is thrown
	// whether it will escape the scope of a span.
	// However, it is trivial to know that an exception
	// will escape, if one checks for an active exception just before ending the span,
	// as done in the example for recording span exceptions.It follows that an
	// exception may still escape the scope of the span
	// even if the exception.escaped attribute was not set or set to false,
	// since the event might have been recorded at a time where it was not
	// clear whether the exception will escape.
	AttributeExceptionEscaped = "exception.escaped"
	// The exception message.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'Division by zero', "Can't convert 'int' object to str implicitly"
	AttributeExceptionMessage = "exception.message"
	// A stacktrace as a string in the natural representation for the language
	// runtime. The representation is to be determined and documented by each language
	// SIG.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'Exception in thread "main" java.lang.RuntimeException: Test
	// exception\\n at '
	//  'com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at '
	//  'com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at '
	//  'com.example.GenerateTrace.main(GenerateTrace.java:5)'
	AttributeExceptionStacktrace = "exception.stacktrace"
	// The type of the exception (its fully-qualified class name, if applicable). The
	// dynamic type of the exception should be preferred over the static type in
	// languages that support it.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'java.net.ConnectException', 'OSError'
	AttributeExceptionType = "exception.type"
)

// FaaS attributes
const (
	// A boolean that is true if the serverless function is executed for the first
	// time (aka cold-start).
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFaaSColdstart = "faas.coldstart"
	// A string containing the schedule period as Cron Expression.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0/5 * * * ? *'
	AttributeFaaSCron = "faas.cron"
	// The name of the source on which the triggering operation was performed. For
	// example, in Cloud Storage or S3 corresponds to the bucket name, and in Cosmos
	// DB to the database name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myBucketName', 'myDBName'
	AttributeFaaSDocumentCollection = "faas.document.collection"
	// The document name/table subjected to the operation. For example, in Cloud
	// Storage or S3 is the name of the file, and in Cosmos DB the table name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myFile.txt', 'myTableName'
	AttributeFaaSDocumentName = "faas.document.name"
	// Describes the type of the operation that was performed on the data.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFaaSDocumentOperation = "faas.document.operation"
	// A string containing the time when the data was accessed in the ISO 8601 format
	// expressed in UTC.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2020-01-23T13:47:06Z'
	AttributeFaaSDocumentTime = "faas.document.time"
	// The execution environment ID as a string, that will be potentially reused for
	// other invocations to the same function/function version.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de'
	// Note: <ul>
	// <li>AWS Lambda: Use the (full) log stream name.</li>
	// </ul>
	AttributeFaaSInstance = "faas.instance"
	// The invocation ID of the current function invocation.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'af9d5aa4-a685-4c5f-a22b-444f80b3cc28'
	AttributeFaaSInvocationID = "faas.invocation_id"
	// The name of the invoked function.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-function'
	// Note: SHOULD be equal to the faas.name resource attribute of the invoked
	// function.
	AttributeFaaSInvokedName = "faas.invoked_name"
	// The cloud provider of the invoked function.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: SHOULD be equal to the cloud.provider resource attribute of the invoked
	// function.
	AttributeFaaSInvokedProvider = "faas.invoked_provider"
	// The cloud region of the invoked function.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'eu-central-1'
	// Note: SHOULD be equal to the cloud.region resource attribute of the invoked
	// function.
	AttributeFaaSInvokedRegion = "faas.invoked_region"
	// The amount of memory available to the serverless function converted to Bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 134217728
	// Note: It's recommended to set this attribute since e.g. too little memory can
	// easily stop a Java AWS Lambda function from working correctly. On AWS Lambda,
	// the environment variable AWS_LAMBDA_FUNCTION_MEMORY_SIZE provides this
	// information (which must be multiplied by 1,048,576).
	AttributeFaaSMaxMemory = "faas.max_memory"
	// The name of the single function that this runtime instance executes.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-function', 'myazurefunctionapp/some-function-name'
	// Note: This is the name of the function as configured/deployed on the FaaS
	// platform and is usually different from the name of the callback
	// function (which may be stored in the
	// code.namespace/code.function
	// span attributes).For some cloud providers, the above definition is ambiguous.
	// The following
	// definition of function name MUST be used for this attribute
	// (and consequently the span name) for the listed cloud providers/products:<ul>
	// <li>Azure:  The full name <FUNCAPP>/<FUNC>, i.e., function app name
	// followed by a forward slash followed by the function name (this form
	// can also be seen in the resource JSON for the function).
	// This means that a span attribute MUST be used, as an Azure function
	// app can host multiple functions that would usually share
	// a TracerProvider (see also the cloud.resource_id attribute).</li>
	// </ul>
	AttributeFaaSName = "faas.name"
	// A string containing the function invocation time in the ISO 8601 format
	// expressed in UTC.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2020-01-23T13:47:06Z'
	AttributeFaaSTime = "faas.time"
	// Type of the trigger which caused this function invocation.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFaaSTrigger = "faas.trigger"
	// The immutable version of the function being executed.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '26', 'pinkfroid-00002'
	// Note: Depending on the cloud provider and platform, use:<ul>
	// <li>AWS Lambda: The function version
	// (an integer represented as a decimal string).</li>
	// <li>Google Cloud Run (Services): The revision
	// (i.e., the function name plus the revision suffix).</li>
	// <li>Google Cloud Functions: The value of the
	// K_REVISION environment variable.</li>
	// <li>Azure Functions: Not applicable. Do not set this attribute.</li>
	// </ul>
	AttributeFaaSVersion = "faas.version"
)

const (
	// When a new object is created
	AttributeFaaSDocumentOperationInsert = "insert"
	// When an object is modified
	AttributeFaaSDocumentOperationEdit = "edit"
	// When an object is deleted
	AttributeFaaSDocumentOperationDelete = "delete"
)

const (
	// Alibaba Cloud
	AttributeFaaSInvokedProviderAlibabaCloud = "alibaba_cloud"
	// Amazon Web Services
	AttributeFaaSInvokedProviderAWS = "aws"
	// Microsoft Azure
	AttributeFaaSInvokedProviderAzure = "azure"
	// Google Cloud Platform
	AttributeFaaSInvokedProviderGCP = "gcp"
	// Tencent Cloud
	AttributeFaaSInvokedProviderTencentCloud = "tencent_cloud"
)

const (
	// A response to some data source operation such as a database or filesystem read/write
	AttributeFaaSTriggerDatasource = "datasource"
	// To provide an answer to an inbound HTTP request
	AttributeFaaSTriggerHTTP = "http"
	// A function is set to be executed when messages are sent to a messaging system
	AttributeFaaSTriggerPubsub = "pubsub"
	// A function is scheduled to be executed regularly
	AttributeFaaSTriggerTimer = "timer"
	// If none of the others apply
	AttributeFaaSTriggerOther = "other"
)

// Attributes for Feature Flags.
const (
	// The unique identifier of the feature flag.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'logo-color'
	AttributeFeatureFlagKey = "feature_flag.key"
	// The name of the service provider that performs the flag evaluation.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Flag Manager'
	AttributeFeatureFlagProviderName = "feature_flag.provider_name"
	// SHOULD be a semantic identifier for a value. If one is unavailable, a
	// stringified version of the value can be used.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'red', 'true', 'on'
	// Note: A semantic identifier, commonly referred to as a variant, provides a
	// means
	// for referring to a value without including the value itself. This can
	// provide additional context for understanding the meaning behind a value.
	// For example, the variant red maybe be used for the value #c05543.A stringified
	// version of the value can be used in situations where a
	// semantic identifier is unavailable. String representation of the value
	// should be determined by the implementer.
	AttributeFeatureFlagVariant = "feature_flag.variant"
)

// Describes file attributes.
const (
	// Directory where the file is located. It should include the drive letter, when
	// appropriate.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/home/user', 'C:\\Program Files\\MyApp'
	AttributeFileDirectory = "file.directory"
	// File extension, excluding the leading dot.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'png', 'gz'
	// Note: When the file name has multiple extensions (example.tar.gz), only the
	// last one should be captured (&quot;gz&quot;, not &quot;tar.gz&quot;).
	AttributeFileExtension = "file.extension"
	// Name of the file including the extension, without the directory.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'example.png'
	AttributeFileName = "file.name"
	// Full path to the file, including the file name. It should include the drive
	// letter, when appropriate.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/home/alice/example.png', 'C:\\Program Files\\MyApp\\myapp.exe'
	AttributeFilePath = "file.path"
	// File size in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFileSize = "file.size"
)

// Attributes for Google Cloud Run.
const (
	// The name of the Cloud Run execution being run for the Job, as set by the
	// CLOUD_RUN_EXECUTION environment variable.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'job-name-xxxx', 'sample-job-mdw84'
	AttributeGCPCloudRunJobExecution = "gcp.cloud_run.job.execution"
	// The index for a task within an execution as provided by the
	// CLOUD_RUN_TASK_INDEX environment variable.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 1
	AttributeGCPCloudRunJobTaskIndex = "gcp.cloud_run.job.task_index"
)

// Attributes for Google Compute Engine (GCE).
const (
	// The hostname of a GCE instance. This is the full value of the default or custom
	// hostname.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-host1234.example.com', 'sample-vm.us-west1-b.c.my-
	// project.internal'
	AttributeGCPGceInstanceHostname = "gcp.gce.instance.hostname"
	// The instance name of a GCE instance. This is the value provided by host.name,
	// the visible name of the instance in the Cloud Console UI, and the prefix for
	// the default hostname of the instance as defined by the default internal DNS
	// name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'instance-1', 'my-vm-name'
	AttributeGCPGceInstanceName = "gcp.gce.instance.name"
)

// A host is defined as a computing instance. For example, physical servers,
// virtual machines, switches or disk array.
const (
	// The CPU architecture the host system is running on.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeHostArch = "host.arch"
	// The amount of level 2 memory cache available to the processor (in Bytes).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 12288000
	AttributeHostCPUCacheL2Size = "host.cpu.cache.l2.size"
	// Family or generation of the CPU.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '6', 'PA-RISC 1.1e'
	AttributeHostCPUFamily = "host.cpu.family"
	// Model identifier. It provides more granular information about the CPU,
	// distinguishing it from other CPUs within the same family.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '6', '9000/778/B180L'
	AttributeHostCPUModelID = "host.cpu.model.id"
	// Model designation of the processor.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz'
	AttributeHostCPUModelName = "host.cpu.model.name"
	// Stepping or core revisions.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1', 'r1p1'
	AttributeHostCPUStepping = "host.cpu.stepping"
	// Processor manufacturer identifier. A maximum 12-character string.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'GenuineIntel'
	// Note: CPUID command returns the vendor ID string in EBX, EDX and ECX registers.
	// Writing these to memory in this order results in a 12-character string.
	AttributeHostCPUVendorID = "host.cpu.vendor.id"
	// Unique host ID. For Cloud, this must be the instance_id assigned by the cloud
	// provider. For non-containerized systems, this should be the machine-id. See the
	// table below for the sources to use to determine the machine-id based on
	// operating system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'fdbf79e8af94cb7f9e8df36789187052'
	AttributeHostID = "host.id"
	// VM image ID or host OS image ID. For Cloud, this value is from the provider.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ami-07b06b442921831e5'
	AttributeHostImageID = "host.image.id"
	// Name of the VM image or OS install the host was instantiated from.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'infra-ami-eks-worker-node-7d4ec78312', 'CentOS-8-x86_64-1905'
	AttributeHostImageName = "host.image.name"
	// The version string of the VM image or host OS as defined in Version Attributes.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0.1'
	AttributeHostImageVersion = "host.image.version"
	// Available IP addresses of the host, excluding loopback interfaces.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '192.168.1.140', 'fe80::abc2:4a28:737a:609e'
	// Note: IPv4 Addresses MUST be specified in dotted-quad notation. IPv6 addresses
	// MUST be specified in the RFC 5952 format.
	AttributeHostIP = "host.ip"
	// Available MAC addresses of the host, excluding loopback interfaces.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'AC-DE-48-23-45-67', 'AC-DE-48-23-45-67-01-9F'
	// Note: MAC Addresses MUST be represented in IEEE RA hexadecimal form: as hyphen-
	// separated octets in uppercase hexadecimal form from most to least significant.
	AttributeHostMac = "host.mac"
	// Name of the host. On Unix systems, it may contain what the hostname command
	// returns, or the fully qualified hostname, or another name specified by the
	// user.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-test'
	AttributeHostName = "host.name"
	// Type of host. For Cloud, this must be the machine type.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'n1-standard-1'
	AttributeHostType = "host.type"
)

const (
	// AMD64
	AttributeHostArchAMD64 = "amd64"
	// ARM32
	AttributeHostArchARM32 = "arm32"
	// ARM64
	AttributeHostArchARM64 = "arm64"
	// Itanium
	AttributeHostArchIA64 = "ia64"
	// 32-bit PowerPC
	AttributeHostArchPPC32 = "ppc32"
	// 64-bit PowerPC
	AttributeHostArchPPC64 = "ppc64"
	// IBM z/Architecture
	AttributeHostArchS390x = "s390x"
	// 32-bit x86
	AttributeHostArchX86 = "x86"
)

// Semantic convention attributes in the HTTP namespace.
const (
	// State of the HTTP connection in the HTTP connection pool.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'active', 'idle'
	AttributeHTTPConnectionState = "http.connection.state"
	// The size of the request payload body in bytes. This is the number of bytes
	// transferred excluding headers and is often, but not always, present as the
	// Content-Length header. For requests using transport encoding, this should be
	// the compressed size.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3495
	AttributeHTTPRequestBodySize = "http.request.body.size"
	// HTTP request method.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'GET', 'POST', 'HEAD'
	// Note: HTTP request method value SHOULD be &quot;known&quot; to the
	// instrumentation.
	// By default, this convention defines &quot;known&quot; methods as the ones
	// listed in RFC9110
	// and the PATCH method defined in RFC5789.If the HTTP request method is not known
	// to instrumentation, it MUST set the http.request.method attribute to _OTHER.If
	// the HTTP instrumentation could end up converting valid HTTP request methods to
	// _OTHER, then it MUST provide a way to override
	// the list of known HTTP methods. If this override is done via environment
	// variable, then the environment variable MUST be named
	// OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS and support a comma-separated list of
	// case-sensitive known HTTP methods
	// (this list MUST be a full override of the default known method, it is not a
	// list of known methods in addition to the defaults).HTTP method names are case-
	// sensitive and http.request.method attribute value MUST match a known HTTP
	// method name exactly.
	// Instrumentations for specific web frameworks that consider HTTP methods to be
	// case insensitive, SHOULD populate a canonical equivalent.
	// Tracing instrumentations that do so, MUST also set http.request.method_original
	// to the original value.
	AttributeHTTPRequestMethod = "http.request.method"
	// Original HTTP method sent by the client in the request line.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'GeT', 'ACL', 'foo'
	AttributeHTTPRequestMethodOriginal = "http.request.method_original"
	// The ordinal number of request resending attempt (for any reason, including
	// redirects).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 3
	// Note: The resend count SHOULD be updated each time an HTTP request gets resent
	// by the client, regardless of what was the cause of the resending (e.g.
	// redirection, authorization failure, 503 Server Unavailable, network issues, or
	// any other).
	AttributeHTTPRequestResendCount = "http.request.resend_count"
	// The total size of the request in bytes. This should be the total number of
	// bytes sent over the wire, including the request line (HTTP/1.1), framing
	// (HTTP/2 and HTTP/3), headers, and request body if any.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1437
	AttributeHTTPRequestSize = "http.request.size"
	// The size of the response payload body in bytes. This is the number of bytes
	// transferred excluding headers and is often, but not always, present as the
	// Content-Length header. For requests using transport encoding, this should be
	// the compressed size.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3495
	AttributeHTTPResponseBodySize = "http.response.body.size"
	// The total size of the response in bytes. This should be the total number of
	// bytes sent over the wire, including the status line (HTTP/1.1), framing (HTTP/2
	// and HTTP/3), headers, and response body and trailers if any.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1437
	AttributeHTTPResponseSize = "http.response.size"
	// HTTP response status code.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 200
	AttributeHTTPResponseStatusCode = "http.response.status_code"
	// The matched route, that is, the path template in the format used by the
	// respective server framework.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '/users/:userID?', '{controller}/{action}/{id?}'
	// Note: MUST NOT be populated when this is not supported by the HTTP server
	// framework as the route attribute should have low-cardinality and the URI path
	// can NOT substitute it.
	// SHOULD include the application root if there is one.
	AttributeHTTPRoute = "http.route"
)

const (
	// active state
	AttributeHTTPConnectionStateActive = "active"
	// idle state
	AttributeHTTPConnectionStateIdle = "idle"
)

const (
	// CONNECT method
	AttributeHTTPRequestMethodConnect = "CONNECT"
	// DELETE method
	AttributeHTTPRequestMethodDelete = "DELETE"
	// GET method
	AttributeHTTPRequestMethodGet = "GET"
	// HEAD method
	AttributeHTTPRequestMethodHead = "HEAD"
	// OPTIONS method
	AttributeHTTPRequestMethodOptions = "OPTIONS"
	// PATCH method
	AttributeHTTPRequestMethodPatch = "PATCH"
	// POST method
	AttributeHTTPRequestMethodPost = "POST"
	// PUT method
	AttributeHTTPRequestMethodPut = "PUT"
	// TRACE method
	AttributeHTTPRequestMethodTrace = "TRACE"
	// Any HTTP method that the instrumentation has no prior knowledge of
	AttributeHTTPRequestMethodOther = "_OTHER"
)

// Kubernetes resource attributes.
const (
	// The name of the cluster.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-cluster'
	AttributeK8SClusterName = "k8s.cluster.name"
	// A pseudo-ID for the cluster, set to the UID of the kube-system namespace.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '218fc5a9-a5f1-4b54-aa05-46717d0ab26d'
	// Note: K8S doesn't have support for obtaining a cluster ID. If this is ever
	// added, we will recommend collecting the k8s.cluster.uid through the
	// official APIs. In the meantime, we are able to use the uid of the
	// kube-system namespace as a proxy for cluster ID. Read on for the
	// rationale.Every object created in a K8S cluster is assigned a distinct UID. The
	// kube-system namespace is used by Kubernetes itself and will exist
	// for the lifetime of the cluster. Using the uid of the kube-system
	// namespace is a reasonable proxy for the K8S ClusterID as it will only
	// change if the cluster is rebuilt. Furthermore, Kubernetes UIDs are
	// UUIDs as standardized by
	// ISO/IEC 9834-8 and ITU-T X.667.
	// Which states:<blockquote>
	// If generated according to one of the mechanisms defined in Rec.</blockquote>
	// ITU-T X.667 | ISO/IEC 9834-8, a UUID is either guaranteed to be
	//   different from all other UUIDs generated before 3603 A.D., or is
	//   extremely likely to be different (depending on the mechanism
	// chosen).Therefore, UIDs between clusters should be extremely unlikely to
	// conflict.
	AttributeK8SClusterUID = "k8s.cluster.uid"
	// The name of the Container from Pod specification, must be unique within a Pod.
	// Container runtime usually uses different globally unique name (container.name).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'redis'
	AttributeK8SContainerName = "k8s.container.name"
	// Number of times the container was restarted. This attribute can be used to
	// identify a particular container (running or stopped) within a container spec.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 2
	AttributeK8SContainerRestartCount = "k8s.container.restart_count"
	// The name of the CronJob.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SCronJobName = "k8s.cronjob.name"
	// The UID of the CronJob.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SCronJobUID = "k8s.cronjob.uid"
	// The name of the DaemonSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SDaemonSetName = "k8s.daemonset.name"
	// The UID of the DaemonSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SDaemonSetUID = "k8s.daemonset.uid"
	// The name of the Deployment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SDeploymentName = "k8s.deployment.name"
	// The UID of the Deployment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SDeploymentUID = "k8s.deployment.uid"
	// The name of the Job.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SJobName = "k8s.job.name"
	// The UID of the Job.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SJobUID = "k8s.job.uid"
	// The name of the namespace that the pod is running in.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'default'
	AttributeK8SNamespaceName = "k8s.namespace.name"
	// The name of the Node.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'node-1'
	AttributeK8SNodeName = "k8s.node.name"
	// The UID of the Node.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1eb3a0c6-0477-4080-a9cb-0cb7db65c6a2'
	AttributeK8SNodeUID = "k8s.node.uid"
	// The name of the Pod.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-pod-autoconf'
	AttributeK8SPodName = "k8s.pod.name"
	// The UID of the Pod.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SPodUID = "k8s.pod.uid"
	// The name of the ReplicaSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SReplicaSetName = "k8s.replicaset.name"
	// The UID of the ReplicaSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SReplicaSetUID = "k8s.replicaset.uid"
	// The name of the StatefulSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SStatefulSetName = "k8s.statefulset.name"
	// The UID of the StatefulSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SStatefulSetUID = "k8s.statefulset.uid"
)

// Attributes describing telemetry around messaging systems and messaging
// activities.
const (
	// The number of messages sent, received, or processed in the scope of the
	// batching operation.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 1, 2
	// Note: Instrumentations SHOULD NOT set messaging.batch.message_count on spans
	// that operate with a single message. When a messaging client library supports
	// both batch and single-message API for the same operation, instrumentations
	// SHOULD use messaging.batch.message_count for batching APIs and SHOULD NOT use
	// it for single-message APIs.
	AttributeMessagingBatchMessageCount = "messaging.batch.message_count"
	// A unique identifier for the client that consumes or produces a message.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'client-5', 'myhost@8742@s8083jm'
	AttributeMessagingClientID = "messaging.client_id"
	// A boolean that is true if the message destination is anonymous (could be
	// unnamed or have auto-generated name).
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingDestinationAnonymous = "messaging.destination.anonymous"
	// The message destination name
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MyQueue', 'MyTopic'
	// Note: Destination name SHOULD uniquely identify a specific queue, topic or
	// other entity within the broker. If
	// the broker doesn't have such notion, the destination name SHOULD uniquely
	// identify the broker.
	AttributeMessagingDestinationName = "messaging.destination.name"
	// The identifier of the partition messages are sent to or received from, unique
	// within the messaging.destination.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1'
	AttributeMessagingDestinationPartitionID = "messaging.destination.partition.id"
	// Low cardinality representation of the messaging destination name
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/customers/{customerID}'
	// Note: Destination names could be constructed from templates. An example would
	// be a destination name involving a user name or product id. Although the
	// destination name in this case is of high cardinality, the underlying template
	// is of low cardinality and can be effectively used for grouping and aggregation.
	AttributeMessagingDestinationTemplate = "messaging.destination.template"
	// A boolean that is true if the message destination is temporary and might not
	// exist anymore after messages are processed.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingDestinationTemporary = "messaging.destination.temporary"
	// A boolean that is true if the publish message destination is anonymous (could
	// be unnamed or have auto-generated name).
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingDestinationPublishAnonymous = "messaging.destination_publish.anonymous"
	// The name of the original destination the message was published to
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MyQueue', 'MyTopic'
	// Note: The name SHOULD uniquely identify a specific queue, topic, or other
	// entity within the broker. If
	// the broker doesn't have such notion, the original destination name SHOULD
	// uniquely identify the broker.
	AttributeMessagingDestinationPublishName = "messaging.destination_publish.name"
	// The name of the consumer group the event consumer is associated with.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'indexer'
	AttributeMessagingEventhubsConsumerGroup = "messaging.eventhubs.consumer.group"
	// The UTC epoch seconds at which the message has been accepted and stored in the
	// entity.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1701393730
	AttributeMessagingEventhubsMessageEnqueuedTime = "messaging.eventhubs.message.enqueued_time"
	// The ordering key for a given message. If the attribute is not present, the
	// message does not have an ordering key.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ordering_key'
	AttributeMessagingGCPPubsubMessageOrderingKey = "messaging.gcp_pubsub.message.ordering_key"
	// Name of the Kafka Consumer Group that is handling the message. Only applies to
	// consumers, not producers.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-group'
	AttributeMessagingKafkaConsumerGroup = "messaging.kafka.consumer.group"
	// Message keys in Kafka are used for grouping alike messages to ensure they're
	// processed on the same partition. They differ from messaging.message.id in that
	// they're not unique. If the key is null, the attribute MUST NOT be set.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myKey'
	// Note: If the key type is not string, it's string representation has to be
	// supplied for the attribute. If the key has no unambiguous, canonical string
	// form, don't include its value.
	AttributeMessagingKafkaMessageKey = "messaging.kafka.message.key"
	// The offset of a record in the corresponding Kafka partition.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 42
	AttributeMessagingKafkaMessageOffset = "messaging.kafka.message.offset"
	// A boolean that is true if the message is a tombstone.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingKafkaMessageTombstone = "messaging.kafka.message.tombstone"
	// The size of the message body in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1439
	// Note: This can refer to both the compressed or uncompressed body size. If both
	// sizes are known, the uncompressed
	// body size should be used.
	AttributeMessagingMessageBodySize = "messaging.message.body.size"
	// The conversation ID identifying the conversation to which the message belongs,
	// represented as a string. Sometimes called &quot;Correlation ID&quot;.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MyConversationID'
	AttributeMessagingMessageConversationID = "messaging.message.conversation_id"
	// The size of the message body and metadata in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 2738
	// Note: This can refer to both the compressed or uncompressed size. If both sizes
	// are known, the uncompressed
	// size should be used.
	AttributeMessagingMessageEnvelopeSize = "messaging.message.envelope.size"
	// A value used by the messaging system as an identifier for the message,
	// represented as a string.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '452a7c7c7c7048c2f887f61572b18fc2'
	AttributeMessagingMessageID = "messaging.message.id"
	// A string identifying the kind of messaging operation.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: If a custom value is used, it MUST be of low cardinality.
	AttributeMessagingOperation = "messaging.operation"
	// RabbitMQ message routing key.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myKey'
	AttributeMessagingRabbitmqDestinationRoutingKey = "messaging.rabbitmq.destination.routing_key"
	// RabbitMQ message delivery tag
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 123
	AttributeMessagingRabbitmqMessageDeliveryTag = "messaging.rabbitmq.message.delivery_tag"
	// Name of the RocketMQ producer/consumer group that is handling the message. The
	// client type is identified by the SpanKind.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myConsumerGroup'
	AttributeMessagingRocketmqClientGroup = "messaging.rocketmq.client_group"
	// Model of message consumption. This only applies to consumer spans.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingRocketmqConsumptionModel = "messaging.rocketmq.consumption_model"
	// The delay time level for delay message, which determines the message delay
	// time.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3
	AttributeMessagingRocketmqMessageDelayTimeLevel = "messaging.rocketmq.message.delay_time_level"
	// The timestamp in milliseconds that the delay message is expected to be
	// delivered to consumer.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1665987217045
	AttributeMessagingRocketmqMessageDeliveryTimestamp = "messaging.rocketmq.message.delivery_timestamp"
	// It is essential for FIFO message. Messages that belong to the same message
	// group are always processed one by one within the same consumer group.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myMessageGroup'
	AttributeMessagingRocketmqMessageGroup = "messaging.rocketmq.message.group"
	// Key(s) of message, another way to mark message besides message id.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'keyA', 'keyB'
	AttributeMessagingRocketmqMessageKeys = "messaging.rocketmq.message.keys"
	// The secondary classifier of message besides topic.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'tagA'
	AttributeMessagingRocketmqMessageTag = "messaging.rocketmq.message.tag"
	// Type of message.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingRocketmqMessageType = "messaging.rocketmq.message.type"
	// Namespace of RocketMQ resources, resources in different namespaces are
	// individual.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myNamespace'
	AttributeMessagingRocketmqNamespace = "messaging.rocketmq.namespace"
	// The name of the subscription in the topic messages are received from.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'mySubscription'
	AttributeMessagingServicebusDestinationSubscriptionName = "messaging.servicebus.destination.subscription_name"
	// Describes the settlement type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingServicebusDispositionStatus = "messaging.servicebus.disposition_status"
	// Number of deliveries that have been attempted for this message.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 2
	AttributeMessagingServicebusMessageDeliveryCount = "messaging.servicebus.message.delivery_count"
	// The UTC epoch seconds at which the message has been accepted and stored in the
	// entity.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1701393730
	AttributeMessagingServicebusMessageEnqueuedTime = "messaging.servicebus.message.enqueued_time"
	// An identifier for the messaging system being used. See below for a list of
	// well-known identifiers.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessagingSystem = "messaging.system"
)

const (
	// One or more messages are provided for publishing to an intermediary. If a single message is published, the context of the "Publish" span can be used as the creation context and no "Create" span needs to be created
	AttributeMessagingOperationPublish = "publish"
	// A message is created. "Create" spans always refer to a single message and are used to provide a unique creation context for messages in batch publishing scenarios
	AttributeMessagingOperationCreate = "create"
	// One or more messages are requested by a consumer. This operation refers to pull-based scenarios, where consumers explicitly call methods of messaging SDKs to receive messages
	AttributeMessagingOperationReceive = "receive"
	// One or more messages are delivered to or processed by a consumer
	AttributeMessagingOperationDeliver = "process"
	// One or more messages are settled
	AttributeMessagingOperationSettle = "settle"
)

const (
	// Clustering consumption model
	AttributeMessagingRocketmqConsumptionModelClustering = "clustering"
	// Broadcasting consumption model
	AttributeMessagingRocketmqConsumptionModelBroadcasting = "broadcasting"
)

const (
	// Normal message
	AttributeMessagingRocketmqMessageTypeNormal = "normal"
	// FIFO message
	AttributeMessagingRocketmqMessageTypeFifo = "fifo"
	// Delay message
	AttributeMessagingRocketmqMessageTypeDelay = "delay"
	// Transaction message
	AttributeMessagingRocketmqMessageTypeTransaction = "transaction"
)

const (
	// Message is completed
	AttributeMessagingServicebusDispositionStatusComplete = "complete"
	// Message is abandoned
	AttributeMessagingServicebusDispositionStatusAbandon = "abandon"
	// Message is sent to dead letter queue
	AttributeMessagingServicebusDispositionStatusDeadLetter = "dead_letter"
	// Message is deferred
	AttributeMessagingServicebusDispositionStatusDefer = "defer"
)

const (
	// Apache ActiveMQ
	AttributeMessagingSystemActivemq = "activemq"
	// Amazon Simple Queue Service (SQS)
	AttributeMessagingSystemAWSSqs = "aws_sqs"
	// Azure Event Grid
	AttributeMessagingSystemEventgrid = "eventgrid"
	// Azure Event Hubs
	AttributeMessagingSystemEventhubs = "eventhubs"
	// Azure Service Bus
	AttributeMessagingSystemServicebus = "servicebus"
	// Google Cloud Pub/Sub
	AttributeMessagingSystemGCPPubsub = "gcp_pubsub"
	// Java Message Service
	AttributeMessagingSystemJms = "jms"
	// Apache Kafka
	AttributeMessagingSystemKafka = "kafka"
	// RabbitMQ
	AttributeMessagingSystemRabbitmq = "rabbitmq"
	// Apache RocketMQ
	AttributeMessagingSystemRocketmq = "rocketmq"
)

// These attributes may be used for any network related operation.
const (
	// The ISO 3166-1 alpha-2 2-character country code associated with the mobile
	// carrier network.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'DE'
	AttributeNetworkCarrierIcc = "network.carrier.icc"
	// The mobile carrier country code.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '310'
	AttributeNetworkCarrierMcc = "network.carrier.mcc"
	// The mobile carrier network code.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '001'
	AttributeNetworkCarrierMnc = "network.carrier.mnc"
	// The name of the mobile carrier.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'sprint'
	AttributeNetworkCarrierName = "network.carrier.name"
	// This describes more details regarding the connection.type. It may be the type
	// of cell technology connection, but it could be used for describing details
	// about a wifi connection.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'LTE'
	AttributeNetworkConnectionSubtype = "network.connection.subtype"
	// The internet connection type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'wifi'
	AttributeNetworkConnectionType = "network.connection.type"
	// The network IO operation direction.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'transmit'
	AttributeNetworkIoDirection = "network.io.direction"
	// Local address of the network connection - IP address or Unix domain socket
	// name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '10.1.2.80', '/tmp/my.sock'
	AttributeNetworkLocalAddress = "network.local.address"
	// Local port number of the network connection.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 65123
	AttributeNetworkLocalPort = "network.local.port"
	// Peer address of the network connection - IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '10.1.2.80', '/tmp/my.sock'
	AttributeNetworkPeerAddress = "network.peer.address"
	// Peer port number of the network connection.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 65123
	AttributeNetworkPeerPort = "network.peer.port"
	// OSI application layer or non-OSI equivalent.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'amqp', 'http', 'mqtt'
	// Note: The value SHOULD be normalized to lowercase.
	AttributeNetworkProtocolName = "network.protocol.name"
	// The actual version of the protocol used for network communication.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '1.1', '2'
	// Note: If protocol version is subject to negotiation (for example using ALPN),
	// this attribute SHOULD be set to the negotiated version. If the actual protocol
	// version is not known, this attribute SHOULD NOT be set.
	AttributeNetworkProtocolVersion = "network.protocol.version"
	// OSI transport layer or inter-process communication method.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'tcp', 'udp'
	// Note: The value SHOULD be normalized to lowercase.Consider always setting the
	// transport when setting a port number, since
	// a port number is ambiguous without knowing the transport. For example
	// different processes could be listening on TCP port 12345 and UDP port 12345.
	AttributeNetworkTransport = "network.transport"
	// OSI network layer or non-OSI equivalent.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'ipv4', 'ipv6'
	// Note: The value SHOULD be normalized to lowercase.
	AttributeNetworkType = "network.type"
)

const (
	// GPRS
	AttributeNetworkConnectionSubtypeGprs = "gprs"
	// EDGE
	AttributeNetworkConnectionSubtypeEdge = "edge"
	// UMTS
	AttributeNetworkConnectionSubtypeUmts = "umts"
	// CDMA
	AttributeNetworkConnectionSubtypeCdma = "cdma"
	// EVDO Rel. 0
	AttributeNetworkConnectionSubtypeEvdo0 = "evdo_0"
	// EVDO Rev. A
	AttributeNetworkConnectionSubtypeEvdoA = "evdo_a"
	// CDMA2000 1XRTT
	AttributeNetworkConnectionSubtypeCdma20001xrtt = "cdma2000_1xrtt"
	// HSDPA
	AttributeNetworkConnectionSubtypeHsdpa = "hsdpa"
	// HSUPA
	AttributeNetworkConnectionSubtypeHsupa = "hsupa"
	// HSPA
	AttributeNetworkConnectionSubtypeHspa = "hspa"
	// IDEN
	AttributeNetworkConnectionSubtypeIden = "iden"
	// EVDO Rev. B
	AttributeNetworkConnectionSubtypeEvdoB = "evdo_b"
	// LTE
	AttributeNetworkConnectionSubtypeLte = "lte"
	// EHRPD
	AttributeNetworkConnectionSubtypeEhrpd = "ehrpd"
	// HSPAP
	AttributeNetworkConnectionSubtypeHspap = "hspap"
	// GSM
	AttributeNetworkConnectionSubtypeGsm = "gsm"
	// TD-SCDMA
	AttributeNetworkConnectionSubtypeTdScdma = "td_scdma"
	// IWLAN
	AttributeNetworkConnectionSubtypeIwlan = "iwlan"
	// 5G NR (New Radio)
	AttributeNetworkConnectionSubtypeNr = "nr"
	// 5G NRNSA (New Radio Non-Standalone)
	AttributeNetworkConnectionSubtypeNrnsa = "nrnsa"
	// LTE CA
	AttributeNetworkConnectionSubtypeLteCa = "lte_ca"
)

const (
	// wifi
	AttributeNetworkConnectionTypeWifi = "wifi"
	// wired
	AttributeNetworkConnectionTypeWired = "wired"
	// cell
	AttributeNetworkConnectionTypeCell = "cell"
	// unavailable
	AttributeNetworkConnectionTypeUnavailable = "unavailable"
	// unknown
	AttributeNetworkConnectionTypeUnknown = "unknown"
)

const (
	// transmit
	AttributeNetworkIoDirectionTransmit = "transmit"
	// receive
	AttributeNetworkIoDirectionReceive = "receive"
)

const (
	// TCP
	AttributeNetworkTransportTCP = "tcp"
	// UDP
	AttributeNetworkTransportUDP = "udp"
	// Named or anonymous pipe
	AttributeNetworkTransportPipe = "pipe"
	// Unix domain socket
	AttributeNetworkTransportUnix = "unix"
)

const (
	// IPv4
	AttributeNetworkTypeIpv4 = "ipv4"
	// IPv6
	AttributeNetworkTypeIpv6 = "ipv6"
)

// An OCI image manifest.
const (
	// The digest of the OCI image manifest. For container images specifically is the
	// digest by which the container image is known.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples:
	// 'sha256:e4ca62c0d62f3e886e684806dfe9d4e0cda60d54986898173c1083856cfda0f4'
	// Note: Follows OCI Image Manifest Specification, and specifically the Digest
	// property.
	// An example can be found in Example Image Manifest.
	AttributeOciManifestDigest = "oci.manifest.digest"
)

// The operating system (OS) on which the process represented by this resource
// is running.
const (
	// Unique identifier for a particular build or compilation of the operating
	// system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'TQ3C.230805.001.B2', '20E247', '22621'
	AttributeOSBuildID = "os.build_id"
	// Human readable (not intended to be parsed) OS version information, like e.g.
	// reported by ver or lsb_release -a commands.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Microsoft Windows [Version 10.0.18363.778]', 'Ubuntu 18.04.1 LTS'
	AttributeOSDescription = "os.description"
	// Human readable operating system name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iOS', 'Android', 'Ubuntu'
	AttributeOSName = "os.name"
	// The operating system type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeOSType = "os.type"
	// The version string of the operating system as defined in Version Attributes.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '14.2.1', '18.04.1'
	AttributeOSVersion = "os.version"
)

const (
	// Microsoft Windows
	AttributeOSTypeWindows = "windows"
	// Linux
	AttributeOSTypeLinux = "linux"
	// Apple Darwin
	AttributeOSTypeDarwin = "darwin"
	// FreeBSD
	AttributeOSTypeFreeBSD = "freebsd"
	// NetBSD
	AttributeOSTypeNetBSD = "netbsd"
	// OpenBSD
	AttributeOSTypeOpenBSD = "openbsd"
	// DragonFly BSD
	AttributeOSTypeDragonflyBSD = "dragonflybsd"
	// HP-UX (Hewlett Packard Unix)
	AttributeOSTypeHPUX = "hpux"
	// AIX (Advanced Interactive eXecutive)
	AttributeOSTypeAIX = "aix"
	// SunOS, Oracle Solaris
	AttributeOSTypeSolaris = "solaris"
	// IBM z/OS
	AttributeOSTypeZOS = "z_os"
)

// An operating system process.
const (
	// The command used to launch the process (i.e. the command name). On Linux based
	// systems, can be set to the zeroth string in proc/[pid]/cmdline. On Windows, can
	// be set to the first parameter extracted from GetCommandLineW.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'cmd/otelcol'
	AttributeProcessCommand = "process.command"
	// All the command arguments (including the command/executable itself) as received
	// by the process. On Linux-based systems (and some other Unixoid systems
	// supporting procfs), can be set according to the list of null-delimited strings
	// extracted from proc/[pid]/cmdline. For libc-based executables, this would be
	// the full argv vector passed to main.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'cmd/otecol', '--config=config.yaml'
	AttributeProcessCommandArgs = "process.command_args"
	// The full command used to launch the process as a single string representing the
	// full command. On Windows, can be set to the result of GetCommandLineW. Do not
	// set this if you have to assemble it just for monitoring; use
	// process.command_args instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'C:\\cmd\\otecol --config="my directory\\config.yaml"'
	AttributeProcessCommandLine = "process.command_line"
	// The name of the process executable. On Linux based systems, can be set to the
	// Name in proc/[pid]/status. On Windows, can be set to the base name of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcol'
	AttributeProcessExecutableName = "process.executable.name"
	// The full path to the process executable. On Linux based systems, can be set to
	// the target of proc/[pid]/exe. On Windows, can be set to the result of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/usr/bin/cmd/otelcol'
	AttributeProcessExecutablePath = "process.executable.path"
	// The username of the user that owns the process.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'root'
	AttributeProcessOwner = "process.owner"
	// Parent Process identifier (PPID).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 111
	AttributeProcessParentPID = "process.parent_pid"
	// Process identifier (PID).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1234
	AttributeProcessPID = "process.pid"
	// An additional description about the runtime of the process, for example a
	// specific vendor customization of the runtime environment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Eclipse OpenJ9 Eclipse OpenJ9 VM openj9-0.21.0'
	AttributeProcessRuntimeDescription = "process.runtime.description"
	// The name of the runtime of this process. For compiled native binaries, this
	// SHOULD be the name of the compiler.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'OpenJDK Runtime Environment'
	AttributeProcessRuntimeName = "process.runtime.name"
	// The version of the runtime of this process, as returned by the runtime without
	// modification.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '14.0.2'
	AttributeProcessRuntimeVersion = "process.runtime.version"
)

// Attributes for remote procedure calls.
const (
	// The error codes of the Connect request. Error codes are always string values.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeRPCConnectRPCErrorCode = "rpc.connect_rpc.error_code"
	// The numeric status code of the gRPC request.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeRPCGRPCStatusCode = "rpc.grpc.status_code"
	// error.code property of response if it is an error response.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: -32700, 100
	AttributeRPCJsonrpcErrorCode = "rpc.jsonrpc.error_code"
	// error.message property of response if it is an error response.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Parse error', 'User already exists'
	AttributeRPCJsonrpcErrorMessage = "rpc.jsonrpc.error_message"
	// id property of request or response. Since protocol allows id to be int, string,
	// null or missing (for notifications), value is expected to be cast to string for
	// simplicity. Use empty string in case of null value. Omit entirely if this is a
	// notification.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '10', 'request-7', ''
	AttributeRPCJsonrpcRequestID = "rpc.jsonrpc.request_id"
	// Protocol version as in jsonrpc property of request/response. Since JSON-RPC 1.0
	// doesn't specify this, the value can be omitted.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2.0', '1.0'
	AttributeRPCJsonrpcVersion = "rpc.jsonrpc.version"
	// The name of the (logical) method being called, must be equal to the $method
	// part in the span name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'exampleMethod'
	// Note: This is the logical name of the method from the RPC interface
	// perspective, which can be different from the name of any implementing
	// method/function. The code.function attribute may be used to store the latter
	// (e.g., method actually executing the call on the server side, RPC client stub
	// method on the client side).
	AttributeRPCMethod = "rpc.method"
	// The full (logical) name of the service being called, including its package
	// name, if applicable.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'myservice.EchoService'
	// Note: This is the logical name of the service from the RPC interface
	// perspective, which can be different from the name of any implementing class.
	// The code.namespace attribute may be used to store the latter (despite the
	// attribute name, it may include a class name; e.g., class with method actually
	// executing the call on the server side, RPC client stub class on the client
	// side).
	AttributeRPCService = "rpc.service"
	// A string identifying the remoting system. See below for a list of well-known
	// identifiers.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeRPCSystem = "rpc.system"
)

const (
	// cancelled
	AttributeRPCConnectRPCErrorCodeCancelled = "cancelled"
	// unknown
	AttributeRPCConnectRPCErrorCodeUnknown = "unknown"
	// invalid_argument
	AttributeRPCConnectRPCErrorCodeInvalidArgument = "invalid_argument"
	// deadline_exceeded
	AttributeRPCConnectRPCErrorCodeDeadlineExceeded = "deadline_exceeded"
	// not_found
	AttributeRPCConnectRPCErrorCodeNotFound = "not_found"
	// already_exists
	AttributeRPCConnectRPCErrorCodeAlreadyExists = "already_exists"
	// permission_denied
	AttributeRPCConnectRPCErrorCodePermissionDenied = "permission_denied"
	// resource_exhausted
	AttributeRPCConnectRPCErrorCodeResourceExhausted = "resource_exhausted"
	// failed_precondition
	AttributeRPCConnectRPCErrorCodeFailedPrecondition = "failed_precondition"
	// aborted
	AttributeRPCConnectRPCErrorCodeAborted = "aborted"
	// out_of_range
	AttributeRPCConnectRPCErrorCodeOutOfRange = "out_of_range"
	// unimplemented
	AttributeRPCConnectRPCErrorCodeUnimplemented = "unimplemented"
	// internal
	AttributeRPCConnectRPCErrorCodeInternal = "internal"
	// unavailable
	AttributeRPCConnectRPCErrorCodeUnavailable = "unavailable"
	// data_loss
	AttributeRPCConnectRPCErrorCodeDataLoss = "data_loss"
	// unauthenticated
	AttributeRPCConnectRPCErrorCodeUnauthenticated = "unauthenticated"
)

const (
	// OK
	AttributeRPCGRPCStatusCodeOk = "0"
	// CANCELLED
	AttributeRPCGRPCStatusCodeCancelled = "1"
	// UNKNOWN
	AttributeRPCGRPCStatusCodeUnknown = "2"
	// INVALID_ARGUMENT
	AttributeRPCGRPCStatusCodeInvalidArgument = "3"
	// DEADLINE_EXCEEDED
	AttributeRPCGRPCStatusCodeDeadlineExceeded = "4"
	// NOT_FOUND
	AttributeRPCGRPCStatusCodeNotFound = "5"
	// ALREADY_EXISTS
	AttributeRPCGRPCStatusCodeAlreadyExists = "6"
	// PERMISSION_DENIED
	AttributeRPCGRPCStatusCodePermissionDenied = "7"
	// RESOURCE_EXHAUSTED
	AttributeRPCGRPCStatusCodeResourceExhausted = "8"
	// FAILED_PRECONDITION
	AttributeRPCGRPCStatusCodeFailedPrecondition = "9"
	// ABORTED
	AttributeRPCGRPCStatusCodeAborted = "10"
	// OUT_OF_RANGE
	AttributeRPCGRPCStatusCodeOutOfRange = "11"
	// UNIMPLEMENTED
	AttributeRPCGRPCStatusCodeUnimplemented = "12"
	// INTERNAL
	AttributeRPCGRPCStatusCodeInternal = "13"
	// UNAVAILABLE
	AttributeRPCGRPCStatusCodeUnavailable = "14"
	// DATA_LOSS
	AttributeRPCGRPCStatusCodeDataLoss = "15"
	// UNAUTHENTICATED
	AttributeRPCGRPCStatusCodeUnauthenticated = "16"
)

const (
	// gRPC
	AttributeRPCSystemGRPC = "grpc"
	// Java RMI
	AttributeRPCSystemJavaRmi = "java_rmi"
	// .NET WCF
	AttributeRPCSystemDotnetWcf = "dotnet_wcf"
	// Apache Dubbo
	AttributeRPCSystemApacheDubbo = "apache_dubbo"
	// Connect RPC
	AttributeRPCSystemConnectRPC = "connect_rpc"
)

// These attributes may be used to describe the server in a connection-based
// network interaction where there is one side that initiates the connection
// (the client is the side that initiates the connection). This covers all TCP
// network interactions since TCP is connection-based and one side initiates
// the connection (an exception is made for peer-to-peer communication over TCP
// where the "user-facing" surface of the protocol / API doesn't expose a clear
// notion of client and server). This also covers UDP network interactions
// where one side initiates the interaction, e.g. QUIC (HTTP/3) and DNS.
const (
	// Server domain name if available without reverse DNS lookup; otherwise, IP
	// address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the client side, and when communicating through an
	// intermediary, server.address SHOULD represent the server address behind any
	// intermediaries, for example proxies, if it's available.
	AttributeServerAddress = "server.address"
	// Server port number.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 80, 8080, 443
	// Note: When observed from the client side, and when communicating through an
	// intermediary, server.port SHOULD represent the server port behind any
	// intermediaries, for example proxies, if it's available.
	AttributeServerPort = "server.port"
)

// A service instance.
const (
	// The string ID of the service instance.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '627cc493-f310-47de-96bd-71410b7dec09'
	// Note: MUST be unique for each instance of the same
	// service.namespace,service.name pair (in other words
	// service.namespace,service.name,service.instance.id triplet MUST be globally
	// unique). The ID helps to
	// distinguish instances of the same service that exist at the same time (e.g.
	// instances of a horizontally scaled
	// service).Implementations, such as SDKs, are recommended to generate a random
	// Version 1 or Version 4 RFC
	// 4122 UUID, but are free to use an inherent unique ID as the source of
	// this value if stability is desirable. In that case, the ID SHOULD be used as
	// source of a UUID Version 5 and
	// SHOULD use the following UUID as the namespace: 4d63009a-8d0f-11ee-
	// aad7-4c796ed8e320.UUIDs are typically recommended, as only an opaque value for
	// the purposes of identifying a service instance is
	// needed. Similar to what can be seen in the man page for the
	// /etc/machine-id file, the underlying
	// data, such as pod name and namespace should be treated as confidential, being
	// the user's choice to expose it
	// or not via another resource attribute.For applications running behind an
	// application server (like unicorn), we do not recommend using one identifier
	// for all processes participating in the application. Instead, it's recommended
	// each division (e.g. a worker
	// thread in unicorn) to have its own instance.id.It's not recommended for a
	// Collector to set service.instance.id if it can't unambiguously determine the
	// service instance that is generating that telemetry. For instance, creating an
	// UUID based on pod.name will
	// likely be wrong, as the Collector might not know from which container within
	// that pod the telemetry originated.
	// However, Collectors can set the service.instance.id if they can unambiguously
	// determine the service instance
	// for that telemetry. This is typically the case for scraping receivers, as they
	// know the target address and
	// port.
	AttributeServiceInstanceID = "service.instance.id"
	// Logical name of the service.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'shoppingcart'
	// Note: MUST be the same for all instances of horizontally scaled services. If
	// the value was not specified, SDKs MUST fallback to unknown_service:
	// concatenated with process.executable.name, e.g. unknown_service:bash. If
	// process.executable.name is not available, the value MUST be set to
	// unknown_service.
	AttributeServiceName = "service.name"
	// A namespace for service.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Shop'
	// Note: A string value having a meaning that helps to distinguish a group of
	// services, for example the team name that owns a group of services. service.name
	// is expected to be unique within the same namespace. If service.namespace is not
	// specified in the Resource then service.name is expected to be unique for all
	// services that have no explicit namespace defined (so the empty/unspecified
	// namespace is simply one more valid namespace). Zero-length namespace string is
	// assumed equal to unspecified namespace.
	AttributeServiceNamespace = "service.namespace"
	// The version string of the service API or implementation. The format is not
	// defined by these conventions.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '2.0.0', 'a01dbef8a'
	AttributeServiceVersion = "service.version"
)

// Session is defined as the period of time encompassing all activities
// performed by the application and the actions executed by the end user.
// Consequently, a Session is represented as a collection of Logs, Events, and
// Spans emitted by the Client Application throughout the Session's duration.
// Each Session is assigned a unique identifier, which is included as an
// attribute in the Logs, Events, and Spans generated during the Session's
// lifecycle.
// When a session reaches end of life, typically due to user inactivity or
// session timeout, a new session identifier will be assigned. The previous
// session identifier may be provided by the instrumentation so that telemetry
// backends can link the two sessions.
const (
	// A unique id to identify a session.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '00112233-4455-6677-8899-aabbccddeeff'
	AttributeSessionID = "session.id"
	// The previous session.id for this user, when known.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '00112233-4455-6677-8899-aabbccddeeff'
	AttributeSessionPreviousID = "session.previous_id"
)

// These attributes may be used to describe the sender of a network
// exchange/packet. These should be used when there is no client/server
// relationship between the two sides, or when that relationship is unknown.
// This covers low-level network interactions (e.g. packet tracing) where you
// don't know if there was a connection or which side initiated it. This also
// covers unidirectional UDP flows and peer-to-peer communication where the
// "user-facing" surface of the protocol / API doesn't expose a clear notion of
// client and server.
const (
	// Source address - domain name if available without reverse DNS lookup;
	// otherwise, IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'source.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the destination side, and when communicating through
	// an intermediary, source.address SHOULD represent the source address behind any
	// intermediaries, for example proxies, if it's available.
	AttributeSourceAddress = "source.address"
	// Source port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeSourcePort = "source.port"
)

// Attributes for telemetry SDK.
const (
	// The language of the telemetry SDK.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: stable
	AttributeTelemetrySDKLanguage = "telemetry.sdk.language"
	// The name of the telemetry SDK as defined above.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: stable
	// Examples: 'opentelemetry'
	// Note: The OpenTelemetry SDK MUST set the telemetry.sdk.name attribute to
	// opentelemetry.
	// If another SDK, like a fork or a vendor-provided implementation, is used, this
	// SDK MUST set the
	// telemetry.sdk.name attribute to the fully-qualified class or module name of
	// this SDK's main entry point
	// or another suitable identifier depending on the language.
	// The identifier opentelemetry is reserved and MUST NOT be used in this case.
	// All custom identifiers SHOULD be stable across different versions of an
	// implementation.
	AttributeTelemetrySDKName = "telemetry.sdk.name"
	// The version string of the telemetry SDK.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: stable
	// Examples: '1.2.3'
	AttributeTelemetrySDKVersion = "telemetry.sdk.version"
	// The name of the auto instrumentation agent or distribution, if used.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'parts-unlimited-java'
	// Note: Official auto instrumentation agents and distributions SHOULD set the
	// telemetry.distro.name attribute to
	// a string starting with opentelemetry-, e.g. opentelemetry-java-instrumentation.
	AttributeTelemetryDistroName = "telemetry.distro.name"
	// The version string of the auto instrumentation agent or distribution, if used.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1.2.3'
	AttributeTelemetryDistroVersion = "telemetry.distro.version"
)

const (
	// cpp
	AttributeTelemetrySDKLanguageCPP = "cpp"
	// dotnet
	AttributeTelemetrySDKLanguageDotnet = "dotnet"
	// erlang
	AttributeTelemetrySDKLanguageErlang = "erlang"
	// go
	AttributeTelemetrySDKLanguageGo = "go"
	// java
	AttributeTelemetrySDKLanguageJava = "java"
	// nodejs
	AttributeTelemetrySDKLanguageNodejs = "nodejs"
	// php
	AttributeTelemetrySDKLanguagePHP = "php"
	// python
	AttributeTelemetrySDKLanguagePython = "python"
	// ruby
	AttributeTelemetrySDKLanguageRuby = "ruby"
	// rust
	AttributeTelemetrySDKLanguageRust = "rust"
	// swift
	AttributeTelemetrySDKLanguageSwift = "swift"
	// webjs
	AttributeTelemetrySDKLanguageWebjs = "webjs"
)

// These attributes may be used for any operation to store information about a
// thread that started a span.
const (
	// Current &quot;managed&quot; thread ID (as opposed to OS thread ID).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 42
	AttributeThreadID = "thread.id"
	// Current thread name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'main'
	AttributeThreadName = "thread.name"
)

// Semantic convention attributes in the TLS namespace.
const (
	// String indicating the cipher used during the current connection.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'TLS_RSA_WITH_3DES_EDE_CBC_SHA',
	// 'TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256'
	// Note: The values allowed for tls.cipher MUST be one of the Descriptions of the
	// registered TLS Cipher Suits.
	AttributeTLSCipher = "tls.cipher"
	// PEM-encoded stand-alone certificate offered by the client. This is usually
	// mutually-exclusive of client.certificate_chain since this value also exists in
	// that list.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MII...'
	AttributeTLSClientCertificate = "tls.client.certificate"
	// Array of PEM-encoded certificates that make up the certificate chain offered by
	// the client. This is usually mutually-exclusive of client.certificate since that
	// value should be the first certificate in the chain.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MII...', 'MI...'
	AttributeTLSClientCertificateChain = "tls.client.certificate_chain"
	// Certificate fingerprint using the MD5 digest of DER-encoded version of
	// certificate offered by the client. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0F76C7F2C55BFD7D8E8B8F4BFBF0C9EC'
	AttributeTLSClientHashMd5 = "tls.client.hash.md5"
	// Certificate fingerprint using the SHA1 digest of DER-encoded version of
	// certificate offered by the client. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '9E393D93138888D288266C2D915214D1D1CCEB2A'
	AttributeTLSClientHashSha1 = "tls.client.hash.sha1"
	// Certificate fingerprint using the SHA256 digest of DER-encoded version of
	// certificate offered by the client. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0687F666A054EF17A08E2F2162EAB4CBC0D265E1D7875BE74BF3C712CA92DAF0'
	AttributeTLSClientHashSha256 = "tls.client.hash.sha256"
	// Distinguished name of subject of the issuer of the x.509 certificate presented
	// by the client.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'CN=Example Root CA, OU=Infrastructure Team, DC=example, DC=com'
	AttributeTLSClientIssuer = "tls.client.issuer"
	// A hash that identifies clients based on how they perform an SSL/TLS handshake.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'd4e5b18d6b55c71272893221c96ba240'
	AttributeTLSClientJa3 = "tls.client.ja3"
	// Date/Time indicating when client certificate is no longer considered valid.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2021-01-01T00:00:00.000Z'
	AttributeTLSClientNotAfter = "tls.client.not_after"
	// Date/Time indicating when client certificate is first considered valid.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1970-01-01T00:00:00.000Z'
	AttributeTLSClientNotBefore = "tls.client.not_before"
	// Also called an SNI, this tells the server which hostname to which the client is
	// attempting to connect to.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry.io'
	AttributeTLSClientServerName = "tls.client.server_name"
	// Distinguished name of subject of the x.509 certificate presented by the client.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'CN=myclient, OU=Documentation Team, DC=example, DC=com'
	AttributeTLSClientSubject = "tls.client.subject"
	// Array of ciphers offered by the client during the client hello.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	// "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", "..."'
	AttributeTLSClientSupportedCiphers = "tls.client.supported_ciphers"
	// String indicating the curve used for the given cipher, when applicable
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'secp256r1'
	AttributeTLSCurve = "tls.curve"
	// Boolean flag indicating if the TLS negotiation was successful and transitioned
	// to an encrypted tunnel.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: True
	AttributeTLSEstablished = "tls.established"
	// String indicating the protocol being tunneled. Per the values in the IANA
	// registry, this string should be lower case.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'http/1.1'
	AttributeTLSNextProtocol = "tls.next_protocol"
	// Normalized lowercase protocol name parsed from original string of the
	// negotiated SSL/TLS protocol version
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeTLSProtocolName = "tls.protocol.name"
	// Numeric part of the version parsed from the original string of the negotiated
	// SSL/TLS protocol version
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1.2', '3'
	AttributeTLSProtocolVersion = "tls.protocol.version"
	// Boolean flag indicating if this TLS connection was resumed from an existing TLS
	// negotiation.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: True
	AttributeTLSResumed = "tls.resumed"
	// PEM-encoded stand-alone certificate offered by the server. This is usually
	// mutually-exclusive of server.certificate_chain since this value also exists in
	// that list.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MII...'
	AttributeTLSServerCertificate = "tls.server.certificate"
	// Array of PEM-encoded certificates that make up the certificate chain offered by
	// the server. This is usually mutually-exclusive of server.certificate since that
	// value should be the first certificate in the chain.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MII...', 'MI...'
	AttributeTLSServerCertificateChain = "tls.server.certificate_chain"
	// Certificate fingerprint using the MD5 digest of DER-encoded version of
	// certificate offered by the server. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0F76C7F2C55BFD7D8E8B8F4BFBF0C9EC'
	AttributeTLSServerHashMd5 = "tls.server.hash.md5"
	// Certificate fingerprint using the SHA1 digest of DER-encoded version of
	// certificate offered by the server. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '9E393D93138888D288266C2D915214D1D1CCEB2A'
	AttributeTLSServerHashSha1 = "tls.server.hash.sha1"
	// Certificate fingerprint using the SHA256 digest of DER-encoded version of
	// certificate offered by the server. For consistency with other hash values, this
	// value should be formatted as an uppercase hash.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0687F666A054EF17A08E2F2162EAB4CBC0D265E1D7875BE74BF3C712CA92DAF0'
	AttributeTLSServerHashSha256 = "tls.server.hash.sha256"
	// Distinguished name of subject of the issuer of the x.509 certificate presented
	// by the client.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'CN=Example Root CA, OU=Infrastructure Team, DC=example, DC=com'
	AttributeTLSServerIssuer = "tls.server.issuer"
	// A hash that identifies servers based on how they perform an SSL/TLS handshake.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'd4e5b18d6b55c71272893221c96ba240'
	AttributeTLSServerJa3s = "tls.server.ja3s"
	// Date/Time indicating when server certificate is no longer considered valid.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2021-01-01T00:00:00.000Z'
	AttributeTLSServerNotAfter = "tls.server.not_after"
	// Date/Time indicating when server certificate is first considered valid.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1970-01-01T00:00:00.000Z'
	AttributeTLSServerNotBefore = "tls.server.not_before"
	// Distinguished name of subject of the x.509 certificate presented by the server.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'CN=myserver, OU=Documentation Team, DC=example, DC=com'
	AttributeTLSServerSubject = "tls.server.subject"
)

const (
	// ssl
	AttributeTLSProtocolNameSsl = "ssl"
	// tls
	AttributeTLSProtocolNameTLS = "tls"
)

// Attributes describing URL.
const (
	// Domain extracted from the url.full, such as &quot;opentelemetry.io&quot;.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'www.foo.bar', 'opentelemetry.io', '3.12.167.2',
	// '[1080:0:0:0:8:800:200C:417A]'
	// Note: In some cases a URL may refer to an IP and/or port directly, without a
	// domain name. In this case, the IP address would go to the domain field. If the
	// URL contains a literal IPv6 address enclosed by [ and ], the [ and ] characters
	// should also be captured in the domain field.
	AttributeURLDomain = "url.domain"
	// The file extension extracted from the url.full, excluding the leading dot.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'png', 'gz'
	// Note: The file extension is only set if it exists, as not every url has a file
	// extension. When the file name has multiple extensions example.tar.gz, only the
	// last one should be captured gz, not tar.gz.
	AttributeURLExtension = "url.extension"
	// The URI fragment component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'SemConv'
	AttributeURLFragment = "url.fragment"
	// Absolute URL describing a network resource according to RFC3986
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv', '//localhost'
	// Note: For network calls, URL usually has
	// scheme://host[:port][path][?query][#fragment] format, where the fragment is not
	// transmitted over HTTP, but if it is known, it SHOULD be included nevertheless.
	// url.full MUST NOT contain credentials passed via URL in form of
	// https://username:password@www.example.com/. In such case username and password
	// SHOULD be redacted and attribute's value SHOULD be
	// https://REDACTED:REDACTED@www.example.com/.
	// url.full SHOULD capture the absolute URL when it is available (or can be
	// reconstructed). Sensitive content provided in url.full SHOULD be scrubbed when
	// instrumentations can identify it.
	AttributeURLFull = "url.full"
	// Unmodified original URL as seen in the event source.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv',
	// 'search?q=OpenTelemetry'
	// Note: In network monitoring, the observed URL may be a full URL, whereas in
	// access logs, the URL is often just represented as a path. This field is meant
	// to represent the URL as it was observed, complete or not.
	// url.original might contain credentials passed via URL in form of
	// https://username:password@www.example.com/. In such case password and username
	// SHOULD NOT be redacted and attribute's value SHOULD remain the same.
	AttributeURLOriginal = "url.original"
	// The URI path component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '/search'
	// Note: Sensitive content provided in url.path SHOULD be scrubbed when
	// instrumentations can identify it.
	AttributeURLPath = "url.path"
	// Port extracted from the url.full
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 443
	AttributeURLPort = "url.port"
	// The URI query component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'q=OpenTelemetry'
	// Note: Sensitive content provided in url.query SHOULD be scrubbed when
	// instrumentations can identify it.
	AttributeURLQuery = "url.query"
	// The highest registered url domain, stripped of the subdomain.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'example.com', 'foo.co.uk'
	// Note: This value can be determined precisely with the public suffix list. For
	// example, the registered domain for foo.example.com is example.com. Trying to
	// approximate this by simply taking the last two labels will not work well for
	// TLDs such as co.uk.
	AttributeURLRegisteredDomain = "url.registered_domain"
	// The URI scheme component identifying the used protocol.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'https', 'ftp', 'telnet'
	AttributeURLScheme = "url.scheme"
	// The subdomain portion of a fully qualified domain name includes all of the
	// names except the host name under the registered_domain. In a partially
	// qualified domain, or if the qualification level of the full name cannot be
	// determined, subdomain contains all of the names below the registered domain.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'east', 'sub2.sub1'
	// Note: The subdomain portion of www.east.mydomain.co.uk is east. If the domain
	// has multiple levels of subdomain, such as sub2.sub1.example.com, the subdomain
	// field should contain sub2.sub1, with no trailing period.
	AttributeURLSubdomain = "url.subdomain"
	// The effective top level domain (eTLD), also known as the domain suffix, is the
	// last part of the domain name. For example, the top level domain for example.com
	// is com.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'com', 'co.uk'
	// Note: This value can be determined precisely with the public suffix list.
	AttributeURLTopLevelDomain = "url.top_level_domain"
)

// Describes user-agent attributes.
const (
	// Name of the user-agent extracted from original. Usually refers to the browser's
	// name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Safari', 'YourApp'
	// Note: Example of extracting browser's name from original string. In the case of
	// using a user-agent for non-browser products, such as microservices with
	// multiple names/versions inside the user_agent.original, the most significant
	// name SHOULD be selected. In such a scenario it should align with
	// user_agent.version
	AttributeUserAgentName = "user_agent.name"
	// Value of the HTTP User-Agent header sent by the client.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'CERN-LineMode/2.15 libwww/2.17b3', 'Mozilla/5.0 (iPhone; CPU iPhone
	// OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko)
	// Version/14.1.2 Mobile/15E148 Safari/604.1', 'YourApp/1.0.0 grpc-java-
	// okhttp/1.27.2'
	AttributeUserAgentOriginal = "user_agent.original"
	// Version of the user-agent extracted from original. Usually refers to the
	// browser's version
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '14.1.2', '1.0.0'
	// Note: Example of extracting browser's version from original string. In the case
	// of using a user-agent for non-browser products, such as microservices with
	// multiple names/versions inside the user_agent.original, the most significant
	// version SHOULD be selected. In such a scenario it should align with
	// user_agent.name
	AttributeUserAgentVersion = "user_agent.version"
)

func GetAttribute_groupSemanticConventionAttributeNames() []string {
	return []string{
		AttributeEventName,
		AttributeLogRecordUID,
		AttributeLogIostream,
		AttributeLogFileName,
		AttributeLogFileNameResolved,
		AttributeLogFilePath,
		AttributeLogFilePathResolved,
		AttributePoolName,
		AttributeState,
		AttributeAspnetcoreRateLimitingResult,
		AttributeAspnetcoreDiagnosticsHandlerType,
		AttributeAspnetcoreRateLimitingPolicy,
		AttributeAspnetcoreRequestIsUnhandled,
		AttributeAspnetcoreRoutingIsFallback,
		AttributeSignalrConnectionStatus,
		AttributeSignalrTransport,
		AttributeJvmBufferPoolName,
		AttributeJvmMemoryPoolName,
		AttributeJvmMemoryType,
		AttributeProcessCPUState,
		AttributeSystemDevice,
		AttributeSystemCPULogicalNumber,
		AttributeSystemCPUState,
		AttributeSystemMemoryState,
		AttributeSystemPagingDirection,
		AttributeSystemPagingState,
		AttributeSystemPagingType,
		AttributeSystemFilesystemMode,
		AttributeSystemFilesystemMountpoint,
		AttributeSystemFilesystemState,
		AttributeSystemFilesystemType,
		AttributeSystemNetworkState,
		AttributeSystemProcessStatus,
		AttributeAndroidOSAPILevel,
		AttributeAWSDynamoDBAttributeDefinitions,
		AttributeAWSDynamoDBAttributesToGet,
		AttributeAWSDynamoDBConsistentRead,
		AttributeAWSDynamoDBConsumedCapacity,
		AttributeAWSDynamoDBCount,
		AttributeAWSDynamoDBExclusiveStartTable,
		AttributeAWSDynamoDBGlobalSecondaryIndexUpdates,
		AttributeAWSDynamoDBGlobalSecondaryIndexes,
		AttributeAWSDynamoDBIndexName,
		AttributeAWSDynamoDBItemCollectionMetrics,
		AttributeAWSDynamoDBLimit,
		AttributeAWSDynamoDBLocalSecondaryIndexes,
		AttributeAWSDynamoDBProjection,
		AttributeAWSDynamoDBProvisionedReadCapacity,
		AttributeAWSDynamoDBProvisionedWriteCapacity,
		AttributeAWSDynamoDBScanForward,
		AttributeAWSDynamoDBScannedCount,
		AttributeAWSDynamoDBSegment,
		AttributeAWSDynamoDBSelect,
		AttributeAWSDynamoDBTableCount,
		AttributeAWSDynamoDBTableNames,
		AttributeAWSDynamoDBTotalSegments,
		AttributeBrowserBrands,
		AttributeBrowserLanguage,
		AttributeBrowserMobile,
		AttributeBrowserPlatform,
		AttributeClientAddress,
		AttributeClientPort,
		AttributeCloudAccountID,
		AttributeCloudAvailabilityZone,
		AttributeCloudPlatform,
		AttributeCloudProvider,
		AttributeCloudRegion,
		AttributeCloudResourceID,
		AttributeCloudeventsEventID,
		AttributeCloudeventsEventSource,
		AttributeCloudeventsEventSpecVersion,
		AttributeCloudeventsEventSubject,
		AttributeCloudeventsEventType,
		AttributeCodeColumn,
		AttributeCodeFilepath,
		AttributeCodeFunction,
		AttributeCodeLineNumber,
		AttributeCodeNamespace,
		AttributeCodeStacktrace,
		AttributeContainerCommand,
		AttributeContainerCommandArgs,
		AttributeContainerCommandLine,
		AttributeContainerCPUState,
		AttributeContainerID,
		AttributeContainerImageID,
		AttributeContainerImageName,
		AttributeContainerImageRepoDigests,
		AttributeContainerImageTags,
		AttributeContainerName,
		AttributeContainerRuntime,
		AttributeDBCassandraConsistencyLevel,
		AttributeDBCassandraCoordinatorDC,
		AttributeDBCassandraCoordinatorID,
		AttributeDBCassandraIdempotence,
		AttributeDBCassandraPageSize,
		AttributeDBCassandraSpeculativeExecutionCount,
		AttributeDBCassandraTable,
		AttributeDBCosmosDBClientID,
		AttributeDBCosmosDBConnectionMode,
		AttributeDBCosmosDBContainer,
		AttributeDBCosmosDBOperationType,
		AttributeDBCosmosDBRequestCharge,
		AttributeDBCosmosDBRequestContentLength,
		AttributeDBCosmosDBStatusCode,
		AttributeDBCosmosDBSubStatusCode,
		AttributeDBElasticsearchClusterName,
		AttributeDBInstanceID,
		AttributeDBMongoDBCollection,
		AttributeDBMSSQLInstanceName,
		AttributeDBName,
		AttributeDBOperation,
		AttributeDBRedisDBIndex,
		AttributeDBSQLTable,
		AttributeDBStatement,
		AttributeDBSystem,
		AttributeDBUser,
		AttributeDeploymentEnvironment,
		AttributeDBConnectionString,
		AttributeDBElasticsearchNodeName,
		AttributeDBJDBCDriverClassname,
		AttributeHTTPFlavor,
		AttributeHTTPMethod,
		AttributeHTTPRequestContentLength,
		AttributeHTTPResponseContentLength,
		AttributeHTTPScheme,
		AttributeHTTPStatusCode,
		AttributeHTTPTarget,
		AttributeHTTPURL,
		AttributeHTTPUserAgent,
		AttributeMessagingKafkaDestinationPartition,
		AttributeNetHostName,
		AttributeNetHostPort,
		AttributeNetPeerName,
		AttributeNetPeerPort,
		AttributeNetProtocolName,
		AttributeNetProtocolVersion,
		AttributeNetSockFamily,
		AttributeNetSockHostAddr,
		AttributeNetSockHostPort,
		AttributeNetSockPeerAddr,
		AttributeNetSockPeerName,
		AttributeNetSockPeerPort,
		AttributeNetTransport,
		AttributeSystemProcessesStatus,
		AttributeDestinationAddress,
		AttributeDestinationPort,
		AttributeDeviceID,
		AttributeDeviceManufacturer,
		AttributeDeviceModelIdentifier,
		AttributeDeviceModelName,
		AttributeDiskIoDirection,
		AttributeDNSQuestionName,
		AttributeEnduserID,
		AttributeEnduserRole,
		AttributeEnduserScope,
		AttributeErrorType,
		AttributeExceptionEscaped,
		AttributeExceptionMessage,
		AttributeExceptionStacktrace,
		AttributeExceptionType,
		AttributeFaaSColdstart,
		AttributeFaaSCron,
		AttributeFaaSDocumentCollection,
		AttributeFaaSDocumentName,
		AttributeFaaSDocumentOperation,
		AttributeFaaSDocumentTime,
		AttributeFaaSInstance,
		AttributeFaaSInvocationID,
		AttributeFaaSInvokedName,
		AttributeFaaSInvokedProvider,
		AttributeFaaSInvokedRegion,
		AttributeFaaSMaxMemory,
		AttributeFaaSName,
		AttributeFaaSTime,
		AttributeFaaSTrigger,
		AttributeFaaSVersion,
		AttributeFeatureFlagKey,
		AttributeFeatureFlagProviderName,
		AttributeFeatureFlagVariant,
		AttributeFileDirectory,
		AttributeFileExtension,
		AttributeFileName,
		AttributeFilePath,
		AttributeFileSize,
		AttributeGCPCloudRunJobExecution,
		AttributeGCPCloudRunJobTaskIndex,
		AttributeGCPGceInstanceHostname,
		AttributeGCPGceInstanceName,
		AttributeHostArch,
		AttributeHostCPUCacheL2Size,
		AttributeHostCPUFamily,
		AttributeHostCPUModelID,
		AttributeHostCPUModelName,
		AttributeHostCPUStepping,
		AttributeHostCPUVendorID,
		AttributeHostID,
		AttributeHostImageID,
		AttributeHostImageName,
		AttributeHostImageVersion,
		AttributeHostIP,
		AttributeHostMac,
		AttributeHostName,
		AttributeHostType,
		AttributeHTTPConnectionState,
		AttributeHTTPRequestBodySize,
		AttributeHTTPRequestMethod,
		AttributeHTTPRequestMethodOriginal,
		AttributeHTTPRequestResendCount,
		AttributeHTTPRequestSize,
		AttributeHTTPResponseBodySize,
		AttributeHTTPResponseSize,
		AttributeHTTPResponseStatusCode,
		AttributeHTTPRoute,
		AttributeK8SClusterName,
		AttributeK8SClusterUID,
		AttributeK8SContainerName,
		AttributeK8SContainerRestartCount,
		AttributeK8SCronJobName,
		AttributeK8SCronJobUID,
		AttributeK8SDaemonSetName,
		AttributeK8SDaemonSetUID,
		AttributeK8SDeploymentName,
		AttributeK8SDeploymentUID,
		AttributeK8SJobName,
		AttributeK8SJobUID,
		AttributeK8SNamespaceName,
		AttributeK8SNodeName,
		AttributeK8SNodeUID,
		AttributeK8SPodName,
		AttributeK8SPodUID,
		AttributeK8SReplicaSetName,
		AttributeK8SReplicaSetUID,
		AttributeK8SStatefulSetName,
		AttributeK8SStatefulSetUID,
		AttributeMessagingBatchMessageCount,
		AttributeMessagingClientID,
		AttributeMessagingDestinationAnonymous,
		AttributeMessagingDestinationName,
		AttributeMessagingDestinationPartitionID,
		AttributeMessagingDestinationTemplate,
		AttributeMessagingDestinationTemporary,
		AttributeMessagingDestinationPublishAnonymous,
		AttributeMessagingDestinationPublishName,
		AttributeMessagingEventhubsConsumerGroup,
		AttributeMessagingEventhubsMessageEnqueuedTime,
		AttributeMessagingGCPPubsubMessageOrderingKey,
		AttributeMessagingKafkaConsumerGroup,
		AttributeMessagingKafkaMessageKey,
		AttributeMessagingKafkaMessageOffset,
		AttributeMessagingKafkaMessageTombstone,
		AttributeMessagingMessageBodySize,
		AttributeMessagingMessageConversationID,
		AttributeMessagingMessageEnvelopeSize,
		AttributeMessagingMessageID,
		AttributeMessagingOperation,
		AttributeMessagingRabbitmqDestinationRoutingKey,
		AttributeMessagingRabbitmqMessageDeliveryTag,
		AttributeMessagingRocketmqClientGroup,
		AttributeMessagingRocketmqConsumptionModel,
		AttributeMessagingRocketmqMessageDelayTimeLevel,
		AttributeMessagingRocketmqMessageDeliveryTimestamp,
		AttributeMessagingRocketmqMessageGroup,
		AttributeMessagingRocketmqMessageKeys,
		AttributeMessagingRocketmqMessageTag,
		AttributeMessagingRocketmqMessageType,
		AttributeMessagingRocketmqNamespace,
		AttributeMessagingServicebusDestinationSubscriptionName,
		AttributeMessagingServicebusDispositionStatus,
		AttributeMessagingServicebusMessageDeliveryCount,
		AttributeMessagingServicebusMessageEnqueuedTime,
		AttributeMessagingSystem,
		AttributeNetworkCarrierIcc,
		AttributeNetworkCarrierMcc,
		AttributeNetworkCarrierMnc,
		AttributeNetworkCarrierName,
		AttributeNetworkConnectionSubtype,
		AttributeNetworkConnectionType,
		AttributeNetworkIoDirection,
		AttributeNetworkLocalAddress,
		AttributeNetworkLocalPort,
		AttributeNetworkPeerAddress,
		AttributeNetworkPeerPort,
		AttributeNetworkProtocolName,
		AttributeNetworkProtocolVersion,
		AttributeNetworkTransport,
		AttributeNetworkType,
		AttributeOciManifestDigest,
		AttributeOSBuildID,
		AttributeOSDescription,
		AttributeOSName,
		AttributeOSType,
		AttributeOSVersion,
		AttributeProcessCommand,
		AttributeProcessCommandArgs,
		AttributeProcessCommandLine,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessOwner,
		AttributeProcessParentPID,
		AttributeProcessPID,
		AttributeProcessRuntimeDescription,
		AttributeProcessRuntimeName,
		AttributeProcessRuntimeVersion,
		AttributeRPCConnectRPCErrorCode,
		AttributeRPCGRPCStatusCode,
		AttributeRPCJsonrpcErrorCode,
		AttributeRPCJsonrpcErrorMessage,
		AttributeRPCJsonrpcRequestID,
		AttributeRPCJsonrpcVersion,
		AttributeRPCMethod,
		AttributeRPCService,
		AttributeRPCSystem,
		AttributeServerAddress,
		AttributeServerPort,
		AttributeServiceInstanceID,
		AttributeServiceName,
		AttributeServiceNamespace,
		AttributeServiceVersion,
		AttributeSessionID,
		AttributeSessionPreviousID,
		AttributeSourceAddress,
		AttributeSourcePort,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKVersion,
		AttributeTelemetryDistroName,
		AttributeTelemetryDistroVersion,
		AttributeThreadID,
		AttributeThreadName,
		AttributeTLSCipher,
		AttributeTLSClientCertificate,
		AttributeTLSClientCertificateChain,
		AttributeTLSClientHashMd5,
		AttributeTLSClientHashSha1,
		AttributeTLSClientHashSha256,
		AttributeTLSClientIssuer,
		AttributeTLSClientJa3,
		AttributeTLSClientNotAfter,
		AttributeTLSClientNotBefore,
		AttributeTLSClientServerName,
		AttributeTLSClientSubject,
		AttributeTLSClientSupportedCiphers,
		AttributeTLSCurve,
		AttributeTLSEstablished,
		AttributeTLSNextProtocol,
		AttributeTLSProtocolName,
		AttributeTLSProtocolVersion,
		AttributeTLSResumed,
		AttributeTLSServerCertificate,
		AttributeTLSServerCertificateChain,
		AttributeTLSServerHashMd5,
		AttributeTLSServerHashSha1,
		AttributeTLSServerHashSha256,
		AttributeTLSServerIssuer,
		AttributeTLSServerJa3s,
		AttributeTLSServerNotAfter,
		AttributeTLSServerNotBefore,
		AttributeTLSServerSubject,
		AttributeURLDomain,
		AttributeURLExtension,
		AttributeURLFragment,
		AttributeURLFull,
		AttributeURLOriginal,
		AttributeURLPath,
		AttributeURLPort,
		AttributeURLQuery,
		AttributeURLRegisteredDomain,
		AttributeURLScheme,
		AttributeURLSubdomain,
		AttributeURLTopLevelDomain,
		AttributeUserAgentName,
		AttributeUserAgentOriginal,
		AttributeUserAgentVersion,
	}
}
