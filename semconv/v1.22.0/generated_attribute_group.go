// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// These attributes may be used to describe the client in a connection-based
// network interaction where there is one side that initiates the connection
// (the client is the side that initiates the connection). This covers all TCP
// network interactions since TCP is connection-based and one side initiates
// the connection (an exception is made for peer-to-peer communication over TCP
// where the "user-facing" surface of the protocol / API does not expose a
// clear notion of client and server). This also covers UDP network
// interactions where one side initiates the interaction, e.g. QUIC (HTTP/3)
// and DNS.
const (
	// Client address - domain name if available without reverse DNS lookup, otherwise
	// IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'client.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.address SHOULD represent the client address behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeClientAddress = "client.address"
	// Client port number.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 65123
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.port SHOULD represent the client port behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeClientPort = "client.port"
)

// These attributes may be used for any network related operation.
const (
	// Deprecated, use server.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'example.com'
	AttributeNetHostName = "net.host.name"
	// Deprecated, use server.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 8080
	AttributeNetHostPort = "net.host.port"
	// Deprecated, use server.address on client spans and client.address on server
	// spans.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'example.com'
	AttributeNetPeerName = "net.peer.name"
	// Deprecated, use server.port on client spans and client.port on server spans.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 8080
	AttributeNetPeerPort = "net.peer.port"
	// Deprecated, use network.protocol.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'amqp', 'http', 'mqtt'
	AttributeNetProtocolName = "net.protocol.name"
	// Deprecated, use network.protocol.version.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '3.1.1'
	AttributeNetProtocolVersion = "net.protocol.version"
	// Deprecated, use network.transport and network.type.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: deprecated
	AttributeNetSockFamily = "net.sock.family"
	// Deprecated, use network.local.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '/var/my.sock'
	AttributeNetSockHostAddr = "net.sock.host.addr"
	// Deprecated, use network.local.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 8080
	AttributeNetSockHostPort = "net.sock.host.port"
	// Deprecated, use network.peer.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '192.168.0.1'
	AttributeNetSockPeerAddr = "net.sock.peer.addr"
	// Deprecated, no replacement at this time.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '/var/my.sock'
	AttributeNetSockPeerName = "net.sock.peer.name"
	// Deprecated, use network.peer.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 65531
	AttributeNetSockPeerPort = "net.sock.peer.port"
	// Deprecated, use network.transport.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: deprecated
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

// These attributes may be used to describe the receiver of a network
// exchange/packet. These should be used when there is no client/server
// relationship between the two sides, or when that relationship is unknown.
// This covers low-level network interactions (e.g. packet tracing) where you
// don't know if there was a connection or which side initiated it. This also
// covers unidirectional UDP flows and peer-to-peer communication where the
// "user-facing" surface of the protocol / API does not expose a clear notion
// of client and server.
const (
	// Destination address - domain name if available without reverse DNS lookup,
	// otherwise IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'destination.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the source side, and when communicating through an
	// intermediary, destination.address SHOULD represent the destination address
	// behind any intermediaries (e.g. proxies) if it's available.
	AttributeDestinationAddress = "destination.address"
	// Destination port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeDestinationPort = "destination.port"
)

// The shared attributes used to report an error.
const (
	// Describes a class of error the operation ended with.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'timeout', 'java.net.UnknownHostException',
	// 'server_certificate_invalid', '500'
	// Note: The error.type SHOULD be predictable and SHOULD have low cardinality.
	// Instrumentations SHOULD document the list of errors they report.The cardinality
	// of error.type within one instrumentation library SHOULD be low, but
	// telemetry consumers that aggregate data from multiple instrumentation libraries
	// and applications
	// should be prepared for error.type to have high cardinality at query time, when
	// no
	// additional filters are applied.If the operation has completed successfully,
	// instrumentations SHOULD NOT set error.type.If a specific domain defines its own
	// set of error codes (such as HTTP or gRPC status codes),
	// it's RECOMMENDED to use a domain-specific attribute and also set error.type to
	// capture
	// all errors, regardless of whether they are defined within the domain-specific
	// set or not.
	AttributeErrorType = "error.type"
)

const (
	// A fallback error value to be used when the instrumentation does not define a custom value for it
	AttributeErrorTypeOther = "_OTHER"
)

// Describes FaaS attributes.
const (
	// The name of the invoked function.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'my-function'
	// Note: SHOULD be equal to the faas.name resource attribute of the invoked
	// function.
	AttributeFaaSInvokedName = "faas.invoked_name"
	// The cloud provider of the invoked function.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Note: SHOULD be equal to the cloud.provider resource attribute of the invoked
	// function.
	AttributeFaaSInvokedProvider = "faas.invoked_provider"
	// The cloud region of the invoked function.
	//
	// Type: string
	// Requirement Level: Conditionally Required - For some cloud providers, like AWS
	// or GCP, the region in which a function is hosted is essential to uniquely
	// identify the function and also part of its endpoint. Since it's part of the
	// endpoint being called, the region is always known to clients. In these cases,
	// `faas.invoked_region` MUST be set accordingly. If the region is unknown to the
	// client or not required for identifying the invoked function, setting
	// `faas.invoked_region` is optional.
	// Stability: experimental
	// Examples: 'eu-central-1'
	// Note: SHOULD be equal to the cloud.region resource attribute of the invoked
	// function.
	AttributeFaaSInvokedRegion = "faas.invoked_region"
	// Type of the trigger which caused this function invocation.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFaaSTrigger = "faas.trigger"
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

// Attributes for Events represented using Log Records.
const (
	// The domain identifies the business context for the events.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Note: Events across different domains may have same event.name, yet be
	// unrelated events.
	AttributeEventDomain = "event.domain"
	// The name identifies the event.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'click', 'exception'
	AttributeEventName = "event.name"
)

const (
	// Events from browser apps
	AttributeEventDomainBrowser = "browser"
	// Events from mobile apps
	AttributeEventDomainDevice = "device"
	// Events from Kubernetes
	AttributeEventDomainK8S = "k8s"
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
	// case the connection pool implementation does not provide a name, then the
	// db.connection_string should be used
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
	// Stability: experimental
	// Examples: 'G1 Old Gen', 'G1 Eden space', 'G1 Survivor Space'
	// Note: Pool names are generally obtained via MemoryPoolMXBean#getName().
	AttributeJvmMemoryPoolName = "jvm.memory.pool.name"
	// The type of memory.
	//
	// Type: Enum
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'heap', 'non_heap'
	AttributeJvmMemoryType = "jvm.memory.type"
)

const (
	// Heap memory
	AttributeJvmMemoryTypeHeap = "heap"
	// Non-heap memory
	AttributeJvmMemoryTypeNonHeap = "non_heap"
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
	// The state of the CPU
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
	// total
	AttributeSystemMemoryStateTotal = "total"
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

// Describes System Disk metric attributes
const (
	// The disk operation direction
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'read'
	AttributeSystemDiskDirection = "system.disk.direction"
)

const (
	// read
	AttributeSystemDiskDirectionRead = "read"
	// write
	AttributeSystemDiskDirectionWrite = "write"
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
	//
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'transmit'
	AttributeSystemNetworkDirection = "system.network.direction"
	// A stateless protocol MUST NOT set this attribute
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'close_wait'
	AttributeSystemNetworkState = "system.network.state"
)

const (
	// transmit
	AttributeSystemNetworkDirectionTransmit = "transmit"
	// receive
	AttributeSystemNetworkDirectionReceive = "receive"
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

// These attributes may be used for any network related operation.
const (
	// Local address of the network connection - IP address or Unix domain socket
	// name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '10.1.2.80', '/tmp/my.sock'
	AttributeNetworkLocalAddress = "network.local.address"
	// Local port number of the network connection.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 65123
	AttributeNetworkLocalPort = "network.local.port"
	// Peer address of the network connection - IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '10.1.2.80', '/tmp/my.sock'
	AttributeNetworkPeerAddress = "network.peer.address"
	// Peer port number of the network connection.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 65123
	AttributeNetworkPeerPort = "network.peer.port"
	// OSI application layer or non-OSI equivalent.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'amqp', 'http', 'mqtt'
	// Note: The value SHOULD be normalized to lowercase.
	AttributeNetworkProtocolName = "network.protocol.name"
	// Version of the protocol specified in network.protocol.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '3.1.1'
	// Note: network.protocol.version refers to the version of the protocol used and
	// might be different from the protocol client's version. If the HTTP client used
	// has a version of 0.27.2, but sends HTTP version 1.1, this attribute should be
	// set to 1.1.
	AttributeNetworkProtocolVersion = "network.protocol.version"
	// OSI transport layer or inter-process communication method.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'tcp', 'udp'
	// Note: The value SHOULD be normalized to lowercase.Consider always setting the
	// transport when setting a port number, since
	// a port number is ambiguous without knowing the transport, for example
	// different processes could be listening on TCP port 12345 and UDP port 12345.
	AttributeNetworkTransport = "network.transport"
	// OSI network layer or non-OSI equivalent.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ipv4', 'ipv6'
	// Note: The value SHOULD be normalized to lowercase.
	AttributeNetworkType = "network.type"
)

const (
	// TCP
	AttributeNetworkTransportTCP = "tcp"
	// UDP
	AttributeNetworkTransportUDP = "udp"
	// Named or anonymous pipe. See note below
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

// Describes deprecated HTTP attributes.
const (
	// Deprecated, use http.request.method instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'GET', 'POST', 'HEAD'
	AttributeHTTPMethod = "http.method"
	// Deprecated, use http.request.body.size instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 3495
	AttributeHTTPRequestContentLength = "http.request_content_length"
	// Deprecated, use http.response.body.size instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 3495
	AttributeHTTPResponseContentLength = "http.response_content_length"
	// Deprecated, use url.scheme instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'http', 'https'
	AttributeHTTPScheme = "http.scheme"
	// Deprecated, use http.response.status_code instead.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 200
	AttributeHTTPStatusCode = "http.status_code"
	// Deprecated, use url.path and url.query instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '/search?q=OpenTelemetry#SemConv'
	AttributeHTTPTarget = "http.target"
	// Deprecated, use url.full instead.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv'
	AttributeHTTPURL = "http.url"
)

// Semantic convention attributes in the HTTP namespace.
const (
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
	// Stability: experimental
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
	// Stability: experimental
	// Examples: 'GeT', 'ACL', 'foo'
	AttributeHTTPRequestMethodOriginal = "http.request.method_original"
	// The ordinal number of request resending attempt (for any reason, including
	// redirects).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3
	// Note: The resend count SHOULD be updated each time an HTTP request gets resent
	// by the client, regardless of what was the cause of the resending (e.g.
	// redirection, authorization failure, 503 Server Unavailable, network issues, or
	// any other).
	AttributeHTTPResendCount = "http.resend_count"
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
	// HTTP response status code.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 200
	AttributeHTTPResponseStatusCode = "http.response.status_code"
	// The matched route (path template in the format used by the respective server
	// framework). See note below
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/users/:userID?', '{controller}/{action}/{id?}'
	// Note: MUST NOT be populated when this is not supported by the HTTP server
	// framework as the route attribute should have low-cardinality and the URI path
	// can NOT substitute it.
	// SHOULD include the application root if there is one.
	AttributeHTTPRoute = "http.route"
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

// These attributes may be used to describe the server in a connection-based
// network interaction where there is one side that initiates the connection
// (the client is the side that initiates the connection). This covers all TCP
// network interactions since TCP is connection-based and one side initiates
// the connection (an exception is made for peer-to-peer communication over TCP
// where the "user-facing" surface of the protocol / API does not expose a
// clear notion of client and server). This also covers UDP network
// interactions where one side initiates the interaction, e.g. QUIC (HTTP/3)
// and DNS.
const (
	// Server address - domain name if available without reverse DNS lookup, otherwise
	// IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the client side, and when communicating through an
	// intermediary, server.address SHOULD represent
	// the server address behind any intermediaries (e.g. proxies) if it's available.
	AttributeServerAddress = "server.address"
	// Server port number.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 80, 8080, 443
	// Note: When observed from the client side, and when communicating through an
	// intermediary, server.port SHOULD represent the server port behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeServerPort = "server.port"
)

// Session is defined as the period of time encompassing all activities
// performed by the application and the actions executed by the end user.
// Consequently, a Session is represented as a collection of Logs, Events, and
// Spans emitted by the Client Application throughout the Session's duration.
// Each Session is assigned a unique identifier, which is included as an
// attribute in the Logs, Events, and Spans generated during the Session's
// lifecycle.
const (
	// A unique id to identify a session.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '00112233-4455-6677-8899-aabbccddeeff'
	AttributeSessionID = "session.id"
)

// These attributes may be used to describe the sender of a network
// exchange/packet. These should be used when there is no client/server
// relationship between the two sides, or when that relationship is unknown.
// This covers low-level network interactions (e.g. packet tracing) where you
// don't know if there was a connection or which side initiated it. This also
// covers unidirectional UDP flows and peer-to-peer communication where the
// "user-facing" surface of the protocol / API does not expose a clear notion
// of client and server.
const (
	// Source address - domain name if available without reverse DNS lookup, otherwise
	// IP address or Unix domain socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'source.example.com', '10.1.2.80', '/tmp/my.sock'
	// Note: When observed from the destination side, and when communicating through
	// an intermediary, source.address SHOULD represent the source address behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeSourceAddress = "source.address"
	// Source port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeSourcePort = "source.port"
)

// Semantic convention describing per-message attributes populated on messaging
// spans or links.
const (
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
)

// Semantic convention for attributes that describe messaging destination on
// broker
const (
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
	// the broker does not have such notion, the destination name SHOULD uniquely
	// identify the broker.
	AttributeMessagingDestinationName = "messaging.destination.name"
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
)

// Semantic convention for attributes that describe the publish messaging
// destination on broker. The term Publish Destination refers to the
// destination the message was originally published to. These attributes should
// be used on the consumer side when information about the publish destination
// is available and different than the destination message are consumed from.
const (
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
	// the broker does not have such notion, the original destination name SHOULD
	// uniquely identify the broker.
	AttributeMessagingDestinationPublishName = "messaging.destination_publish.name"
)

// Attributes for RabbitMQ
const (
	// RabbitMQ message routing key.
	//
	// Type: string
	// Requirement Level: Conditionally Required - If not empty.
	// Stability: experimental
	// Examples: 'myKey'
	AttributeMessagingRabbitmqDestinationRoutingKey = "messaging.rabbitmq.destination.routing_key"
)

// Attributes for Apache Kafka
const (
	// Name of the Kafka Consumer Group that is handling the message. Only applies to
	// consumers, not producers.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-group'
	AttributeMessagingKafkaConsumerGroup = "messaging.kafka.consumer.group"
	// Partition the message is sent to.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 2
	AttributeMessagingKafkaDestinationPartition = "messaging.kafka.destination.partition"
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
	// Requirement Level: Conditionally Required - If value is `true`. When missing,
	// the value is assumed to be `false`.
	// Stability: experimental
	AttributeMessagingKafkaMessageTombstone = "messaging.kafka.message.tombstone"
)

// Attributes for Apache RocketMQ
const (
	// Name of the RocketMQ producer/consumer group that is handling the message. The
	// client type is identified by the SpanKind.
	//
	// Type: string
	// Requirement Level: Required
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
	// Requirement Level: Conditionally Required - If the message type is delay and
	// delivery timestamp is not specified.
	// Stability: experimental
	// Examples: 3
	AttributeMessagingRocketmqMessageDelayTimeLevel = "messaging.rocketmq.message.delay_time_level"
	// The timestamp in milliseconds that the delay message is expected to be
	// delivered to consumer.
	//
	// Type: int
	// Requirement Level: Conditionally Required - If the message type is delay and
	// delay time level is not specified.
	// Stability: experimental
	// Examples: 1665987217045
	AttributeMessagingRocketmqMessageDeliveryTimestamp = "messaging.rocketmq.message.delivery_timestamp"
	// It is essential for FIFO message. Messages that belong to the same message
	// group are always processed one by one within the same consumer group.
	//
	// Type: string
	// Requirement Level: Conditionally Required - If the message type is FIFO.
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
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'myNamespace'
	AttributeMessagingRocketmqNamespace = "messaging.rocketmq.namespace"
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

// Attributes describing URL.
const (
	// The URI fragment component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'SemConv'
	AttributeURLFragment = "url.fragment"
	// Absolute URL describing a network resource according to RFC3986
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv', '//localhost'
	// Note: For network calls, URL usually has
	// scheme://host[:port][path][?query][#fragment] format, where the fragment is not
	// transmitted over HTTP, but if it is known, it should be included nevertheless.
	// url.full MUST NOT contain credentials passed via URL in form of
	// https://username:password@www.example.com/. In such case username and password
	// should be redacted and attribute's value should be
	// https://REDACTED:REDACTED@www.example.com/.
	// url.full SHOULD capture the absolute URL when it is available (or can be
	// reconstructed) and SHOULD NOT be validated or modified except for sanitizing
	// purposes.
	AttributeURLFull = "url.full"
	// The URI path component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/search'
	// Note: When missing, the value is assumed to be /
	AttributeURLPath = "url.path"
	// The URI query component
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'q=OpenTelemetry'
	// Note: Sensitive content provided in query string SHOULD be scrubbed when
	// instrumentations can identify it.
	AttributeURLQuery = "url.query"
	// The URI scheme component identifying the used protocol.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'https', 'ftp', 'telnet'
	AttributeURLScheme = "url.scheme"
)

// Describes user-agent attributes.
const (
	// Value of the HTTP User-Agent header sent by the client.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'CERN-LineMode/2.15 libwww/2.17b3'
	AttributeUserAgentOriginal = "user_agent.original"
)

func GetTraceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeClientAddress,
		AttributeClientPort,
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
		AttributeDestinationAddress,
		AttributeDestinationPort,
		AttributeErrorType,
		AttributeFaaSInvokedName,
		AttributeFaaSInvokedProvider,
		AttributeFaaSInvokedRegion,
		AttributeFaaSTrigger,
		AttributeEventDomain,
		AttributeEventName,
		AttributeLogRecordUID,
		AttributeLogIostream,
		AttributeLogFileName,
		AttributeLogFileNameResolved,
		AttributeLogFilePath,
		AttributeLogFilePathResolved,
		AttributePoolName,
		AttributeState,
		AttributeJvmBufferPoolName,
		AttributeJvmMemoryPoolName,
		AttributeJvmMemoryType,
		AttributeSystemDevice,
		AttributeSystemCPULogicalNumber,
		AttributeSystemCPUState,
		AttributeSystemMemoryState,
		AttributeSystemPagingDirection,
		AttributeSystemPagingState,
		AttributeSystemPagingType,
		AttributeSystemDiskDirection,
		AttributeSystemFilesystemMode,
		AttributeSystemFilesystemMountpoint,
		AttributeSystemFilesystemState,
		AttributeSystemFilesystemType,
		AttributeSystemNetworkDirection,
		AttributeSystemNetworkState,
		AttributeSystemProcessesStatus,
		AttributeNetworkLocalAddress,
		AttributeNetworkLocalPort,
		AttributeNetworkPeerAddress,
		AttributeNetworkPeerPort,
		AttributeNetworkProtocolName,
		AttributeNetworkProtocolVersion,
		AttributeNetworkTransport,
		AttributeNetworkType,
		AttributeNetworkCarrierIcc,
		AttributeNetworkCarrierMcc,
		AttributeNetworkCarrierMnc,
		AttributeNetworkCarrierName,
		AttributeNetworkConnectionSubtype,
		AttributeNetworkConnectionType,
		AttributeHTTPMethod,
		AttributeHTTPRequestContentLength,
		AttributeHTTPResponseContentLength,
		AttributeHTTPScheme,
		AttributeHTTPStatusCode,
		AttributeHTTPTarget,
		AttributeHTTPURL,
		AttributeHTTPRequestBodySize,
		AttributeHTTPRequestMethod,
		AttributeHTTPRequestMethodOriginal,
		AttributeHTTPResendCount,
		AttributeHTTPResponseBodySize,
		AttributeHTTPResponseStatusCode,
		AttributeHTTPRoute,
		AttributeServerAddress,
		AttributeServerPort,
		AttributeSessionID,
		AttributeSourceAddress,
		AttributeSourcePort,
		AttributeMessagingMessageBodySize,
		AttributeMessagingMessageConversationID,
		AttributeMessagingMessageEnvelopeSize,
		AttributeMessagingMessageID,
		AttributeMessagingDestinationAnonymous,
		AttributeMessagingDestinationName,
		AttributeMessagingDestinationTemplate,
		AttributeMessagingDestinationTemporary,
		AttributeMessagingDestinationPublishAnonymous,
		AttributeMessagingDestinationPublishName,
		AttributeMessagingRabbitmqDestinationRoutingKey,
		AttributeMessagingKafkaConsumerGroup,
		AttributeMessagingKafkaDestinationPartition,
		AttributeMessagingKafkaMessageKey,
		AttributeMessagingKafkaMessageOffset,
		AttributeMessagingKafkaMessageTombstone,
		AttributeMessagingRocketmqClientGroup,
		AttributeMessagingRocketmqConsumptionModel,
		AttributeMessagingRocketmqMessageDelayTimeLevel,
		AttributeMessagingRocketmqMessageDeliveryTimestamp,
		AttributeMessagingRocketmqMessageGroup,
		AttributeMessagingRocketmqMessageKeys,
		AttributeMessagingRocketmqMessageTag,
		AttributeMessagingRocketmqMessageType,
		AttributeMessagingRocketmqNamespace,
		AttributeURLFragment,
		AttributeURLFull,
		AttributeURLPath,
		AttributeURLQuery,
		AttributeURLScheme,
		AttributeUserAgentOriginal,
	}
}
