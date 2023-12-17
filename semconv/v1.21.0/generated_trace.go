// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// The shared attributes used to report a single exception associated with a
// span or log.
const (
	// The exception message.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Division by zero', "Can't convert 'int' object to str implicitly"
	AttributeExceptionMessage = "exception.message"
	// A stacktrace as a string in the natural representation for the language
	// runtime. The representation is to be determined and documented by each language
	// SIG.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
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
	// Stability: experimental
	// Examples: 'java.net.ConnectException', 'OSError'
	AttributeExceptionType = "exception.type"
)

// Span attributes used by AWS Lambda (in addition to general `faas`
// attributes).
const (
	// The full invoked ARN as provided on the Context passed to the function (Lambda-
	// Runtime-Invoked-Function-ARN header on the /runtime/invocation/next
	// applicable).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:lambda:us-east-1:123456:function:myfunction:myalias'
	// Note: This may be different from cloud.resource_id if an alias is involved.
	AttributeAWSLambdaInvokedARN = "aws.lambda.invoked_arn"
)

// Attributes for CloudEvents. CloudEvents is a specification on how to define
// event data in a standard way. These attributes can be attached to spans when
// performing operations with CloudEvents, regardless of the protocol being
// used.
const (
	// The event_id uniquely identifies the event.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: '123e4567-e89b-12d3-a456-426614174000', '0001'
	AttributeCloudeventsEventID = "cloudevents.event_id"
	// The source identifies the context in which an event happened.
	//
	// Type: string
	// Requirement Level: Required
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

// Semantic conventions for the OpenTracing Shim
const (
	// Parent-child Reference type
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: The causal relationship between a child Span and a parent Span.
	AttributeOpentracingRefType = "opentracing.ref_type"
)

const (
	// The parent Span depends on the child Span in some capacity
	AttributeOpentracingRefTypeChildOf = "child_of"
	// The parent Span does not depend in any way on the result of the child Span
	AttributeOpentracingRefTypeFollowsFrom = "follows_from"
)

// The attributes used to perform database client calls.
const (
	// The connection string used to connect to the database. It is recommended to
	// remove embedded credentials.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Server=(localdb)\\v11.0;Integrated Security=true;'
	AttributeDBConnectionString = "db.connection_string"
	// The fully-qualified class name of the Java Database Connectivity (JDBC) driver
	// used to connect.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'org.postgresql.Driver',
	// 'com.microsoft.sqlserver.jdbc.SQLServerDriver'
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"
	// This attribute is used to report the name of the database being accessed. For
	// commands that switch the database, this should be set to the target database
	// (even if the command fails).
	//
	// Type: string
	// Requirement Level: Conditionally Required - If applicable.
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
	// Requirement Level: Conditionally Required - If `db.statement` is not
	// applicable.
	// Stability: experimental
	// Examples: 'findAndModify', 'HMSET', 'SELECT'
	// Note: When setting this to an SQL keyword, it is not recommended to attempt any
	// client-side parsing of db.statement just to get this property, but it should be
	// set if the operation name is provided by the library being instrumented. If the
	// SQL statement has an ambiguous operation, or performs more than one operation,
	// this value may be omitted.
	AttributeDBOperation = "db.operation"
	// The database statement being executed.
	//
	// Type: string
	// Requirement Level: Recommended - Should be collected by default only if there
	// is sanitization that excludes sensitive information.
	// Stability: experimental
	// Examples: 'SELECT * FROM wuser_table', 'SET mykey "WuValue"'
	AttributeDBStatement = "db.statement"
	// An identifier for the database management system (DBMS) product being used. See
	// below for a list of well-known identifiers.
	//
	// Type: Enum
	// Requirement Level: Required
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

// Connection-level attributes for Microsoft SQL Server
const (
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
)

// Call-level attributes for Cassandra
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
	// The name of the primary table that the operation is acting upon, including the
	// keyspace name (if applicable).
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'mytable'
	// Note: This mirrors the db.sql.table attribute but references cassandra rather
	// than sql. It is not recommended to attempt any client-side parsing of
	// db.statement just to get this property, but it should be set if it is provided
	// by the library being instrumented. If the operation is acting upon an anonymous
	// table, or more than one table, this value MUST NOT be set.
	AttributeDBCassandraTable = "db.cassandra.table"
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

// Call-level attributes for Redis
const (
	// The index of the database being accessed as used in the SELECT command,
	// provided as an integer. To be used instead of the generic db.name attribute.
	//
	// Type: int
	// Requirement Level: Conditionally Required - If other than the default database
	// (`0`).
	// Stability: experimental
	// Examples: 0, 1, 15
	AttributeDBRedisDBIndex = "db.redis.database_index"
)

// Call-level attributes for MongoDB
const (
	// The collection being accessed within the database stated in db.name.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'customers', 'products'
	AttributeDBMongoDBCollection = "db.mongodb.collection"
)

// Call-level attributes for SQL databases
const (
	// The name of the primary table that the operation is acting upon, including the
	// database name (if applicable).
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: experimental
	// Examples: 'public.users', 'customers'
	// Note: It is not recommended to attempt any client-side parsing of db.statement
	// just to get this property, but it should be set if it is provided by the
	// library being instrumented. If the operation is acting upon an anonymous table,
	// or more than one table, this value MUST NOT be set.
	AttributeDBSQLTable = "db.sql.table"
)

// Call-level attributes for Cosmos DB.
const (
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
	// Requirement Level: Conditionally Required - if not `direct` (or pick gw as
	// default)
	// Stability: experimental
	AttributeDBCosmosDBConnectionMode = "db.cosmosdb.connection_mode"
	// Cosmos DB container name.
	//
	// Type: string
	// Requirement Level: Conditionally Required - if available
	// Stability: experimental
	// Examples: 'anystring'
	AttributeDBCosmosDBContainer = "db.cosmosdb.container"
	// CosmosDB Operation Type.
	//
	// Type: Enum
	// Requirement Level: Conditionally Required - when performing one of the
	// operations in this list
	// Stability: experimental
	AttributeDBCosmosDBOperationType = "db.cosmosdb.operation_type"
	// RU consumed for that operation
	//
	// Type: double
	// Requirement Level: Conditionally Required - when available
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
	// Requirement Level: Conditionally Required - if response was received
	// Stability: experimental
	// Examples: 200, 201
	AttributeDBCosmosDBStatusCode = "db.cosmosdb.status_code"
	// Cosmos DB sub status code.
	//
	// Type: int
	// Requirement Level: Conditionally Required - when response was received and
	// contained sub-code.
	// Stability: experimental
	// Examples: 1000, 1002
	AttributeDBCosmosDBSubStatusCode = "db.cosmosdb.sub_status_code"
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

// Span attributes used by non-OTLP exporters to represent OpenTelemetry Span's
// concepts.
const (
	// Name of the code, either &quot;OK&quot; or &quot;ERROR&quot;. MUST NOT be set
	// if the status code is UNSET.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeOTelStatusCode = "otel.status_code"
	// Description of the Status if it has a value, otherwise not set.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'resource not found'
	AttributeOTelStatusDescription = "otel.status_description"
)

const (
	// The operation has been validated by an Application developer or Operator to have completed successfully
	AttributeOTelStatusCodeOk = "OK"
	// The operation contains an error
	AttributeOTelStatusCodeError = "ERROR"
)

// This semantic convention describes an instance of a function that runs
// without provisioning or managing of servers (also known as serverless
// functions or Function as a Service (FaaS)) with spans.
const (
	// The invocation ID of the current function invocation.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'af9d5aa4-a685-4c5f-a22b-444f80b3cc28'
	AttributeFaaSInvocationID = "faas.invocation_id"
	// Type of the trigger which caused this function invocation.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: For the server/consumer span on the incoming side,
	// faas.trigger MUST be set.Clients invoking FaaS instances usually cannot set
	// faas.trigger,
	// since they would typically need to look in the payload to determine
	// the event type. If clients set it, it should be the same as the
	// trigger that corresponding incoming would have (i.e., this has
	// nothing to do with the underlying transport used to make the API
	// call to invoke the lambda, which is often HTTP).
	AttributeFaaSTrigger = "faas.trigger"
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

// Semantic Convention for FaaS triggered as a response to some data source
// operation such as a database or filesystem read/write.
const (
	// The name of the source on which the triggering operation was performed. For
	// example, in Cloud Storage or S3 corresponds to the bucket name, and in Cosmos
	// DB to the database name.
	//
	// Type: string
	// Requirement Level: Required
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
	// Requirement Level: Required
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
)

const (
	// When a new object is created
	AttributeFaaSDocumentOperationInsert = "insert"
	// When an object is modified
	AttributeFaaSDocumentOperationEdit = "edit"
	// When an object is deleted
	AttributeFaaSDocumentOperationDelete = "delete"
)

// Semantic Convention for FaaS scheduled to be executed regularly.
const (
	// A string containing the schedule period as Cron Expression.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0/5 * * * ? *'
	AttributeFaaSCron = "faas.cron"
	// A string containing the function invocation time in the ISO 8601 format
	// expressed in UTC.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2020-01-23T13:47:06Z'
	AttributeFaaSTime = "faas.time"
)

// Contains additional attributes for incoming FaaS spans.
const (
	// A boolean that is true if the serverless function is executed for the first
	// time (aka cold-start).
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeFaaSColdstart = "faas.coldstart"
)

// Contains additional attributes for outgoing FaaS spans.
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

// Operations that access some remote service.
const (
	// The service.name of the remote service. SHOULD be equal to the actual
	// service.name resource attribute of the remote service if any.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'AuthTokenCache'
	AttributePeerService = "peer.service"
)

// These attributes may be used for any operation with an authenticated and/or
// authorized enduser.
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
)

// Semantic Convention for HTTP Client
const (
	// The ordinal number of request resending attempt (for any reason, including
	// redirects).
	//
	// Type: int
	// Requirement Level: Recommended - if and only if request was retried.
	// Stability: experimental
	// Examples: 3
	// Note: The resend count SHOULD be updated each time an HTTP request gets resent
	// by the client, regardless of what was the cause of the resending (e.g.
	// redirection, authorization failure, 503 Server Unavailable, network issues, or
	// any other).
	AttributeHTTPResendCount = "http.resend_count"
)

// The `aws` conventions apply to operations using the AWS SDK. They map
// request or response parameters in AWS SDK API calls to attributes on a Span.
// The conventions have been collected over time based on feedback from AWS
// users of tracing and will continue to evolve as new interesting conventions
// are found.
// Some descriptions are also provided for populating general OpenTelemetry
// semantic conventions based on these APIs.
const (
	// The AWS request ID as returned in the response headers x-amz-request-id or
	// x-amz-requestid.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '79b9da39-b7ae-508a-a6bc-864b2829c622', 'C9ER4AJX75574TDJ'
	AttributeAWSRequestID = "aws.request_id"
)

// Attributes that exist for multiple DynamoDB request types.
const (
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
	// The value of the Select request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ALL_ATTRIBUTES', 'COUNT'
	AttributeAWSDynamoDBSelect = "aws.dynamodb.select"
	// The keys in the RequestItems object field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Users', 'Cats'
	AttributeAWSDynamoDBTableNames = "aws.dynamodb.table_names"
)

// DynamoDB.CreateTable
const (
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
)

// DynamoDB.ListTables
const (
	// The value of the ExclusiveStartTableName request parameter.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Users', 'CatsTable'
	AttributeAWSDynamoDBExclusiveStartTable = "aws.dynamodb.exclusive_start_table"
	// The the number of items in the TableNames response parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 20
	AttributeAWSDynamoDBTableCount = "aws.dynamodb.table_count"
)

// DynamoDB.Query
const (
	// The value of the ScanIndexForward request parameter.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	AttributeAWSDynamoDBScanForward = "aws.dynamodb.scan_forward"
)

// DynamoDB.Scan
const (
	// The value of the Count response parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 10
	AttributeAWSDynamoDBCount = "aws.dynamodb.count"
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
	// The value of the TotalSegments request parameter.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 100
	AttributeAWSDynamoDBTotalSegments = "aws.dynamodb.total_segments"
)

// DynamoDB.UpdateTable
const (
	// The JSON-serialized value of each item in the AttributeDefinitions request
	// field.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '{ "AttributeName": "string", "AttributeType": "string" }'
	AttributeAWSDynamoDBAttributeDefinitions = "aws.dynamodb.attribute_definitions"
	// The JSON-serialized value of each item in the the GlobalSecondaryIndexUpdates
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
)

// Attributes that exist for S3 request types.
const (
	// The S3 bucket name the request refers to. Corresponds to the --bucket parameter
	// of the S3 API operations.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'some-bucket-name'
	// Note: The bucket attribute is applicable to all S3 operations that reference a
	// bucket, i.e. that require the bucket name as a mandatory parameter.
	// This applies to almost all S3 operations except list-buckets.
	AttributeAWSS3Bucket = "aws.s3.bucket"
	// The source object (in the form bucket/key) for the copy operation.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'someFile.yml'
	// Note: The copy_source attribute applies to S3 copy operations and corresponds
	// to the --copy-source parameter
	// of the copy-object operation within the S3 API.
	// This applies in particular to the following operations:<ul>
	// <li>copy-object</li>
	// <li>upload-part-copy</li>
	// </ul>
	AttributeAWSS3CopySource = "aws.s3.copy_source"
	// The delete request container that specifies the objects to be deleted.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Objects=[{Key=string,VersionID=string},{Key=string,VersionID=string}
	// ],Quiet=boolean'
	// Note: The delete attribute is only applicable to the delete-object operation.
	// The delete attribute corresponds to the --delete parameter of the
	// delete-objects operation within the S3 API.
	AttributeAWSS3Delete = "aws.s3.delete"
	// The S3 object key the request refers to. Corresponds to the --key parameter of
	// the S3 API operations.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'someFile.yml'
	// Note: The key attribute is applicable to all object-related S3 operations, i.e.
	// that require the object key as a mandatory parameter.
	// This applies in particular to the following operations:<ul>
	// <li>copy-object</li>
	// <li>delete-object</li>
	// <li>get-object</li>
	// <li>head-object</li>
	// <li>put-object</li>
	// <li>restore-object</li>
	// <li>select-object-content</li>
	// <li>abort-multipart-upload</li>
	// <li>complete-multipart-upload</li>
	// <li>create-multipart-upload</li>
	// <li>list-parts</li>
	// <li>upload-part</li>
	// <li>upload-part-copy</li>
	// </ul>
	AttributeAWSS3Key = "aws.s3.key"
	// The part number of the part being uploaded in a multipart-upload operation.
	// This is a positive integer between 1 and 10,000.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 3456
	// Note: The part_number attribute is only applicable to the upload-part
	// and upload-part-copy operations.
	// The part_number attribute corresponds to the --part-number parameter of the
	// upload-part operation within the S3 API.
	AttributeAWSS3PartNumber = "aws.s3.part_number"
	// Upload ID that identifies the multipart upload.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'dfRtDYWFbkRONycy.Yxwh66Yjlx.cph0gtNBtJ'
	// Note: The upload_id attribute applies to S3 multipart-upload operations and
	// corresponds to the --upload-id parameter
	// of the S3 API multipart operations.
	// This applies in particular to the following operations:<ul>
	// <li>abort-multipart-upload</li>
	// <li>complete-multipart-upload</li>
	// <li>list-parts</li>
	// <li>upload-part</li>
	// <li>upload-part-copy</li>
	// </ul>
	AttributeAWSS3UploadID = "aws.s3.upload_id"
)

// Semantic conventions to apply when instrumenting the GraphQL implementation.
// They map GraphQL operations to attributes on a Span.
const (
	// The GraphQL document being executed.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'query findBookByID { bookByID(id: ?) { name } }'
	// Note: The value may be sanitized to exclude sensitive information.
	AttributeGraphqlDocument = "graphql.document"
	// The name of the operation being executed.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'findBookByID'
	AttributeGraphqlOperationName = "graphql.operation.name"
	// The type of the operation being executed.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'query', 'mutation', 'subscription'
	AttributeGraphqlOperationType = "graphql.operation.type"
)

const (
	// GraphQL query
	AttributeGraphqlOperationTypeQuery = "query"
	// GraphQL mutation
	AttributeGraphqlOperationTypeMutation = "mutation"
	// GraphQL subscription
	AttributeGraphqlOperationTypeSubscription = "subscription"
)

// General attributes used in messaging systems.
const (
	// The number of messages sent, received, or processed in the scope of the
	// batching operation.
	//
	// Type: int
	// Requirement Level: Conditionally Required - If the span describes an operation
	// on a batch of messages.
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
	// Requirement Level: Recommended - If a client id is available
	// Stability: experimental
	// Examples: 'client-5', 'myhost@8742@s8083jm'
	AttributeMessagingClientID = "messaging.client_id"
	// A string identifying the kind of messaging operation as defined in the
	// Operation names section above.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Note: If a custom value is used, it MUST be of low cardinality.
	AttributeMessagingOperation = "messaging.operation"
	// A string identifying the messaging system.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'kafka', 'rabbitmq', 'rocketmq', 'activemq', 'AmazonSQS'
	AttributeMessagingSystem = "messaging.system"
)

const (
	// publish
	AttributeMessagingOperationPublish = "publish"
	// receive
	AttributeMessagingOperationReceive = "receive"
	// process
	AttributeMessagingOperationProcess = "process"
)

// Semantic conventions for remote procedure calls.
const (
	// The name of the (logical) method being called, must be equal to the $method
	// part in the span name.
	//
	// Type: string
	// Requirement Level: Recommended
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
	// Requirement Level: Recommended
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
	// Requirement Level: Required
	// Stability: experimental
	AttributeRPCSystem = "rpc.system"
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

// Tech-specific attributes for gRPC.
const (
	// The numeric status code of the gRPC request.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	AttributeRPCGRPCStatusCode = "rpc.grpc.status_code"
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

// Tech-specific attributes for [JSON RPC](https://www.jsonrpc.org/).
const (
	// error.code property of response if it is an error response.
	//
	// Type: int
	// Requirement Level: Conditionally Required - If response is not successful.
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
	// does not specify this, the value can be omitted.
	//
	// Type: string
	// Requirement Level: Conditionally Required - If other than the default version
	// (`1.0`)
	// Stability: experimental
	// Examples: '2.0', '1.0'
	AttributeRPCJsonrpcVersion = "rpc.jsonrpc.version"
)

// Tech-specific attributes for Connect RPC.
const (
	// The error codes of the Connect request. Error codes are always string values.
	//
	// Type: Enum
	// Requirement Level: Conditionally Required - If response is not successful and
	// if error code available.
	// Stability: experimental
	AttributeRPCConnectRPCErrorCode = "rpc.connect_rpc.error_code"
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

func GetTraceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeExceptionMessage,
		AttributeExceptionStacktrace,
		AttributeExceptionType,
		AttributeAWSLambdaInvokedARN,
		AttributeCloudeventsEventID,
		AttributeCloudeventsEventSource,
		AttributeCloudeventsEventSpecVersion,
		AttributeCloudeventsEventSubject,
		AttributeCloudeventsEventType,
		AttributeOpentracingRefType,
		AttributeDBConnectionString,
		AttributeDBJDBCDriverClassname,
		AttributeDBName,
		AttributeDBOperation,
		AttributeDBStatement,
		AttributeDBSystem,
		AttributeDBUser,
		AttributeDBMSSQLInstanceName,
		AttributeDBCassandraConsistencyLevel,
		AttributeDBCassandraCoordinatorDC,
		AttributeDBCassandraCoordinatorID,
		AttributeDBCassandraIdempotence,
		AttributeDBCassandraPageSize,
		AttributeDBCassandraSpeculativeExecutionCount,
		AttributeDBCassandraTable,
		AttributeDBRedisDBIndex,
		AttributeDBMongoDBCollection,
		AttributeDBSQLTable,
		AttributeDBCosmosDBClientID,
		AttributeDBCosmosDBConnectionMode,
		AttributeDBCosmosDBContainer,
		AttributeDBCosmosDBOperationType,
		AttributeDBCosmosDBRequestCharge,
		AttributeDBCosmosDBRequestContentLength,
		AttributeDBCosmosDBStatusCode,
		AttributeDBCosmosDBSubStatusCode,
		AttributeOTelStatusCode,
		AttributeOTelStatusDescription,
		AttributeFaaSInvocationID,
		AttributeFaaSTrigger,
		AttributeFaaSDocumentCollection,
		AttributeFaaSDocumentName,
		AttributeFaaSDocumentOperation,
		AttributeFaaSDocumentTime,
		AttributeFaaSCron,
		AttributeFaaSTime,
		AttributeFaaSColdstart,
		AttributeFaaSInvokedName,
		AttributeFaaSInvokedProvider,
		AttributeFaaSInvokedRegion,
		AttributePeerService,
		AttributeEnduserID,
		AttributeEnduserRole,
		AttributeEnduserScope,
		AttributeThreadID,
		AttributeThreadName,
		AttributeCodeColumn,
		AttributeCodeFilepath,
		AttributeCodeFunction,
		AttributeCodeLineNumber,
		AttributeCodeNamespace,
		AttributeHTTPResendCount,
		AttributeAWSRequestID,
		AttributeAWSDynamoDBAttributesToGet,
		AttributeAWSDynamoDBConsistentRead,
		AttributeAWSDynamoDBConsumedCapacity,
		AttributeAWSDynamoDBIndexName,
		AttributeAWSDynamoDBItemCollectionMetrics,
		AttributeAWSDynamoDBLimit,
		AttributeAWSDynamoDBProjection,
		AttributeAWSDynamoDBProvisionedReadCapacity,
		AttributeAWSDynamoDBProvisionedWriteCapacity,
		AttributeAWSDynamoDBSelect,
		AttributeAWSDynamoDBTableNames,
		AttributeAWSDynamoDBGlobalSecondaryIndexes,
		AttributeAWSDynamoDBLocalSecondaryIndexes,
		AttributeAWSDynamoDBExclusiveStartTable,
		AttributeAWSDynamoDBTableCount,
		AttributeAWSDynamoDBScanForward,
		AttributeAWSDynamoDBCount,
		AttributeAWSDynamoDBScannedCount,
		AttributeAWSDynamoDBSegment,
		AttributeAWSDynamoDBTotalSegments,
		AttributeAWSDynamoDBAttributeDefinitions,
		AttributeAWSDynamoDBGlobalSecondaryIndexUpdates,
		AttributeAWSS3Bucket,
		AttributeAWSS3CopySource,
		AttributeAWSS3Delete,
		AttributeAWSS3Key,
		AttributeAWSS3PartNumber,
		AttributeAWSS3UploadID,
		AttributeGraphqlDocument,
		AttributeGraphqlOperationName,
		AttributeGraphqlOperationType,
		AttributeMessagingBatchMessageCount,
		AttributeMessagingClientID,
		AttributeMessagingOperation,
		AttributeMessagingSystem,
		AttributeRPCMethod,
		AttributeRPCService,
		AttributeRPCSystem,
		AttributeRPCGRPCStatusCode,
		AttributeRPCJsonrpcErrorCode,
		AttributeRPCJsonrpcErrorMessage,
		AttributeRPCJsonrpcRequestID,
		AttributeRPCJsonrpcVersion,
		AttributeRPCConnectRPCErrorCode,
	}
}
