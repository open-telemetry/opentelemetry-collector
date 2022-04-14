// Copyright The OpenTelemetry Authors
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

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv // import "go.opentelemetry.io/collector/model/semconv/v1.5.0"

// Span attributes used by AWS Lambda (in addition to general `faas` attributes).
const (
	// The full invoked ARN as provided on the Context passed to the function (Lambda-
	// Runtime-Invoked-Function-ARN header on the /runtime/invocation/next
	// applicable).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:lambda:us-east-1:123456:function:myfunction:myalias'
	// Note: This may be different from faas.id if an alias is involved.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSLambdaInvokedARN = "aws.lambda.invoked_arn"
)

// This document defines the attributes used to perform database client calls.
const (
	// An identifier for the database management system (DBMS) product being used. See
	// below for a list of well-known identifiers.
	//
	// Type: Enum
	// Required: Always
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystem = "db.system"
	// The connection string used to connect to the database. It is recommended to
	// remove embedded credentials.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Server=(localdb)\\v11.0;Integrated Security=true;'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBConnectionString = "db.connection_string"
	// Username for accessing the database.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'readonly_user', 'reporting_user'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBUser = "db.user"
	// The fully-qualified class name of the Java Database Connectivity (JDBC) driver
	// used to connect.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'org.postgresql.Driver',
	// 'com.microsoft.sqlserver.jdbc.SQLServerDriver'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"
	// If no tech-specific attribute is defined, this attribute is used to report the
	// name of the database being accessed. For commands that switch the database,
	// this should be set to the target database (even if the command fails).
	//
	// Type: string
	// Required: Required, if applicable and no more-specific attribute is defined.
	// Stability: stable
	// Examples: 'customers', 'main'
	// Note: In some SQL databases, the database name to be used is called
	// &quot;schema name&quot;.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBName = "db.name"
	// The database statement being executed.
	//
	// Type: string
	// Required: Required if applicable and not explicitly disabled via
	// instrumentation configuration.
	// Stability: stable
	// Examples: 'SELECT * FROM wuser_table', 'SET mykey "WuValue"'
	// Note: The value may be sanitized to exclude sensitive information.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBStatement = "db.statement"
	// The name of the operation being executed, e.g. the MongoDB command name such as
	// findAndModify, or the SQL keyword.
	//
	// Type: string
	// Required: Required, if `db.statement` is not applicable.
	// Stability: stable
	// Examples: 'findAndModify', 'HMSET', 'SELECT'
	// Note: When setting this to an SQL keyword, it is not recommended to attempt any
	// client-side parsing of db.statement just to get this property, but it should be
	// set if the operation name is provided by the library being instrumented. If the
	// SQL statement has an ambiguous operation, or performs more than one operation,
	// this value may be omitted.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBOperation = "db.operation"
)

const (
	// Some other SQL database. Fallback only. See notes
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemOtherSQL = "other_sql"
	// Microsoft SQL Server
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMSSQL = "mssql"
	// MySQL
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMySQL = "mysql"
	// Oracle Database
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemOracle = "oracle"
	// IBM DB2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemDB2 = "db2"
	// PostgreSQL
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemPostgreSQL = "postgresql"
	// Amazon Redshift
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemRedshift = "redshift"
	// Apache Hive
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemHive = "hive"
	// Cloudscape
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCloudscape = "cloudscape"
	// HyperSQL DataBase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemHSQLDB = "hsqldb"
	// Progress Database
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemProgress = "progress"
	// SAP MaxDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMaxDB = "maxdb"
	// SAP HANA
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemHanaDB = "hanadb"
	// Ingres
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemIngres = "ingres"
	// FirstSQL
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemFirstSQL = "firstsql"
	// EnterpriseDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemEDB = "edb"
	// InterSystems Caché
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCache = "cache"
	// Adabas (Adaptable Database System)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemAdabas = "adabas"
	// Firebird
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemFirebird = "firebird"
	// Apache Derby
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemDerby = "derby"
	// FileMaker
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemFilemaker = "filemaker"
	// Informix
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemInformix = "informix"
	// InstantDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemInstantDB = "instantdb"
	// InterBase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemInterbase = "interbase"
	// MariaDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMariaDB = "mariadb"
	// Netezza
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemNetezza = "netezza"
	// Pervasive PSQL
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemPervasive = "pervasive"
	// PointBase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemPointbase = "pointbase"
	// SQLite
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemSqlite = "sqlite"
	// Sybase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemSybase = "sybase"
	// Teradata
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemTeradata = "teradata"
	// Vertica
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemVertica = "vertica"
	// H2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemH2 = "h2"
	// ColdFusion IMQ
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemColdfusion = "coldfusion"
	// Apache Cassandra
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCassandra = "cassandra"
	// Apache HBase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemHBase = "hbase"
	// MongoDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMongoDB = "mongodb"
	// Redis
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemRedis = "redis"
	// Couchbase
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCouchbase = "couchbase"
	// CouchDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCouchDB = "couchdb"
	// Microsoft Azure Cosmos DB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCosmosDB = "cosmosdb"
	// Amazon DynamoDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemDynamoDB = "dynamodb"
	// Neo4j
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemNeo4j = "neo4j"
	// Apache Geode
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemGeode = "geode"
	// Elasticsearch
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemElasticsearch = "elasticsearch"
	// Memcached
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemMemcached = "memcached"
	// CockroachDB
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSystemCockroachdb = "cockroachdb"
)

// Connection-level attributes for Microsoft SQL Server
const (
	// The Microsoft SQL Server instance name connecting to. This name is used to
	// determine the port of a named instance.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'MSSQLSERVER'
	// Note: If setting a db.mssql.instance_name, net.peer.port is no longer required
	// (but still recommended if non-standard).
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBMSSQLInstanceName = "db.mssql.instance_name"
)

// Call-level attributes for Cassandra
const (
	// The name of the keyspace being accessed. To be used instead of the generic
	// db.name attribute.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'mykeyspace'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraKeyspace = "db.cassandra.keyspace"
	// The fetch size used for paging, i.e. how many rows will be returned at once.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 5000
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraPageSize = "db.cassandra.page_size"
	// The consistency level of the query. Based on consistency values from CQL.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevel = "db.cassandra.consistency_level"
	// The name of the primary table that the operation is acting upon, including the
	// schema name (if applicable).
	//
	// Type: string
	// Required: Recommended if available.
	// Stability: stable
	// Examples: 'mytable'
	// Note: This mirrors the db.sql.table attribute but references cassandra rather
	// than sql. It is not recommended to attempt any client-side parsing of
	// db.statement just to get this property, but it should be set if it is provided
	// by the library being instrumented. If the operation is acting upon an anonymous
	// table, or more than one table, this value MUST NOT be set.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraTable = "db.cassandra.table"
	// Whether or not the query is idempotent.
	//
	// Type: boolean
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraIdempotence = "db.cassandra.idempotence"
	// The number of times a query was speculatively executed. Not set or 0 if the
	// query was not executed speculatively.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 0, 2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraSpeculativeExecutionCount = "db.cassandra.speculative_execution_count"
	// The ID of the coordinating node for a query.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'be13faa2-8574-4d71-926d-27f16cf8a7af'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraCoordinatorID = "db.cassandra.coordinator.id"
	// The data center of the coordinating node for a query.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'us-west-2'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraCoordinatorDC = "db.cassandra.coordinator.dc"
)

const (
	// all
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelAll = "all"
	// each_quorum
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelEachQuorum = "each_quorum"
	// quorum
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelQuorum = "quorum"
	// local_quorum
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelLocalQuorum = "local_quorum"
	// one
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelOne = "one"
	// two
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelTwo = "two"
	// three
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelThree = "three"
	// local_one
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelLocalOne = "local_one"
	// any
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelAny = "any"
	// serial
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelSerial = "serial"
	// local_serial
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBCassandraConsistencyLevelLocalSerial = "local_serial"
)

// Call-level attributes for Apache HBase
const (
	// The HBase namespace being accessed. To be used instead of the generic db.name
	// attribute.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'default'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBHBaseNamespace = "db.hbase.namespace"
)

// Call-level attributes for Redis
const (
	// The index of the database being accessed as used in the SELECT command,
	// provided as an integer. To be used instead of the generic db.name attribute.
	//
	// Type: int
	// Required: Required, if other than the default database (`0`).
	// Stability: stable
	// Examples: 0, 1, 15
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBRedisDBIndex = "db.redis.database_index"
)

// Call-level attributes for MongoDB
const (
	// The collection being accessed within the database stated in db.name.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'customers', 'products'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBMongoDBCollection = "db.mongodb.collection"
)

// Call-level attrbiutes for SQL databases
const (
	// The name of the primary table that the operation is acting upon, including the
	// schema name (if applicable).
	//
	// Type: string
	// Required: Recommended if available.
	// Stability: stable
	// Examples: 'public.users', 'customers'
	// Note: It is not recommended to attempt any client-side parsing of db.statement
	// just to get this property, but it should be set if it is provided by the
	// library being instrumented. If the operation is acting upon an anonymous table,
	// or more than one table, this value MUST NOT be set.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDBSQLTable = "db.sql.table"
)

// This document defines the attributes used to report a single exception associated with a span.
const (
	// The type of the exception (its fully-qualified class name, if applicable). The
	// dynamic type of the exception should be preferred over the static type in
	// languages that support it.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'java.net.ConnectException', 'OSError'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeExceptionType = "exception.type"
	// The exception message.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Division by zero', "Can't convert 'int' object to str implicitly"
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeExceptionMessage = "exception.message"
	// A stacktrace as a string in the natural representation for the language
	// runtime. The representation is to be determined and documented by each language
	// SIG.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Exception in thread "main" java.lang.RuntimeException: Test
	// exception\\n at '
	//  'com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at '
	//  'com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at '
	//  'com.example.GenerateTrace.main(GenerateTrace.java:5)'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeExceptionStacktrace = "exception.stacktrace"
	// SHOULD be set to true if the exception event is recorded at a point where it is
	// known that the exception is escaping the scope of the span.
	//
	// Type: boolean
	// Required: No
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
	// as done in the example above.It follows that an exception may still escape the
	// scope of the span
	// even if the exception.escaped attribute was not set or set to false,
	// since the event might have been recorded at a time where it was not
	// clear whether the exception will escape.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeExceptionEscaped = "exception.escaped"
)

// This semantic convention describes an instance of a function that runs without provisioning or managing of servers (also known as serverless functions or Function as a Service (FaaS)) with spans.
const (
	// Type of the trigger on which the function is executed.
	//
	// Type: Enum
	// Required: On FaaS instances, faas.trigger MUST be set on incoming invocations.
	// Clients invoking FaaS instances MUST set `faas.trigger` on outgoing
	// invocations, if it is known to the client. This is, for example, not the case,
	// when the transport layer is abstracted in a FaaS client framework without
	// access to its configuration.
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTrigger = "faas.trigger"
	// The execution ID of the current function execution.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'af9d5aa4-a685-4c5f-a22b-444f80b3cc28'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSExecution = "faas.execution"
)

const (
	// A response to some data source operation such as a database or filesystem read/write
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTriggerDatasource = "datasource"
	// To provide an answer to an inbound HTTP request
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTriggerHTTP = "http"
	// A function is set to be executed when messages are sent to a messaging system
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTriggerPubsub = "pubsub"
	// A function is scheduled to be executed regularly
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTriggerTimer = "timer"
	// If none of the others apply
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTriggerOther = "other"
)

// Semantic Convention for FaaS triggered as a response to some data source operation such as a database or filesystem read/write.
const (
	// The name of the source on which the triggering operation was performed. For
	// example, in Cloud Storage or S3 corresponds to the bucket name, and in Cosmos
	// DB to the database name.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'myBucketName', 'myDBName'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentCollection = "faas.document.collection"
	// Describes the type of the operation that was performed on the data.
	//
	// Type: Enum
	// Required: Always
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentOperation = "faas.document.operation"
	// A string containing the time when the data was accessed in the ISO 8601 format
	// expressed in UTC.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: '2020-01-23T13:47:06Z'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentTime = "faas.document.time"
	// The document name/table subjected to the operation. For example, in Cloud
	// Storage or S3 is the name of the file, and in Cosmos DB the table name.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'myFile.txt', 'myTableName'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentName = "faas.document.name"
)

const (
	// When a new object is created
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentOperationInsert = "insert"
	// When an object is modified
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentOperationEdit = "edit"
	// When an object is deleted
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSDocumentOperationDelete = "delete"
)

// Semantic Convention for FaaS scheduled to be executed regularly.
const (
	// A string containing the function invocation time in the ISO 8601 format
	// expressed in UTC.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: '2020-01-23T13:47:06Z'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSTime = "faas.time"
	// A string containing the schedule period as Cron Expression.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '0/5 * * * ? *'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSCron = "faas.cron"
)

// Contains additional attributes for incoming FaaS spans.
const (
	// A boolean that is true if the serverless function is executed for the first
	// time (aka cold-start).
	//
	// Type: boolean
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSColdstart = "faas.coldstart"
)

// Contains additional attributes for outgoing FaaS spans.
const (
	// The name of the invoked function.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'my-function'
	// Note: SHOULD be equal to the faas.name resource attribute of the invoked
	// function.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedName = "faas.invoked_name"
	// The cloud provider of the invoked function.
	//
	// Type: Enum
	// Required: Always
	// Stability: stable
	// Note: SHOULD be equal to the cloud.provider resource attribute of the invoked
	// function.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedProvider = "faas.invoked_provider"
	// The cloud region of the invoked function.
	//
	// Type: string
	// Required: For some cloud providers, like AWS or GCP, the region in which a
	// function is hosted is essential to uniquely identify the function and also part
	// of its endpoint. Since it's part of the endpoint being called, the region is
	// always known to clients. In these cases, `faas.invoked_region` MUST be set
	// accordingly. If the region is unknown to the client or not required for
	// identifying the invoked function, setting `faas.invoked_region` is optional.
	// Stability: stable
	// Examples: 'eu-central-1'
	// Note: SHOULD be equal to the cloud.region resource attribute of the invoked
	// function.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedRegion = "faas.invoked_region"
)

const (
	// Amazon Web Services
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedProviderAWS = "aws"
	// Microsoft Azure
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedProviderAzure = "azure"
	// Google Cloud Platform
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInvokedProviderGCP = "gcp"
)

// These attributes may be used for any network related operation.
const (
	// Transport protocol used. See note below.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransport = "net.transport"
	// Remote address of the peer (dotted decimal for IPv4 or RFC5952 for IPv6)
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '127.0.0.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetPeerIP = "net.peer.ip"
	// Remote port number.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 80, 8080, 443
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetPeerPort = "net.peer.port"
	// Remote hostname or similar, see note below.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'example.com'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetPeerName = "net.peer.name"
	// Like net.peer.ip but for the host IP. Useful in case of a multi-IP host.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '192.168.0.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetHostIP = "net.host.ip"
	// Like net.peer.port but for the host port.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 35555
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetHostPort = "net.host.port"
	// Local hostname or similar, see note below.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'localhost'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetHostName = "net.host.name"
)

const (
	// ip_tcp
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportTCP = "ip_tcp"
	// ip_udp
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportUDP = "ip_udp"
	// Another IP-based protocol
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportIP = "ip"
	// Unix Domain socket. See below
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportUnix = "unix"
	// Named or anonymous pipe. See note below
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportPipe = "pipe"
	// In-process communication
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportInProc = "inproc"
	// Something else (non IP-based)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeNetTransportOther = "other"
)

// Operations that access some remote service.
const (
	// The service.name of the remote service. SHOULD be equal to the actual
	// service.name resource attribute of the remote service if any.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'AuthTokenCache'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributePeerService = "peer.service"
)

// These attributes may be used for any operation with an authenticated and/or authorized enduser.
const (
	// Username or client_id extracted from the access token or Authorization header
	// in the inbound request from outside the system.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'username'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeEnduserID = "enduser.id"
	// Actual/assumed role the client is making the request under extracted from token
	// or application security context.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'admin'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeEnduserRole = "enduser.role"
	// Scopes or granted authorities the client currently possesses extracted from
	// token or application security context. The value would come from the scope
	// associated with an OAuth 2.0 Access Token or an attribute value in a SAML 2.0
	// Assertion.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'read:message, write:files'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeEnduserScope = "enduser.scope"
)

// These attributes may be used for any operation to store information about a thread that started a span.
const (
	// Current &quot;managed&quot; thread ID (as opposed to OS thread ID).
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 42
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeThreadID = "thread.id"
	// Current thread name.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'main'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeThreadName = "thread.name"
)

// These attributes allow to report this unit of code and therefore to provide more context about the span.
const (
	// The method or function name, or equivalent (usually rightmost part of the code
	// unit's name).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'serveRequest'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCodeFunction = "code.function"
	// The &quot;namespace&quot; within which code.function is defined. Usually the
	// qualified class or module name, such that code.namespace + some separator +
	// code.function form a unique identifier for the code unit.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'com.example.MyHTTPService'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCodeNamespace = "code.namespace"
	// The source code file name that identifies the code unit as uniquely as possible
	// (preferably an absolute file path).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '/usr/local/MyApplication/content_root/app/index.php'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCodeFilepath = "code.filepath"
	// The line number in code.filepath best representing the operation. It SHOULD
	// point within the code unit named in code.function.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 42
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCodeLineNumber = "code.lineno"
)

// This document defines semantic conventions for HTTP client and server Spans.
const (
	// HTTP request method.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'GET', 'POST', 'HEAD'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPMethod = "http.method"
	// Full HTTP request URL in the form scheme://host[:port]/path?query[#fragment].
	// Usually the fragment is not transmitted over HTTP, but if it is known, it
	// should be included nevertheless.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'https://www.foo.bar/search?q=OpenTelemetry#SemConv'
	// Note: http.url MUST NOT contain credentials passed via URL in form of
	// https://username:password@www.example.com/. In such case the attribute's value
	// should be https://www.example.com/.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPURL = "http.url"
	// The full request target as passed in a HTTP request line or equivalent.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '/path/12314/?q=ddds#123'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPTarget = "http.target"
	// The value of the HTTP host header. When the header is empty or not present,
	// this attribute should be the same.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'www.example.org'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPHost = "http.host"
	// The URI scheme identifying the used protocol.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'http', 'https'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPScheme = "http.scheme"
	// HTTP response status code.
	//
	// Type: int
	// Required: If and only if one was received/sent.
	// Stability: stable
	// Examples: 200
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPStatusCode = "http.status_code"
	// Kind of HTTP protocol used.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Note: If net.transport is not specified, it can be assumed to be IP.TCP except
	// if http.flavor is QUIC, in which case IP.UDP is assumed.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavor = "http.flavor"
	// Value of the HTTP User-Agent header sent by the client.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'CERN-LineMode/2.15 libwww/2.17b3'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPUserAgent = "http.user_agent"
	// The size of the request payload body in bytes. This is the number of bytes
	// transferred excluding headers and is often, but not always, present as the
	// Content-Length header. For requests using transport encoding, this should be
	// the compressed size.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 3495
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPRequestContentLength = "http.request_content_length"
	// The size of the uncompressed request payload body after transport decoding. Not
	// set if transport encoding not used.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 5493
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPRequestContentLengthUncompressed = "http.request_content_length_uncompressed"
	// The size of the response payload body in bytes. This is the number of bytes
	// transferred excluding headers and is often, but not always, present as the
	// Content-Length header. For requests using transport encoding, this should be
	// the compressed size.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 3495
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPResponseContentLength = "http.response_content_length"
	// The size of the uncompressed response payload body after transport decoding.
	// Not set if transport encoding not used.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 5493
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPResponseContentLengthUncompressed = "http.response_content_length_uncompressed"
)

const (
	// HTTP 1.0
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavorHTTP10 = "1.0"
	// HTTP 1.1
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavorHTTP11 = "1.1"
	// HTTP 2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavorHTTP20 = "2.0"
	// SPDY protocol
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavorSPDY = "SPDY"
	// QUIC protocol
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPFlavorQUIC = "QUIC"
)

// Semantic Convention for HTTP Server
const (
	// The primary server name of the matched virtual host. This should be obtained
	// via configuration. If no such configuration can be obtained, this attribute
	// MUST NOT be set ( net.host.name should be used instead).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'example.com'
	// Note: http.url is usually not readily available on the server side but would
	// have to be assembled in a cumbersome and sometimes lossy process from other
	// information (see e.g. open-telemetry/opentelemetry-python/pull/148). It is thus
	// preferred to supply the raw data that is available.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPServerName = "http.server_name"
	// The matched route (path template).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '/users/:userID?'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPRoute = "http.route"
	// The IP address of the original client behind all proxies, if known (e.g. from
	// X-Forwarded-For).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '83.164.160.102'
	// Note: This is not necessarily the same as net.peer.ip, which would identify the
	// network-level peer, which may be a proxy.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHTTPClientIP = "http.client_ip"
)

// Attributes that exist for multiple DynamoDB request types.
const (
	// The keys in the RequestItems object field.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: 'Users', 'Cats'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBTableNames = "aws.dynamodb.table_names"
	// The JSON-serialized value of each item in the ConsumedCapacity response field.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '{ "CapacityUnits": number, "GlobalSecondaryIndexes": { "string" : {
	// "CapacityUnits": number, "ReadCapacityUnits": number, "WriteCapacityUnits":
	// number } }, "LocalSecondaryIndexes": { "string" : { "CapacityUnits": number,
	// "ReadCapacityUnits": number, "WriteCapacityUnits": number } },
	// "ReadCapacityUnits": number, "Table": { "CapacityUnits": number,
	// "ReadCapacityUnits": number, "WriteCapacityUnits": number }, "TableName":
	// "string", "WriteCapacityUnits": number }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBConsumedCapacity = "aws.dynamodb.consumed_capacity"
	// The JSON-serialized value of the ItemCollectionMetrics response field.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '{ "string" : [ { "ItemCollectionKey": { "string" : { "B": blob,
	// "BOOL": boolean, "BS": [ blob ], "L": [ "AttributeValue" ], "M": { "string" :
	// "AttributeValue" }, "N": "string", "NS": [ "string" ], "NULL": boolean, "S":
	// "string", "SS": [ "string" ] } }, "SizeEstimateRangeGB": [ number ] } ] }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBItemCollectionMetrics = "aws.dynamodb.item_collection_metrics"
	// The value of the ProvisionedThroughput.ReadCapacityUnits request parameter.
	//
	// Type: double
	// Required: No
	// Stability: stable
	// Examples: 1.0, 2.0
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBProvisionedReadCapacity = "aws.dynamodb.provisioned_read_capacity"
	// The value of the ProvisionedThroughput.WriteCapacityUnits request parameter.
	//
	// Type: double
	// Required: No
	// Stability: stable
	// Examples: 1.0, 2.0
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBProvisionedWriteCapacity = "aws.dynamodb.provisioned_write_capacity"
	// The value of the ConsistentRead request parameter.
	//
	// Type: boolean
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBConsistentRead = "aws.dynamodb.consistent_read"
	// The value of the ProjectionExpression request parameter.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Title', 'Title, Price, Color', 'Title, Description, RelatedItems,
	// ProductReviews'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBProjection = "aws.dynamodb.projection"
	// The value of the Limit request parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 10
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBLimit = "aws.dynamodb.limit"
	// The value of the AttributesToGet request parameter.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: 'lives', 'id'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBAttributesToGet = "aws.dynamodb.attributes_to_get"
	// The value of the IndexName request parameter.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'name_to_group'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBIndexName = "aws.dynamodb.index_name"
	// The value of the Select request parameter.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'ALL_ATTRIBUTES', 'COUNT'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBSelect = "aws.dynamodb.select"
)

// DynamoDB.CreateTable
const (
	// The JSON-serialized value of each item of the GlobalSecondaryIndexes request
	// field
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '{ "IndexName": "string", "KeySchema": [ { "AttributeName": "string",
	// "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ],
	// "ProjectionType": "string" }, "ProvisionedThroughput": { "ReadCapacityUnits":
	// number, "WriteCapacityUnits": number } }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBGlobalSecondaryIndexes = "aws.dynamodb.global_secondary_indexes"
	// The JSON-serialized value of each item of the LocalSecondaryIndexes request
	// field.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '{ "IndexARN": "string", "IndexName": "string", "IndexSizeBytes":
	// number, "ItemCount": number, "KeySchema": [ { "AttributeName": "string",
	// "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ],
	// "ProjectionType": "string" } }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBLocalSecondaryIndexes = "aws.dynamodb.local_secondary_indexes"
)

// DynamoDB.ListTables
const (
	// The value of the ExclusiveStartTableName request parameter.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Users', 'CatsTable'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBExclusiveStartTable = "aws.dynamodb.exclusive_start_table"
	// The the number of items in the TableNames response parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 20
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBTableCount = "aws.dynamodb.table_count"
)

// DynamoDB.Query
const (
	// The value of the ScanIndexForward request parameter.
	//
	// Type: boolean
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBScanForward = "aws.dynamodb.scan_forward"
)

// DynamoDB.Scan
const (
	// The value of the Segment request parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 10
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBSegment = "aws.dynamodb.segment"
	// The value of the TotalSegments request parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 100
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBTotalSegments = "aws.dynamodb.total_segments"
	// The value of the Count response parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 10
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBCount = "aws.dynamodb.count"
	// The value of the ScannedCount response parameter.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 50
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBScannedCount = "aws.dynamodb.scanned_count"
)

// DynamoDB.UpdateTable
const (
	// The JSON-serialized value of each item in the AttributeDefinitions request
	// field.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '{ "AttributeName": "string", "AttributeType": "string" }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBAttributeDefinitions = "aws.dynamodb.attribute_definitions"
	// The JSON-serialized value of each item in the the GlobalSecondaryIndexUpdates
	// request field.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '{ "Create": { "IndexName": "string", "KeySchema": [ {
	// "AttributeName": "string", "KeyType": "string" } ], "Projection": {
	// "NonKeyAttributes": [ "string" ], "ProjectionType": "string" },
	// "ProvisionedThroughput": { "ReadCapacityUnits": number, "WriteCapacityUnits":
	// number } }'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSDynamoDBGlobalSecondaryIndexUpdates = "aws.dynamodb.global_secondary_index_updates"
)

// This document defines the attributes used in messaging systems.
const (
	// A string identifying the messaging system.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'kafka', 'rabbitmq', 'activemq', 'AmazonSQS'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingSystem = "messaging.system"
	// The message destination name. This might be equal to the span name but is
	// required nevertheless.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'MyQueue', 'MyTopic'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingDestination = "messaging.destination"
	// The kind of message destination
	//
	// Type: Enum
	// Required: Required only if the message destination is either a `queue` or
	// `topic`.
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingDestinationKind = "messaging.destination_kind"
	// A boolean that is true if the message destination is temporary.
	//
	// Type: boolean
	// Required: If missing, it is assumed to be false.
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingTempDestination = "messaging.temp_destination"
	// The name of the transport protocol.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'AMQP', 'MQTT'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingProtocol = "messaging.protocol"
	// The version of the transport protocol.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '0.9.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingProtocolVersion = "messaging.protocol_version"
	// Connection string.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'tibjmsnaming://localhost:7222',
	// 'https://queue.amazonaws.com/80398EXAMPLE/MyQueue'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingURL = "messaging.url"
	// A value used by the messaging system as an identifier for the message,
	// represented as a string.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '452a7c7c7c7048c2f887f61572b18fc2'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingMessageID = "messaging.message_id"
	// The conversation ID identifying the conversation to which the message belongs,
	// represented as a string. Sometimes called &quot;Correlation ID&quot;.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'MyConversationID'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingConversationID = "messaging.conversation_id"
	// The (uncompressed) size of the message payload in bytes. Also use this
	// attribute if it is unknown whether the compressed or uncompressed payload size
	// is reported.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 2738
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingMessagePayloadSizeBytes = "messaging.message_payload_size_bytes"
	// The compressed size of the message payload in bytes.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 2048
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingMessagePayloadCompressedSizeBytes = "messaging.message_payload_compressed_size_bytes"
)

const (
	// A message sent to a queue
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingDestinationKindQueue = "queue"
	// A message sent to a topic
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingDestinationKindTopic = "topic"
)

// Semantic convention for a consumer of messages received from a messaging system
const (
	// A string identifying the kind of message consumption as defined in the
	// Operation names section above. If the operation is &quot;send&quot;, this
	// attribute MUST NOT be set, since the operation can be inferred from the span
	// kind in that case.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingOperation = "messaging.operation"
)

const (
	// receive
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingOperationReceive = "receive"
	// process
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingOperationProcess = "process"
)

// Attributes for RabbitMQ
const (
	// RabbitMQ message routing key.
	//
	// Type: string
	// Required: Unless it is empty.
	// Stability: stable
	// Examples: 'myKey'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingRabbitmqRoutingKey = "messaging.rabbitmq.routing_key"
)

// Attributes for Apache Kafka
const (
	// Message keys in Kafka are used for grouping alike messages to ensure they're
	// processed on the same partition. They differ from messaging.message_id in that
	// they're not unique. If the key is null, the attribute MUST NOT be set.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'myKey'
	// Note: If the key type is not string, it's string representation has to be
	// supplied for the attribute. If the key has no unambiguous, canonical string
	// form, don't include its value.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingKafkaMessageKey = "messaging.kafka.message_key"
	// Name of the Kafka Consumer Group that is handling the message. Only applies to
	// consumers, not producers.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'my-group'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingKafkaConsumerGroup = "messaging.kafka.consumer_group"
	// Client ID for the Consumer or Producer that is handling the message.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'client-5'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingKafkaClientID = "messaging.kafka.client_id"
	// Partition the message is sent to.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingKafkaPartition = "messaging.kafka.partition"
	// A boolean that is true if the message is a tombstone.
	//
	// Type: boolean
	// Required: If missing, it is assumed to be false.
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeMessagingKafkaTombstone = "messaging.kafka.tombstone"
)

// This document defines semantic conventions for remote procedure calls.
const (
	// A string identifying the remoting system.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'grpc', 'java_rmi', 'wcf'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCSystem = "rpc.system"
	// The full (logical) name of the service being called, including its package
	// name, if applicable.
	//
	// Type: string
	// Required: No, but recommended
	// Stability: stable
	// Examples: 'myservice.EchoService'
	// Note: This is the logical name of the service from the RPC interface
	// perspective, which can be different from the name of any implementing class.
	// The code.namespace attribute may be used to store the latter (despite the
	// attribute name, it may include a class name; e.g., class with method actually
	// executing the call on the server side, RPC client stub class on the client
	// side).
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCService = "rpc.service"
	// The name of the (logical) method being called, must be equal to the $method
	// part in the span name.
	//
	// Type: string
	// Required: No, but recommended
	// Stability: stable
	// Examples: 'exampleMethod'
	// Note: This is the logical name of the method from the RPC interface
	// perspective, which can be different from the name of any implementing
	// method/function. The code.function attribute may be used to store the latter
	// (e.g., method actually executing the call on the server side, RPC client stub
	// method on the client side).
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCMethod = "rpc.method"
)

// Tech-specific attributes for gRPC.
const (
	// The numeric status code of the gRPC request.
	//
	// Type: Enum
	// Required: Always
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCode = "rpc.grpc.status_code"
)

const (
	// OK
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeOk = "0"
	// CANCELLED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeCancelled = "1"
	// UNKNOWN
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeUnknown = "2"
	// INVALID_ARGUMENT
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeInvalidArgument = "3"
	// DEADLINE_EXCEEDED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeDeadlineExceeded = "4"
	// NOT_FOUND
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeNotFound = "5"
	// ALREADY_EXISTS
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeAlreadyExists = "6"
	// PERMISSION_DENIED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodePermissionDenied = "7"
	// RESOURCE_EXHAUSTED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeResourceExhausted = "8"
	// FAILED_PRECONDITION
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeFailedPrecondition = "9"
	// ABORTED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeAborted = "10"
	// OUT_OF_RANGE
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeOutOfRange = "11"
	// UNIMPLEMENTED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeUnimplemented = "12"
	// INTERNAL
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeInternal = "13"
	// UNAVAILABLE
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeUnavailable = "14"
	// DATA_LOSS
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeDataLoss = "15"
	// UNAUTHENTICATED
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCGRPCStatusCodeUnauthenticated = "16"
)

// Tech-specific attributes for [JSON RPC](https://www.jsonrpc.org/).
const (
	// Protocol version as in jsonrpc property of request/response. Since JSON-RPC 1.0
	// does not specify this, the value can be omitted.
	//
	// Type: string
	// Required: If missing, it is assumed to be "1.0".
	// Stability: stable
	// Examples: '2.0', '1.0'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCJsonrpcVersion = "rpc.jsonrpc.version"
	// id property of request or response. Since protocol allows id to be int, string,
	// null or missing (for notifications), value is expected to be cast to string for
	// simplicity. Use empty string in case of null value. Omit entirely if this is a
	// notification.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '10', 'request-7', ''
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCJsonrpcRequestID = "rpc.jsonrpc.request_id"
	// error.code property of response if it is an error response.
	//
	// Type: int
	// Required: If missing, response is assumed to be successful.
	// Stability: stable
	// Examples: -32700, 100
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCJsonrpcErrorCode = "rpc.jsonrpc.error_code"
	// error.message property of response if it is an error response.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Parse error', 'User already exists'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeRPCJsonrpcErrorMessage = "rpc.jsonrpc.error_message"
)

// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
func GetTraceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeAWSLambdaInvokedARN,
		AttributeDBSystem,
		AttributeDBConnectionString,
		AttributeDBUser,
		AttributeDBJDBCDriverClassname,
		AttributeDBName,
		AttributeDBStatement,
		AttributeDBOperation,
		AttributeDBMSSQLInstanceName,
		AttributeDBCassandraKeyspace,
		AttributeDBCassandraPageSize,
		AttributeDBCassandraConsistencyLevel,
		AttributeDBCassandraTable,
		AttributeDBCassandraIdempotence,
		AttributeDBCassandraSpeculativeExecutionCount,
		AttributeDBCassandraCoordinatorID,
		AttributeDBCassandraCoordinatorDC,
		AttributeDBHBaseNamespace,
		AttributeDBRedisDBIndex,
		AttributeDBMongoDBCollection,
		AttributeDBSQLTable,
		AttributeExceptionType,
		AttributeExceptionMessage,
		AttributeExceptionStacktrace,
		AttributeExceptionEscaped,
		AttributeFaaSTrigger,
		AttributeFaaSExecution,
		AttributeFaaSDocumentCollection,
		AttributeFaaSDocumentOperation,
		AttributeFaaSDocumentTime,
		AttributeFaaSDocumentName,
		AttributeFaaSTime,
		AttributeFaaSCron,
		AttributeFaaSColdstart,
		AttributeFaaSInvokedName,
		AttributeFaaSInvokedProvider,
		AttributeFaaSInvokedRegion,
		AttributeNetTransport,
		AttributeNetPeerIP,
		AttributeNetPeerPort,
		AttributeNetPeerName,
		AttributeNetHostIP,
		AttributeNetHostPort,
		AttributeNetHostName,
		AttributePeerService,
		AttributeEnduserID,
		AttributeEnduserRole,
		AttributeEnduserScope,
		AttributeThreadID,
		AttributeThreadName,
		AttributeCodeFunction,
		AttributeCodeNamespace,
		AttributeCodeFilepath,
		AttributeCodeLineNumber,
		AttributeHTTPMethod,
		AttributeHTTPURL,
		AttributeHTTPTarget,
		AttributeHTTPHost,
		AttributeHTTPScheme,
		AttributeHTTPStatusCode,
		AttributeHTTPFlavor,
		AttributeHTTPUserAgent,
		AttributeHTTPRequestContentLength,
		AttributeHTTPRequestContentLengthUncompressed,
		AttributeHTTPResponseContentLength,
		AttributeHTTPResponseContentLengthUncompressed,
		AttributeHTTPServerName,
		AttributeHTTPRoute,
		AttributeHTTPClientIP,
		AttributeAWSDynamoDBTableNames,
		AttributeAWSDynamoDBConsumedCapacity,
		AttributeAWSDynamoDBItemCollectionMetrics,
		AttributeAWSDynamoDBProvisionedReadCapacity,
		AttributeAWSDynamoDBProvisionedWriteCapacity,
		AttributeAWSDynamoDBConsistentRead,
		AttributeAWSDynamoDBProjection,
		AttributeAWSDynamoDBLimit,
		AttributeAWSDynamoDBAttributesToGet,
		AttributeAWSDynamoDBIndexName,
		AttributeAWSDynamoDBSelect,
		AttributeAWSDynamoDBGlobalSecondaryIndexes,
		AttributeAWSDynamoDBLocalSecondaryIndexes,
		AttributeAWSDynamoDBExclusiveStartTable,
		AttributeAWSDynamoDBTableCount,
		AttributeAWSDynamoDBScanForward,
		AttributeAWSDynamoDBSegment,
		AttributeAWSDynamoDBTotalSegments,
		AttributeAWSDynamoDBCount,
		AttributeAWSDynamoDBScannedCount,
		AttributeAWSDynamoDBAttributeDefinitions,
		AttributeAWSDynamoDBGlobalSecondaryIndexUpdates,
		AttributeMessagingSystem,
		AttributeMessagingDestination,
		AttributeMessagingDestinationKind,
		AttributeMessagingTempDestination,
		AttributeMessagingProtocol,
		AttributeMessagingProtocolVersion,
		AttributeMessagingURL,
		AttributeMessagingMessageID,
		AttributeMessagingConversationID,
		AttributeMessagingMessagePayloadSizeBytes,
		AttributeMessagingMessagePayloadCompressedSizeBytes,
		AttributeMessagingOperation,
		AttributeMessagingRabbitmqRoutingKey,
		AttributeMessagingKafkaMessageKey,
		AttributeMessagingKafkaConsumerGroup,
		AttributeMessagingKafkaClientID,
		AttributeMessagingKafkaPartition,
		AttributeMessagingKafkaTombstone,
		AttributeRPCSystem,
		AttributeRPCService,
		AttributeRPCMethod,
		AttributeRPCGRPCStatusCode,
		AttributeRPCJsonrpcVersion,
		AttributeRPCJsonrpcRequestID,
		AttributeRPCJsonrpcErrorCode,
		AttributeRPCJsonrpcErrorMessage,
	}
}
