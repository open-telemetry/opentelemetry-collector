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

// OpenTelemetry Semantic Convention attribute names for database related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/database.md
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
