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
	// Client address - unix domain socket name, IPv4 or IPv6 address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/tmp/my.sock', '10.1.2.80'
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.address SHOULD represent client address behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeClientAddress = "client.address"
	// Client port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 65123
	// Note: When observed from the server side, and when communicating through an
	// intermediary, client.port SHOULD represent client port behind any
	// intermediaries (e.g. proxies) if it's available.
	AttributeClientPort = "client.port"
	// Immediate client peer address - unix domain socket name, IPv4 or IPv6 address.
	//
	// Type: string
	// Requirement Level: Recommended - If different than `client.address`.
	// Stability: experimental
	// Examples: '/tmp/my.sock', '127.0.0.1'
	AttributeClientSocketAddress = "client.socket.address"
	// Immediate client peer port number
	//
	// Type: int
	// Requirement Level: Recommended - If different than `client.port`.
	// Stability: experimental
	// Examples: 35555
	AttributeClientSocketPort = "client.socket.port"
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
	// Deprecated, use server.socket.address.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '/var/my.sock'
	AttributeNetSockHostAddr = "net.sock.host.addr"
	// Deprecated, use server.socket.port.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 8080
	AttributeNetSockHostPort = "net.sock.host.port"
	// Deprecated, use server.socket.address on client spans and client.socket.address
	// on server spans.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '192.168.0.1'
	AttributeNetSockPeerAddr = "net.sock.peer.addr"
	// Deprecated, use server.socket.domain on client spans.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '/var/my.sock'
	AttributeNetSockPeerName = "net.sock.peer.name"
	// Deprecated, use server.socket.port on client spans and client.socket.port on
	// server spans.
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
	// Peer address, for example IP address or UNIX socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '10.5.3.2'
	AttributeDestinationAddress = "destination.address"
	// The domain name of the destination system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'foo.example.com'
	// Note: This value may be a host name, a fully qualified domain name, or another
	// host naming format.
	AttributeDestinationDomain = "destination.domain"
	// Peer port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeDestinationPort = "destination.port"
)

// Describes HTTP attributes.
const (
	// HTTP request method.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'GET', 'POST', 'HEAD'
	// Note: HTTP request method value SHOULD be &quot;known&quot; to the
	// instrumentation.
	// By default, this convention defines &quot;known&quot; methods as the ones
	// listed in RFC9110
	// and the PATCH method defined in RFC5789.If the HTTP request method is not known
	// to instrumentation, it MUST set the http.request.method attribute to _OTHER
	// and, except if reporting a metric, MUST
	// set the exact method received in the request line as value of the
	// http.request.method_original attribute.If the HTTP instrumentation could end up
	// converting valid HTTP request methods to _OTHER, then it MUST provide a way to
	// override
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
	// HTTP response status code.
	//
	// Type: int
	// Requirement Level: Conditionally Required - If and only if one was
	// received/sent.
	// Stability: experimental
	// Examples: 200
	AttributeHTTPResponseStatusCode = "http.response.status_code"
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

// HTTP Server attributes
const (
	// The matched route (path template in the format used by the respective server
	// framework). See note below
	//
	// Type: string
	// Requirement Level: Conditionally Required - If and only if it's available
	// Stability: experimental
	// Examples: '/users/:userID?', '{controller}/{action}/{id?}'
	// Note: MUST NOT be populated when this is not supported by the HTTP server
	// framework as the route attribute should have low-cardinality and the URI path
	// can NOT substitute it.
	// SHOULD include the application root if there is one.
	AttributeHTTPRoute = "http.route"
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

// Describes JVM memory metric attributes.
const (
	// Name of the memory pool.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'G1 Old Gen', 'G1 Eden space', 'G1 Survivor Space'
	// Note: Pool names are generally obtained via MemoryPoolMXBean#getName().
	AttributePool = "pool"
	// The type of memory.
	//
	// Type: Enum
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'heap', 'non_heap'
	AttributeType = "type"
)

const (
	// Heap memory
	AttributeTypeHeap = "heap"
	// Non-heap memory
	AttributeTypeNonHeap = "non_heap"
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
	// Logical server hostname, matches server FQDN if available, and IP or socket
	// address if FQDN is not known.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'example.com'
	AttributeServerAddress = "server.address"
	// Logical server port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 80, 8080, 443
	AttributeServerPort = "server.port"
	// Physical server IP address or Unix socket address. If set from the client,
	// should simply use the socket's peer address, and not attempt to find any actual
	// server IP (i.e., if set from client, this may represent some proxy server
	// instead of the logical server).
	//
	// Type: string
	// Requirement Level: Recommended - If different than `server.address`.
	// Stability: experimental
	// Examples: '10.5.3.2'
	AttributeServerSocketAddress = "server.socket.address"
	// The domain name of an immediate peer.
	//
	// Type: string
	// Requirement Level: Recommended - If different than `server.address`.
	// Stability: experimental
	// Examples: 'proxy.example.com'
	// Note: Typically observed from the client side, and represents a proxy or other
	// intermediary domain name.
	AttributeServerSocketDomain = "server.socket.domain"
	// Physical server port.
	//
	// Type: int
	// Requirement Level: Recommended - If different than `server.port`.
	// Stability: experimental
	// Examples: 16456
	AttributeServerSocketPort = "server.socket.port"
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
	// Source address, for example IP address or Unix socket name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '10.5.3.2'
	AttributeSourceAddress = "source.address"
	// The domain name of the source system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'foo.example.com'
	// Note: This value may be a host name, a fully qualified domain name, or another
	// host naming format.
	AttributeSourceDomain = "source.domain"
	// Source port number
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3389, 2888
	AttributeSourcePort = "source.port"
)

// These attributes may be used for any network related operation.
const (
	// OSI Application Layer or non-OSI equivalent. The value SHOULD be normalized to
	// lowercase.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'amqp', 'http', 'mqtt'
	AttributeNetworkProtocolName = "network.protocol.name"
	// Version of the application layer protocol used. See note below.
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
	// OSI Transport Layer or Inter-process Communication method. The value SHOULD be
	// normalized to lowercase.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'tcp', 'udp'
	AttributeNetworkTransport = "network.transport"
	// OSI Network Layer or non-OSI equivalent. The value SHOULD be normalized to
	// lowercase.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ipv4', 'ipv6'
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

// Semantic conventions for HTTP client and server Spans.
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
	// Original HTTP method sent by the client in the request line.
	//
	// Type: string
	// Requirement Level: Conditionally Required - If and only if it's different than
	// `http.request.method`.
	// Stability: experimental
	// Examples: 'GeT', 'ACL', 'foo'
	AttributeHTTPRequestMethodOriginal = "http.request.method_original"
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
)

// Semantic convention describing per-message attributes populated on messaging
// spans or links.
const (
	// The conversation ID identifying the conversation to which the message belongs,
	// represented as a string. Sometimes called &quot;Correlation ID&quot;.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'MyConversationID'
	AttributeMessagingMessageConversationID = "messaging.message.conversation_id"
	// A value used by the messaging system as an identifier for the message,
	// represented as a string.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '452a7c7c7c7048c2f887f61572b18fc2'
	AttributeMessagingMessageID = "messaging.message.id"
	// The compressed size of the message payload in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 2048
	AttributeMessagingMessagePayloadCompressedSizeBytes = "messaging.message.payload_compressed_size_bytes"
	// The (uncompressed) size of the message payload in bytes. Also use this
	// attribute if it is unknown whether the compressed or uncompressed payload size
	// is reported.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 2738
	AttributeMessagingMessagePayloadSizeBytes = "messaging.message.payload_size_bytes"
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
		AttributeClientSocketAddress,
		AttributeClientSocketPort,
		AttributeHTTPMethod,
		AttributeHTTPRequestContentLength,
		AttributeHTTPResponseContentLength,
		AttributeHTTPScheme,
		AttributeHTTPStatusCode,
		AttributeHTTPTarget,
		AttributeHTTPURL,
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
		AttributeDestinationDomain,
		AttributeDestinationPort,
		AttributeHTTPRequestMethod,
		AttributeHTTPResponseStatusCode,
		AttributeHTTPRoute,
		AttributeEventDomain,
		AttributeEventName,
		AttributeLogRecordUID,
		AttributeLogIostream,
		AttributeLogFileName,
		AttributeLogFileNameResolved,
		AttributeLogFilePath,
		AttributeLogFilePathResolved,
		AttributePool,
		AttributeType,
		AttributeServerAddress,
		AttributeServerPort,
		AttributeServerSocketAddress,
		AttributeServerSocketDomain,
		AttributeServerSocketPort,
		AttributeSourceAddress,
		AttributeSourceDomain,
		AttributeSourcePort,
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
		AttributeHTTPRequestBodySize,
		AttributeHTTPRequestMethodOriginal,
		AttributeHTTPResponseBodySize,
		AttributeMessagingMessageConversationID,
		AttributeMessagingMessageID,
		AttributeMessagingMessagePayloadCompressedSizeBytes,
		AttributeMessagingMessagePayloadSizeBytes,
		AttributeMessagingDestinationAnonymous,
		AttributeMessagingDestinationName,
		AttributeMessagingDestinationTemplate,
		AttributeMessagingDestinationTemporary,
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
