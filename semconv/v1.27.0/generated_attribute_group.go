// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// Namespace: android
const (

	// Uniquely identifies the framework API revision offered by a version (`os.version`) of the android operating system. More information can be found [here].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "33",
	// "32",
	//
	// [here]: https://developer.android.com/guide/topics/manifest/uses-sdk-element#ApiLevels
	AttributeAndroidOSAPILevel = "android.os.api_level"

	// Deprecated use the `device.app.lifecycle` event definition including `android.state` as a payload field instead.
	//
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `device.app.lifecycle`.
	//
	// Examples: undefined
	// Note: The Android lifecycle states are defined in [Activity lifecycle callbacks], and from which the `OS identifiers` are derived
	//
	// [Activity lifecycle callbacks]: https://developer.android.com/guide/components/activities/activity-lifecycle#lc
	AttributeAndroidState = "android.state"
)

// Enum values for android.state
const (
	// Any time before Activity.onResume() or, if the app has no Activity, Context.startService() has been called in the app for the first time
	AttributeAndroidStateCreated = "created"
	// Any time after Activity.onPause() or, if the app has no Activity, Context.stopService() has been called when the app was in the foreground state
	AttributeAndroidStateBackground = "background"
	// Any time after Activity.onResume() or, if the app has no Activity, Context.startService() has been called when the app was in either the created or background states
	AttributeAndroidStateForeground = "foreground"
)

// Namespace: artifact
const (

	// The provenance filename of the built attestation which directly relates to the build artifact filename. This filename SHOULD accompany the artifact at publish time. See the [SLSA Relationship] specification for more information.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "golang-binary-amd64-v0.1.0.attestation",
	// "docker-image-amd64-v0.1.0.intoto.json1",
	// "release-1.tar.gz.attestation",
	// "file-name-package.tar.gz.intoto.json1",
	//
	// [SLSA Relationship]: https://slsa.dev/spec/v1.0/distributing-provenance#relationship-between-artifacts-and-attestations
	AttributeArtifactAttestationFilename = "artifact.attestation.filename"

	// The full [hash value (see glossary)], of the built attestation. Some envelopes in the software attestation space also refer to this as the [digest].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1b31dfcd5b7f9267bf2ff47651df1cfb9147b9e4df1f335accf65b4cda498408",
	//
	// [hash value (see glossary)]: https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf
	// [digest]: https://github.com/in-toto/attestation/blob/main/spec/README.md#in-toto-attestation-framework-spec
	AttributeArtifactAttestationHash = "artifact.attestation.hash"

	// The id of the build [software attestation].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "123",
	//
	// [software attestation]: https://slsa.dev/attestation-model
	AttributeArtifactAttestationID = "artifact.attestation.id"

	// The human readable file name of the artifact, typically generated during build and release processes. Often includes the package name and version in the file name.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "golang-binary-amd64-v0.1.0",
	// "docker-image-amd64-v0.1.0",
	// "release-1.tar.gz",
	// "file-name-package.tar.gz",
	//
	// Note: This file name can also act as the [Package Name]
	// in cases where the package ecosystem maps accordingly.
	// Additionally, the artifact [can be published]
	// for others, but that is not a guarantee
	//
	// [Package Name]: https://slsa.dev/spec/v1.0/terminology#package-model
	// [can be published]: https://slsa.dev/spec/v1.0/terminology#software-supply-chain
	AttributeArtifactFilename = "artifact.filename"

	// The full [hash value (see glossary)], often found in checksum.txt on a release of the artifact and used to verify package integrity.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "9ff4c52759e2c4ac70b7d517bc7fcdc1cda631ca0045271ddd1b192544f8a3e9",
	//
	// Note: The specific algorithm used to create the cryptographic hash value is
	// not defined. In situations where an artifact has multiple
	// cryptographic hashes, it is up to the implementer to choose which
	// hash value to set here; this should be the most secure hash algorithm
	// that is suitable for the situation and consistent with the
	// corresponding attestation. The implementer can then provide the other
	// hash values through an additional set of attribute extensions as they
	// deem necessary
	//
	// [hash value (see glossary)]: https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf
	AttributeArtifactHash = "artifact.hash"

	// The [Package URL] of the [package artifact] provides a standard way to identify and locate the packaged artifact.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "pkg:github/package-url/purl-spec@1209109710924",
	// "pkg:npm/foo@12.12.3",
	//
	// [Package URL]: https://github.com/package-url/purl-spec
	// [package artifact]: https://slsa.dev/spec/v1.0/terminology#package-model
	AttributeArtifactPurl = "artifact.purl"

	// The version of the artifact.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "v0.1.0",
	// "1.2.1",
	// "122691-build",
	AttributeArtifactVersion = "artifact.version"
)

// Namespace: aspnetcore
const (

	// ASP.NET Core exception middleware handling result
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "handled",
	// "unhandled",
	AttributeAspnetcoreDiagnosticsExceptionResult = "aspnetcore.diagnostics.exception.result"

	// Full type name of the [`IExceptionHandler`] implementation that handled the exception.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "Contoso.MyHandler",
	//
	// [`IExceptionHandler`]: https://learn.microsoft.com/dotnet/api/microsoft.aspnetcore.diagnostics.iexceptionhandler
	AttributeAspnetcoreDiagnosticsHandlerType = "aspnetcore.diagnostics.handler.type"

	// Rate limiting policy name.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "fixed",
	// "sliding",
	// "token",
	AttributeAspnetcoreRateLimitingPolicy = "aspnetcore.rate_limiting.policy"

	// Rate-limiting result, shows whether the lease was acquired or contains a rejection reason
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "acquired",
	// "request_canceled",
	AttributeAspnetcoreRateLimitingResult = "aspnetcore.rate_limiting.result"

	// Flag indicating if request was handled by the application pipeline.
	// Stability: Stable
	// Type: boolean
	//
	// Examples:
	// true,
	AttributeAspnetcoreRequestIsUnhandled = "aspnetcore.request.is_unhandled"

	// A value that indicates whether the matched route is a fallback route.
	// Stability: Stable
	// Type: boolean
	//
	// Examples:
	// true,
	AttributeAspnetcoreRoutingIsFallback = "aspnetcore.routing.is_fallback"

	// Match result - success or failure
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "success",
	// "failure",
	AttributeAspnetcoreRoutingMatchStatus = "aspnetcore.routing.match_status"
)

// Enum values for aspnetcore.diagnostics.exception.result
const (
	// Exception was handled by the exception handling middleware
	AttributeAspnetcoreDiagnosticsExceptionResultHandled = "handled"
	// Exception was not handled by the exception handling middleware
	AttributeAspnetcoreDiagnosticsExceptionResultUnhandled = "unhandled"
	// Exception handling was skipped because the response had started
	AttributeAspnetcoreDiagnosticsExceptionResultSkipped = "skipped"
	// Exception handling didn't run because the request was aborted
	AttributeAspnetcoreDiagnosticsExceptionResultAborted = "aborted"
)

// Enum values for aspnetcore.rate_limiting.result
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

// Enum values for aspnetcore.routing.match_status
const (
	// Match succeeded
	AttributeAspnetcoreRoutingMatchStatusSuccess = "success"
	// Match failed
	AttributeAspnetcoreRoutingMatchStatusFailure = "failure"
)

// Namespace: aws
const (

	// The JSON-serialized value of each item in the `AttributeDefinitions` request field.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "{ "AttributeName": "string", "AttributeType": "string" }",
	// ],
	AttributeAWSDynamoDBAttributeDefinitions = "aws.dynamodb.attribute_definitions"

	// The value of the `AttributesToGet` request parameter.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "lives",
	// "id",
	// ],
	AttributeAWSDynamoDBAttributesToGet = "aws.dynamodb.attributes_to_get"

	// The value of the `ConsistentRead` request parameter.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeAWSDynamoDBConsistentRead = "aws.dynamodb.consistent_read"

	// The JSON-serialized value of each item in the `ConsumedCapacity` response field.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "{ "CapacityUnits": number, "GlobalSecondaryIndexes": { "string" : { "CapacityUnits": number, "ReadCapacityUnits": number, "WriteCapacityUnits": number } }, "LocalSecondaryIndexes": { "string" : { "CapacityUnits": number, "ReadCapacityUnits": number, "WriteCapacityUnits": number } }, "ReadCapacityUnits": number, "Table": { "CapacityUnits": number, "ReadCapacityUnits": number, "WriteCapacityUnits": number }, "TableName": "string", "WriteCapacityUnits": number }",
	// ],
	AttributeAWSDynamoDBConsumedCapacity = "aws.dynamodb.consumed_capacity"

	// The value of the `Count` response parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 10,
	AttributeAWSDynamoDBCount = "aws.dynamodb.count"

	// The value of the `ExclusiveStartTableName` request parameter.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Users",
	// "CatsTable",
	AttributeAWSDynamoDBExclusiveStartTable = "aws.dynamodb.exclusive_start_table"

	// The JSON-serialized value of each item in the `GlobalSecondaryIndexUpdates` request field.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "{ "Create": { "IndexName": "string", "KeySchema": [ { "AttributeName": "string", "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ], "ProjectionType": "string" }, "ProvisionedThroughput": { "ReadCapacityUnits": number, "WriteCapacityUnits": number } }",
	// ],
	AttributeAWSDynamoDBGlobalSecondaryIndexUpdates = "aws.dynamodb.global_secondary_index_updates"

	// The JSON-serialized value of each item of the `GlobalSecondaryIndexes` request field
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "{ "IndexName": "string", "KeySchema": [ { "AttributeName": "string", "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ], "ProjectionType": "string" }, "ProvisionedThroughput": { "ReadCapacityUnits": number, "WriteCapacityUnits": number } }",
	// ],
	AttributeAWSDynamoDBGlobalSecondaryIndexes = "aws.dynamodb.global_secondary_indexes"

	// The value of the `IndexName` request parameter.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "name_to_group",
	AttributeAWSDynamoDBIndexName = "aws.dynamodb.index_name"

	// The JSON-serialized value of the `ItemCollectionMetrics` response field.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "{ "string" : [ { "ItemCollectionKey": { "string" : { "B": blob, "BOOL": boolean, "BS": [ blob ], "L": [ "AttributeValue" ], "M": { "string" : "AttributeValue" }, "N": "string", "NS": [ "string" ], "NULL": boolean, "S": "string", "SS": [ "string" ] } }, "SizeEstimateRangeGB": [ number ] } ] }",
	AttributeAWSDynamoDBItemCollectionMetrics = "aws.dynamodb.item_collection_metrics"

	// The value of the `Limit` request parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 10,
	AttributeAWSDynamoDBLimit = "aws.dynamodb.limit"

	// The JSON-serialized value of each item of the `LocalSecondaryIndexes` request field.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "{ "IndexArn": "string", "IndexName": "string", "IndexSizeBytes": number, "ItemCount": number, "KeySchema": [ { "AttributeName": "string", "KeyType": "string" } ], "Projection": { "NonKeyAttributes": [ "string" ], "ProjectionType": "string" } }",
	// ],
	AttributeAWSDynamoDBLocalSecondaryIndexes = "aws.dynamodb.local_secondary_indexes"

	// The value of the `ProjectionExpression` request parameter.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Title",
	// "Title, Price, Color",
	// "Title, Description, RelatedItems, ProductReviews",
	AttributeAWSDynamoDBProjection = "aws.dynamodb.projection"

	// The value of the `ProvisionedThroughput.ReadCapacityUnits` request parameter.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 1.0,
	// 2.0,
	AttributeAWSDynamoDBProvisionedReadCapacity = "aws.dynamodb.provisioned_read_capacity"

	// The value of the `ProvisionedThroughput.WriteCapacityUnits` request parameter.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 1.0,
	// 2.0,
	AttributeAWSDynamoDBProvisionedWriteCapacity = "aws.dynamodb.provisioned_write_capacity"

	// The value of the `ScanIndexForward` request parameter.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeAWSDynamoDBScanForward = "aws.dynamodb.scan_forward"

	// The value of the `ScannedCount` response parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 50,
	AttributeAWSDynamoDBScannedCount = "aws.dynamodb.scanned_count"

	// The value of the `Segment` request parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 10,
	AttributeAWSDynamoDBSegment = "aws.dynamodb.segment"

	// The value of the `Select` request parameter.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "ALL_ATTRIBUTES",
	// "COUNT",
	AttributeAWSDynamoDBSelect = "aws.dynamodb.select"

	// The number of items in the `TableNames` response parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 20,
	AttributeAWSDynamoDBTableCount = "aws.dynamodb.table_count"

	// The keys in the `RequestItems` object field.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "Users",
	// "Cats",
	// ],
	AttributeAWSDynamoDBTableNames = "aws.dynamodb.table_names"

	// The value of the `TotalSegments` request parameter.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 100,
	AttributeAWSDynamoDBTotalSegments = "aws.dynamodb.total_segments"

	// The ARN of an [ECS cluster].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster",
	//
	// [ECS cluster]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/clusters.html
	AttributeAWSECSClusterARN = "aws.ecs.cluster.arn"

	// The Amazon Resource Name (ARN) of an [ECS container instance].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:ecs:us-west-1:123456789123:container/32624152-9086-4f0e-acae-1a75b14fe4d9",
	//
	// [ECS container instance]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_instances.html
	AttributeAWSECSContainerARN = "aws.ecs.container.arn"

	// The [launch type] for an ECS task.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [launch type]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/launch_types.html
	AttributeAWSECSLaunchtype = "aws.ecs.launchtype"

	// The ARN of a running [ECS task].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:ecs:us-west-1:123456789123:task/10838bed-421f-43ef-870a-f43feacbbb5b",
	// "arn:aws:ecs:us-west-1:123456789123:task/my-cluster/task-id/23ebb8ac-c18f-46c6-8bbe-d55d0e37cfbd",
	//
	// [ECS task]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-account-settings.html#ecs-resource-ids
	AttributeAWSECSTaskARN = "aws.ecs.task.arn"

	// The family name of the [ECS task definition] used to create the ECS task.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry-family",
	//
	// [ECS task definition]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_definitions.html
	AttributeAWSECSTaskFamily = "aws.ecs.task.family"

	// The ID of a running ECS task. The ID MUST be extracted from `task.arn`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "10838bed-421f-43ef-870a-f43feacbbb5b",
	// "23ebb8ac-c18f-46c6-8bbe-d55d0e37cfbd",
	AttributeAWSECSTaskID = "aws.ecs.task.id"

	// The revision for the task definition used to create the ECS task.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "8",
	// "26",
	AttributeAWSECSTaskRevision = "aws.ecs.task.revision"

	// The ARN of an EKS cluster.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster",
	AttributeAWSEKSClusterARN = "aws.eks.cluster.arn"

	// The full invoked ARN as provided on the `Context` passed to the function (`Lambda-Runtime-Invoked-Function-Arn` header on the `/runtime/invocation/next` applicable).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:lambda:us-east-1:123456:function:myfunction:myalias",
	//
	// Note: This may be different from `cloud.resource_id` if an alias is involved
	AttributeAWSLambdaInvokedARN = "aws.lambda.invoked_arn"

	// The Amazon Resource Name(s) (ARN) of the AWS log group(s).
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:*",
	// ],
	//
	// Note: See the [log group ARN format documentation]
	//
	// [log group ARN format documentation]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/iam-access-control-overview-cwl.html#CWL_ARN_Format
	AttributeAWSLogGroupARNs = "aws.log.group.arns"

	// The name(s) of the AWS log group(s) an application is writing to.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "/aws/lambda/my-function",
	// "opentelemetry-service",
	// ],
	//
	// Note: Multiple log groups must be supported for cases like multi-container applications, where a single application has sidecar containers, and each write to their own log group
	AttributeAWSLogGroupNames = "aws.log.group.names"

	// The ARN(s) of the AWS log stream(s).
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:log-stream:logs/main/10838bed-421f-43ef-870a-f43feacbbb5b",
	// ],
	//
	// Note: See the [log stream ARN format documentation]. One log group can contain several log streams, so these ARNs necessarily identify both a log group and a log stream
	//
	// [log stream ARN format documentation]: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/iam-access-control-overview-cwl.html#CWL_ARN_Format
	AttributeAWSLogStreamARNs = "aws.log.stream.arns"

	// The name(s) of the AWS log stream(s) an application is writing to.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "logs/main/10838bed-421f-43ef-870a-f43feacbbb5b",
	// ],
	AttributeAWSLogStreamNames = "aws.log.stream.names"

	// The AWS request ID as returned in the response headers `x-amz-request-id` or `x-amz-requestid`.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "79b9da39-b7ae-508a-a6bc-864b2829c622",
	// "C9ER4AJX75574TDJ",
	AttributeAWSRequestID = "aws.request_id"

	// The S3 bucket name the request refers to. Corresponds to the `--bucket` parameter of the [S3 API] operations.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "some-bucket-name",
	//
	// Note: The `bucket` attribute is applicable to all S3 operations that reference a bucket, i.e. that require the bucket name as a mandatory parameter.
	// This applies to almost all S3 operations except `list-buckets`
	//
	// [S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/index.html
	AttributeAWSS3Bucket = "aws.s3.bucket"

	// The source object (in the form `bucket`/`key`) for the copy operation.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "someFile.yml",
	//
	// Note: The `copy_source` attribute applies to S3 copy operations and corresponds to the `--copy-source` parameter
	// of the [copy-object operation within the S3 API].
	// This applies in particular to the following operations:
	//
	//   - [copy-object]
	//   - [upload-part-copy]
	// [copy-object operation within the S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/copy-object.html
	// [copy-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/copy-object.html
	// [upload-part-copy]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part-copy.html
	AttributeAWSS3CopySource = "aws.s3.copy_source"

	// The delete request container that specifies the objects to be deleted.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Objects=[{Key=string,VersionId=string},{Key=string,VersionId=string}],Quiet=boolean",
	//
	// Note: The `delete` attribute is only applicable to the [delete-object] operation.
	// The `delete` attribute corresponds to the `--delete` parameter of the
	// [delete-objects operation within the S3 API]
	//
	// [delete-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/delete-object.html
	// [delete-objects operation within the S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/delete-objects.html
	AttributeAWSS3Delete = "aws.s3.delete"

	// The S3 object key the request refers to. Corresponds to the `--key` parameter of the [S3 API] operations.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "someFile.yml",
	//
	// Note: The `key` attribute is applicable to all object-related S3 operations, i.e. that require the object key as a mandatory parameter.
	// This applies in particular to the following operations:
	//
	//   - [copy-object]
	//   - [delete-object]
	//   - [get-object]
	//   - [head-object]
	//   - [put-object]
	//   - [restore-object]
	//   - [select-object-content]
	//   - [abort-multipart-upload]
	//   - [complete-multipart-upload]
	//   - [create-multipart-upload]
	//   - [list-parts]
	//   - [upload-part]
	//   - [upload-part-copy]
	// [S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/index.html
	// [copy-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/copy-object.html
	// [delete-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/delete-object.html
	// [get-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/get-object.html
	// [head-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/head-object.html
	// [put-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/put-object.html
	// [restore-object]: https://docs.aws.amazon.com/cli/latest/reference/s3api/restore-object.html
	// [select-object-content]: https://docs.aws.amazon.com/cli/latest/reference/s3api/select-object-content.html
	// [abort-multipart-upload]: https://docs.aws.amazon.com/cli/latest/reference/s3api/abort-multipart-upload.html
	// [complete-multipart-upload]: https://docs.aws.amazon.com/cli/latest/reference/s3api/complete-multipart-upload.html
	// [create-multipart-upload]: https://docs.aws.amazon.com/cli/latest/reference/s3api/create-multipart-upload.html
	// [list-parts]: https://docs.aws.amazon.com/cli/latest/reference/s3api/list-parts.html
	// [upload-part]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part.html
	// [upload-part-copy]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part-copy.html
	AttributeAWSS3Key = "aws.s3.key"

	// The part number of the part being uploaded in a multipart-upload operation. This is a positive integer between 1 and 10,000.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 3456,
	//
	// Note: The `part_number` attribute is only applicable to the [upload-part]
	// and [upload-part-copy] operations.
	// The `part_number` attribute corresponds to the `--part-number` parameter of the
	// [upload-part operation within the S3 API]
	//
	// [upload-part]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part.html
	// [upload-part-copy]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part-copy.html
	// [upload-part operation within the S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part.html
	AttributeAWSS3PartNumber = "aws.s3.part_number"

	// Upload ID that identifies the multipart upload.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "dfRtDYWFbkRONycy.Yxwh66Yjlx.cph0gtNBtJ",
	//
	// Note: The `upload_id` attribute applies to S3 multipart-upload operations and corresponds to the `--upload-id` parameter
	// of the [S3 API] multipart operations.
	// This applies in particular to the following operations:
	//
	//   - [abort-multipart-upload]
	//   - [complete-multipart-upload]
	//   - [list-parts]
	//   - [upload-part]
	//   - [upload-part-copy]
	// [S3 API]: https://docs.aws.amazon.com/cli/latest/reference/s3api/index.html
	// [abort-multipart-upload]: https://docs.aws.amazon.com/cli/latest/reference/s3api/abort-multipart-upload.html
	// [complete-multipart-upload]: https://docs.aws.amazon.com/cli/latest/reference/s3api/complete-multipart-upload.html
	// [list-parts]: https://docs.aws.amazon.com/cli/latest/reference/s3api/list-parts.html
	// [upload-part]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part.html
	// [upload-part-copy]: https://docs.aws.amazon.com/cli/latest/reference/s3api/upload-part-copy.html
	AttributeAWSS3UploadID = "aws.s3.upload_id"
)

// Enum values for aws.ecs.launchtype
const (
	// none
	AttributeAWSECSLaunchtypeEC2 = "ec2"
	// none
	AttributeAWSECSLaunchtypeFargate = "fargate"
)

// Namespace: az
const (

	// [Azure Resource Provider Namespace] as recognized by the client.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Microsoft.Storage",
	// "Microsoft.KeyVault",
	// "Microsoft.ServiceBus",
	//
	// [Azure Resource Provider Namespace]: https://learn.microsoft.com/azure/azure-resource-manager/management/azure-services-resource-providers
	AttributeAzNamespace = "az.namespace"

	// The unique identifier of the service request. It's generated by the Azure service and returned with the response.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "00000000-0000-0000-0000-000000000000",
	AttributeAzServiceRequestID = "az.service_request_id"
)

// Namespace: browser
const (

	// Array of brand name and version separated by a space
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// " Not A;Brand 99",
	// "Chromium 99",
	// "Chrome 99",
	// ],
	//
	// Note: This value is intended to be taken from the [UA client hints API] (`navigator.userAgentData.brands`)
	//
	// [UA client hints API]: https://wicg.github.io/ua-client-hints/#interface
	AttributeBrowserBrands = "browser.brands"

	// Preferred language of the user using the browser
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "en",
	// "en-US",
	// "fr",
	// "fr-FR",
	//
	// Note: This value is intended to be taken from the Navigator API `navigator.language`
	AttributeBrowserLanguage = "browser.language"

	// A boolean that is true if the browser is running on a mobile device
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	// Note: This value is intended to be taken from the [UA client hints API] (`navigator.userAgentData.mobile`). If unavailable, this attribute SHOULD be left unset
	//
	// [UA client hints API]: https://wicg.github.io/ua-client-hints/#interface
	AttributeBrowserMobile = "browser.mobile"

	// The platform on which the browser is running
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Windows",
	// "macOS",
	// "Android",
	//
	// Note: This value is intended to be taken from the [UA client hints API] (`navigator.userAgentData.platform`). If unavailable, the legacy `navigator.platform` API SHOULD NOT be used instead and this attribute SHOULD be left unset in order for the values to be consistent.
	// The list of possible values is defined in the [W3C User-Agent Client Hints specification]. Note that some (but not all) of these values can overlap with values in the [`os.type` and `os.name` attributes]. However, for consistency, the values in the `browser.platform` attribute should capture the exact value that the user agent provides
	//
	// [UA client hints API]: https://wicg.github.io/ua-client-hints/#interface
	// [W3C User-Agent Client Hints specification]: https://wicg.github.io/ua-client-hints/#sec-ch-ua-platform
	// [`os.type` and `os.name` attributes]: ./os.md
	AttributeBrowserPlatform = "browser.platform"
)

// Namespace: cicd
const (

	// The human readable name of the pipeline within a CI/CD system.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Build and Test",
	// "Lint",
	// "Deploy Go Project",
	// "deploy_to_environment",
	AttributeCicdPipelineName = "cicd.pipeline.name"

	// The unique identifier of a pipeline run within a CI/CD system.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "120912",
	AttributeCicdPipelineRunID = "cicd.pipeline.run.id"

	// The human readable name of a task within a pipeline. Task here most closely aligns with a [computing process] in a pipeline. Other terms for tasks include commands, steps, and procedures.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Run GoLang Linter",
	// "Go Build",
	// "go-test",
	// "deploy_binary",
	//
	// [computing process]: https://en.wikipedia.org/wiki/Pipeline_(computing)
	AttributeCicdPipelineTaskName = "cicd.pipeline.task.name"

	// The unique identifier of a task run within a pipeline.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "12097",
	AttributeCicdPipelineTaskRunID = "cicd.pipeline.task.run.id"

	// The [URL] of the pipeline run providing the complete address in order to locate and identify the pipeline run.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "https://github.com/open-telemetry/semantic-conventions/actions/runs/9753949763/job/26920038674?pr=1075",
	//
	// [URL]: https://en.wikipedia.org/wiki/URL
	AttributeCicdPipelineTaskRunURLFull = "cicd.pipeline.task.run.url.full"

	// The type of the task within a pipeline.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "build",
	// "test",
	// "deploy",
	AttributeCicdPipelineTaskType = "cicd.pipeline.task.type"
)

// Enum values for cicd.pipeline.task.type
const (
	// build
	AttributeCicdPipelineTaskTypeBuild = "build"
	// test
	AttributeCicdPipelineTaskTypeTest = "test"
	// deploy
	AttributeCicdPipelineTaskTypeDeploy = "deploy"
)

// Namespace: client
const (

	// Client address - domain name if available without reverse DNS lookup; otherwise, IP address or Unix domain socket name.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "client.example.com",
	// "10.1.2.80",
	// "/tmp/my.sock",
	//
	// Note: When observed from the server side, and when communicating through an intermediary, `client.address` SHOULD represent the client address behind any intermediaries,  for example proxies, if it's available
	AttributeClientAddress = "client.address"

	// Client port number.
	// Stability: Stable
	// Type: int
	//
	// Examples:
	// 65123,
	//
	// Note: When observed from the server side, and when communicating through an intermediary, `client.port` SHOULD represent the client port behind any intermediaries,  for example proxies, if it's available
	AttributeClientPort = "client.port"
)

// Namespace: cloud
const (

	// The cloud account ID the resource is assigned to.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "111111111111",
	// "opentelemetry",
	AttributeCloudAccountID = "cloud.account.id"

	// Cloud regions often have multiple, isolated locations known as zones to increase availability. Availability zone represents the zone where the resource is running.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "us-east-1c",
	//
	// Note: Availability zones are called "zones" on Alibaba Cloud and Google Cloud
	AttributeCloudAvailabilityZone = "cloud.availability_zone"

	// The cloud platform in use.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: The prefix of the service SHOULD match the one specified in `cloud.provider`
	AttributeCloudPlatform = "cloud.platform"

	// Name of the cloud provider.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeCloudProvider = "cloud.provider"

	// The geographical region the resource is running.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "us-central1",
	// "us-east-1",
	//
	// Note: Refer to your provider's docs to see the available regions, for example [Alibaba Cloud regions], [AWS regions], [Azure regions], [Google Cloud regions], or [Tencent Cloud regions]
	//
	// [Alibaba Cloud regions]: https://www.alibabacloud.com/help/doc-detail/40654.htm
	// [AWS regions]: https://aws.amazon.com/about-aws/global-infrastructure/regions_az/
	// [Azure regions]: https://azure.microsoft.com/global-infrastructure/geographies/
	// [Google Cloud regions]: https://cloud.google.com/about/locations
	// [Tencent Cloud regions]: https://www.tencentcloud.com/document/product/213/6091
	AttributeCloudRegion = "cloud.region"

	// Cloud provider-specific native identifier of the monitored cloud resource (e.g. an [ARN] on AWS, a [fully qualified resource ID] on Azure, a [full resource name] on GCP)
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "arn:aws:lambda:REGION:ACCOUNT_ID:function:my-function",
	// "//run.googleapis.com/projects/PROJECT_ID/locations/LOCATION_ID/services/SERVICE_ID",
	// "/subscriptions/<SUBSCRIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/sites/<FUNCAPP>/functions/<FUNC>",
	//
	// Note: On some cloud providers, it may not be possible to determine the full ID at startup,
	// so it may be necessary to set `cloud.resource_id` as a span attribute instead.
	//
	// The exact value to use for `cloud.resource_id` depends on the cloud provider.
	// The following well-known definitions MUST be used if you set this attribute and they apply:
	//
	//   - **AWS Lambda:** The function [ARN].
	//     Take care not to use the "invoked ARN" directly but replace any
	//     [alias suffix]
	//     with the resolved function version, as the same runtime instance may be invocable with
	//     multiple different aliases.
	//   - **GCP:** The [URI of the resource]
	//   - **Azure:** The [Fully Qualified Resource ID] of the invoked function,
	//     *not* the function app, having the form
	//     `/subscriptions/<SUBSCRIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/sites/<FUNCAPP>/functions/<FUNC>`.
	//     This means that a span attribute MUST be used, as an Azure function app can host multiple functions that would usually share
	//     a TracerProvider
	// [ARN]: https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	// [fully qualified resource ID]: https://learn.microsoft.com/rest/api/resources/resources/get-by-id
	// [full resource name]: https://cloud.google.com/apis/design/resource_names#full_resource_name
	// [ARN]: https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	// [alias suffix]: https://docs.aws.amazon.com/lambda/latest/dg/configuration-aliases.html
	// [URI of the resource]: https://cloud.google.com/iam/docs/full-resource-names
	// [Fully Qualified Resource ID]: https://docs.microsoft.com/rest/api/resources/resources/get-by-id
	AttributeCloudResourceID = "cloud.resource_id"
)

// Enum values for cloud.platform
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

// Enum values for cloud.provider
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

// Namespace: cloudevents
const (

	// The [event_id] uniquely identifies the event.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "123e4567-e89b-12d3-a456-426614174000",
	// "0001",
	//
	// [event_id]: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#id
	AttributeCloudeventsEventID = "cloudevents.event_id"

	// The [source] identifies the context in which an event happened.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "https://github.com/cloudevents",
	// "/cloudevents/spec/pull/123",
	// "my-service",
	//
	// [source]: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#source-1
	AttributeCloudeventsEventSource = "cloudevents.event_source"

	// The [version of the CloudEvents specification] which the event uses.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "1.0"
	//
	// [version of the CloudEvents specification]: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#specversion
	AttributeCloudeventsEventSpecVersion = "cloudevents.event_spec_version"

	// The [subject] of the event in the context of the event producer (identified by source).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "mynewfile.jpg"
	//
	// [subject]: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#subject
	AttributeCloudeventsEventSubject = "cloudevents.event_subject"

	// The [event_type] contains a value describing the type of event related to the originating occurrence.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "com.github.pull_request.opened",
	// "com.example.object.deleted.v2",
	//
	// [event_type]: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#type
	AttributeCloudeventsEventType = "cloudevents.event_type"
)

// Namespace: code
const (

	// The column number in `code.filepath` best representing the operation. It SHOULD point within the code unit named in `code.function`.
	//
	// Stability: Experimental
	// Type: int
	AttributeCodeColumn = "code.column"

	// The source code file name that identifies the code unit as uniquely as possible (preferably an absolute file path).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "/usr/local/MyApplication/content_root/app/index.php"
	AttributeCodeFilepath = "code.filepath"

	// The method or function name, or equivalent (usually rightmost part of the code unit's name).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "serveRequest"
	AttributeCodeFunction = "code.function"

	// The line number in `code.filepath` best representing the operation. It SHOULD point within the code unit named in `code.function`.
	//
	// Stability: Experimental
	// Type: int
	AttributeCodeLineno = "code.lineno"

	// The "namespace" within which `code.function` is defined. Usually the qualified class or module name, such that `code.namespace` + some separator + `code.function` form a unique identifier for the code unit.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "com.example.MyHttpService"
	AttributeCodeNamespace = "code.namespace"

	// A stacktrace as a string in the natural representation for the language runtime. The representation is to be determined and documented by each language SIG.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\n at com.example.GenerateTrace.main(GenerateTrace.java:5)\n"
	AttributeCodeStacktrace = "code.stacktrace"
)

// Namespace: container
const (

	// The command used to run the container (i.e. the command name).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "otelcontribcol",
	//
	// Note: If using embedded credentials or sensitive data, it is recommended to remove them to prevent potential leakage
	AttributeContainerCommand = "container.command"

	// All the command arguments (including the command/executable itself) run by the container. [2]
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "otelcontribcol",
	// "--config",
	// "config.yaml",
	// ],
	AttributeContainerCommandArgs = "container.command_args"

	// The full command run by the container as a single string representing the full command. [2]
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "otelcontribcol --config config.yaml",
	AttributeContainerCommandLine = "container.command_line"

	// Deprecated, use `cpu.mode` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `cpu.mode`
	//
	// Examples:
	// "user",
	// "kernel",
	AttributeContainerCPUState = "container.cpu.state"

	// Container ID. Usually a UUID, as for example used to [identify Docker containers]. The UUID might be abbreviated.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "a3bf90e006b2",
	//
	// [identify Docker containers]: https://docs.docker.com/engine/containers/run/#container-identification
	AttributeContainerID = "container.id"

	// Runtime specific image identifier. Usually a hash algorithm followed by a UUID.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "sha256:19c92d0a00d1b66d897bceaa7319bee0dd38a10a851c60bcec9474aa3f01e50f",
	//
	// Note: Docker defines a sha256 of the image id; `container.image.id` corresponds to the `Image` field from the Docker container inspect [API] endpoint.
	// K8s defines a link to the container registry repository with digest `"imageID": "registry.azurecr.io /namespace/service/dockerfile@sha256:bdeabd40c3a8a492eaf9e8e44d0ebbb84bac7ee25ac0cf8a7159d25f62555625"`.
	// The ID is assigned by the container runtime and can vary in different environments. Consider using `oci.manifest.digest` if it is important to identify the same image in different environments/runtimes
	//
	// [API]: https://docs.docker.com/engine/api/v1.43/#tag/Container/operation/ContainerInspect
	AttributeContainerImageID = "container.image.id"

	// Name of the image the container was built on.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "gcr.io/opentelemetry/operator",
	AttributeContainerImageName = "container.image.name"

	// Repo digests of the container image as provided by the container runtime.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "example@sha256:afcc7f1ac1b49db317a7196c902e61c6c3c4607d63599ee1a82d702d249a0ccb",
	// "internal.registry.example.com:5000/example@sha256:b69959407d21e8a062e0416bf13405bb2b71ed7a84dde4158ebafacfa06f5578",
	// ],
	//
	// Note: [Docker] and [CRI] report those under the `RepoDigests` field
	//
	// [Docker]: https://docs.docker.com/engine/api/v1.43/#tag/Image/operation/ImageInspect
	// [CRI]: https://github.com/kubernetes/cri-api/blob/c75ef5b473bbe2d0a4fc92f82235efd665ea8e9f/pkg/apis/runtime/v1/api.proto#L1237-L1238
	AttributeContainerImageRepoDigests = "container.image.repo_digests"

	// Container image tags. An example can be found in [Docker Image Inspect]. Should be only the `<tag>` section of the full name for example from `registry.example.com/my-org/my-image:<tag>`.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "v1.27.1",
	// "3.5.7-0",
	// ],
	//
	// [Docker Image Inspect]: https://docs.docker.com/engine/api/v1.43/#tag/Image/operation/ImageInspect
	AttributeContainerImageTags = "container.image.tags"

	// Container labels, `<key>` being the label name, the value being the label value.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "container.label.app=nginx",
	AttributeContainerLabel = "container.label"

	// Deprecated, use `container.label` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `container.label`.
	//
	// Examples:
	// "container.label.app=nginx",
	AttributeContainerLabels = "container.labels"

	// Container name used by container runtime.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry-autoconf",
	AttributeContainerName = "container.name"

	// The container runtime managing this container.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "docker",
	// "containerd",
	// "rkt",
	AttributeContainerRuntime = "container.runtime"
)

// Enum values for container.cpu.state
const (
	// When tasks of the cgroup are in user mode (Linux). When all container processes are in user mode (Windows)
	AttributeContainerCPUStateUser = "user"
	// When CPU is used by the system (host OS)
	AttributeContainerCPUStateSystem = "system"
	// When tasks of the cgroup are in kernel mode (Linux). When all container processes are in kernel mode (Windows)
	AttributeContainerCPUStateKernel = "kernel"
)

// Namespace: cpu
const (

	// The mode of the CPU
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "user",
	// "system",
	AttributeCPUMode = "cpu.mode"
)

// Enum values for cpu.mode
const (
	// none
	AttributeCPUModeUser = "user"
	// none
	AttributeCPUModeSystem = "system"
	// none
	AttributeCPUModeNice = "nice"
	// none
	AttributeCPUModeIdle = "idle"
	// none
	AttributeCPUModeIowait = "iowait"
	// none
	AttributeCPUModeInterrupt = "interrupt"
	// none
	AttributeCPUModeSteal = "steal"
	// none
	AttributeCPUModeKernel = "kernel"
)

// Namespace: db
const (

	// The consistency level of the query. Based on consistency values from [CQL].
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [CQL]: https://docs.datastax.com/en/cassandra-oss/3.0/cassandra/dml/dmlConfigConsistency.html
	AttributeDBCassandraConsistencyLevel = "db.cassandra.consistency_level"

	// The data center of the coordinating node for a query.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "us-west-2"
	AttributeDBCassandraCoordinatorDC = "db.cassandra.coordinator.dc"

	// The ID of the coordinating node for a query.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "be13faa2-8574-4d71-926d-27f16cf8a7af"
	AttributeDBCassandraCoordinatorID = "db.cassandra.coordinator.id"

	// Whether or not the query is idempotent.
	//
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeDBCassandraIdempotence = "db.cassandra.idempotence"

	// The fetch size used for paging, i.e. how many rows will be returned at once.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 5000,
	AttributeDBCassandraPageSize = "db.cassandra.page_size"

	// The number of times a query was speculatively executed. Not set or `0` if the query was not executed speculatively.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 0,
	// 2,
	AttributeDBCassandraSpeculativeExecutionCount = "db.cassandra.speculative_execution_count"

	// Deprecated, use `db.collection.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.collection.name`.
	//
	// Examples: "mytable"
	AttributeDBCassandraTable = "db.cassandra.table"

	// The name of the connection pool; unique within the instrumented application. In case the connection pool implementation doesn't provide a name, instrumentation SHOULD use a combination of parameters that would make the name unique, for example, combining attributes `server.address`, `server.port`, and `db.namespace`, formatted as `server.address:server.port/db.namespace`. Instrumentations that generate connection pool name following different patterns SHOULD document it.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "myDataSource",
	AttributeDBClientConnectionPoolName = "db.client.connection.pool.name"

	// The state of a connection in the pool
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "idle",
	AttributeDBClientConnectionState = "db.client.connection.state"

	// Deprecated, use `db.client.connection.pool.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.client.connection.pool.name`.
	//
	// Examples:
	// "myDataSource",
	AttributeDBClientConnectionsPoolName = "db.client.connections.pool.name"

	// Deprecated, use `db.client.connection.state` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `db.client.connection.state`.
	//
	// Examples:
	// "idle",
	AttributeDBClientConnectionsState = "db.client.connections.state"

	// The name of a collection (table, container) within the database.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "public.users",
	// "customers",
	//
	// Note: It is RECOMMENDED to capture the value as provided by the application without attempting to do any case normalization.
	// If the collection name is parsed from the query text, it SHOULD be the first collection name found in the query and it SHOULD match the value provided in the query text including any schema and database name prefix.
	// For batch operations, if the individual operations are known to have the same collection name then that collection name SHOULD be used, otherwise `db.collection.name` SHOULD NOT be captured
	AttributeDBCollectionName = "db.collection.name"

	// Deprecated, use `server.address`, `server.port` attributes instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `server.address` and `server.port`.
	//
	// Examples: "Server=(localdb)\v11.0;Integrated Security=true;"
	AttributeDBConnectionString = "db.connection_string"

	// Unique Cosmos client instance id.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "3ba4827d-4422-483f-b59f-85b74211c11d"
	AttributeDBCosmosDBClientID = "db.cosmosdb.client_id"

	// Cosmos client connection mode.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeDBCosmosDBConnectionMode = "db.cosmosdb.connection_mode"

	// Deprecated, use `db.collection.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.collection.name`.
	//
	// Examples: "mytable"
	AttributeDBCosmosDBContainer = "db.cosmosdb.container"

	// Cosmos DB Operation Type.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeDBCosmosDBOperationType = "db.cosmosdb.operation_type"

	// RU consumed for that operation
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 46.18,
	// 1.0,
	AttributeDBCosmosDBRequestCharge = "db.cosmosdb.request_charge"

	// Request payload size in bytes
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	AttributeDBCosmosDBRequestContentLength = "db.cosmosdb.request_content_length"

	// Cosmos DB status code.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 200,
	// 201,
	AttributeDBCosmosDBStatusCode = "db.cosmosdb.status_code"

	// Cosmos DB sub status code.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1000,
	// 1002,
	AttributeDBCosmosDBSubStatusCode = "db.cosmosdb.sub_status_code"

	// Deprecated, use `db.namespace` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.namespace`.
	//
	// Examples:
	// "e9106fc68e3044f0b1475b04bf4ffd5f",
	AttributeDBElasticsearchClusterName = "db.elasticsearch.cluster.name"

	// Represents the human-readable identifier of the node/instance to which a request was routed.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "instance-0000000001",
	AttributeDBElasticsearchNodeName = "db.elasticsearch.node.name"

	// A dynamic value in the url path.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "db.elasticsearch.path_parts.index=test-index",
	// "db.elasticsearch.path_parts.doc_id=123",
	//
	// Note: Many Elasticsearch url paths allow dynamic values. These SHOULD be recorded in span attributes in the format `db.elasticsearch.path_parts.<key>`, where `<key>` is the url path part name. The implementation SHOULD reference the [elasticsearch schema] in order to map the path part values to their names
	//
	// [elasticsearch schema]: https://raw.githubusercontent.com/elastic/elasticsearch-specification/main/output/schema/schema.json
	AttributeDBElasticsearchPathParts = "db.elasticsearch.path_parts"

	// Deprecated, no general replacement at this time. For Elasticsearch, use `db.elasticsearch.node.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Deprecated, no general replacement at this time. For Elasticsearch, use `db.elasticsearch.node.name` instead.
	//
	// Examples: "mysql-e26b99z.example.com"
	AttributeDBInstanceID = "db.instance.id"

	// Removed, no replacement at this time.
	// Stability: Experimental
	// Type: string
	// Deprecated: Removed as not used.
	//
	// Examples:
	// "org.postgresql.Driver",
	// "com.microsoft.sqlserver.jdbc.SQLServerDriver",
	AttributeDBJDBCDriverClassname = "db.jdbc.driver_classname"

	// Deprecated, use `db.collection.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.collection.name`.
	//
	// Examples: "mytable"
	AttributeDBMongoDBCollection = "db.mongodb.collection"

	// Deprecated, SQL Server instance is now populated as a part of `db.namespace` attribute.
	// Stability: Experimental
	// Type: string
	// Deprecated: Deprecated, no replacement at this time.
	//
	// Examples: "MSSQLSERVER"
	AttributeDBMSSQLInstanceName = "db.mssql.instance_name"

	// Deprecated, use `db.namespace` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.namespace`.
	//
	// Examples:
	// "customers",
	// "main",
	AttributeDBName = "db.name"

	// The name of the database, fully qualified within the server address and port.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "customers",
	// "test.users",
	//
	// Note: If a database system has multiple namespace components, they SHOULD be concatenated (potentially using database system specific conventions) from most general to most specific namespace component, and more specific namespaces SHOULD NOT be captured without the more general namespaces, to ensure that "startswith" queries for the more general namespaces will be valid.
	// Semantic conventions for individual database systems SHOULD document what `db.namespace` means in the context of that system.
	// It is RECOMMENDED to capture the value as provided by the application without attempting to do any case normalization
	AttributeDBNamespace = "db.namespace"

	// Deprecated, use `db.operation.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.operation.name`.
	//
	// Examples:
	// "findAndModify",
	// "HMSET",
	// "SELECT",
	AttributeDBOperation = "db.operation"

	// The number of queries included in a batch operation.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 2,
	// 3,
	// 4,
	//
	// Note: Operations are only considered batches when they contain two or more operations, and so `db.operation.batch.size` SHOULD never be `1`
	AttributeDBOperationBatchSize = "db.operation.batch.size"

	// The name of the operation or command being executed.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "findAndModify",
	// "HMSET",
	// "SELECT",
	//
	// Note: It is RECOMMENDED to capture the value as provided by the application without attempting to do any case normalization.
	// If the operation name is parsed from the query text, it SHOULD be the first operation name found in the query.
	// For batch operations, if the individual operations are known to have the same operation name then that operation name SHOULD be used prepended by `BATCH `, otherwise `db.operation.name` SHOULD be `BATCH` or some other database system specific term if more applicable
	AttributeDBOperationName = "db.operation.name"

	// A query parameter used in `db.query.text`, with `<key>` being the parameter name, and the attribute value being a string representation of the parameter value.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "someval",
	// "55",
	//
	// Note: Query parameters should only be captured when `db.query.text` is parameterized with placeholders.
	// If a parameter has no name and instead is referenced only by index, then `<key>` SHOULD be the 0-based index
	AttributeDBQueryParameter = "db.query.parameter"

	// The database query being executed.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "SELECT * FROM wuser_table where username = ?",
	// "SET mykey "WuValue"",
	//
	// Note: For sanitization see [Sanitization of `db.query.text`].
	// For batch operations, if the individual operations are known to have the same query text then that query text SHOULD be used, otherwise all of the individual query texts SHOULD be concatenated with separator `; ` or some other database system specific separator if more applicable.
	// Even though parameterized query text can potentially have sensitive data, by using a parameterized query the user is giving a strong signal that any sensitive data will be passed as parameter values, and the benefit to observability of capturing the static part of the query text by default outweighs the risk
	//
	// [Sanitization of `db.query.text`]: ../../docs/database/database-spans.md#sanitization-of-dbquerytext
	AttributeDBQueryText = "db.query.text"

	// Deprecated, use `db.namespace` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `db.namespace`.
	//
	// Examples:
	// 0,
	// 1,
	// 15,
	AttributeDBRedisDatabaseIndex = "db.redis.database_index"

	// Deprecated, use `db.collection.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.collection.name`.
	//
	// Examples: "mytable"
	AttributeDBSQLTable = "db.sql.table"

	// The database statement being executed.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.query.text`.
	//
	// Examples:
	// "SELECT * FROM wuser_table",
	// "SET mykey "WuValue"",
	AttributeDBStatement = "db.statement"

	// The database management system (DBMS) product as identified by the client instrumentation.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: The actual DBMS may differ from the one identified by the client. For example, when using PostgreSQL client libraries to connect to a CockroachDB, the `db.system` is set to `postgresql` based on the instrumentation's best knowledge
	AttributeDBSystem = "db.system"

	// Deprecated, no replacement at this time.
	// Stability: Experimental
	// Type: string
	// Deprecated: No replacement at this time.
	//
	// Examples:
	// "readonly_user",
	// "reporting_user",
	AttributeDBUser = "db.user"
)

// Enum values for db.cassandra.consistency_level
const (
	// none
	AttributeDBCassandraConsistencyLevelAll = "all"
	// none
	AttributeDBCassandraConsistencyLevelEachQuorum = "each_quorum"
	// none
	AttributeDBCassandraConsistencyLevelQuorum = "quorum"
	// none
	AttributeDBCassandraConsistencyLevelLocalQuorum = "local_quorum"
	// none
	AttributeDBCassandraConsistencyLevelOne = "one"
	// none
	AttributeDBCassandraConsistencyLevelTwo = "two"
	// none
	AttributeDBCassandraConsistencyLevelThree = "three"
	// none
	AttributeDBCassandraConsistencyLevelLocalOne = "local_one"
	// none
	AttributeDBCassandraConsistencyLevelAny = "any"
	// none
	AttributeDBCassandraConsistencyLevelSerial = "serial"
	// none
	AttributeDBCassandraConsistencyLevelLocalSerial = "local_serial"
)

// Enum values for db.client.connection.state
const (
	// none
	AttributeDBClientConnectionStateIdle = "idle"
	// none
	AttributeDBClientConnectionStateUsed = "used"
)

// Enum values for db.client.connections.state
const (
	// none
	AttributeDBClientConnectionsStateIdle = "idle"
	// none
	AttributeDBClientConnectionsStateUsed = "used"
)

// Enum values for db.cosmosdb.connection_mode
const (
	// Gateway (HTTP) connections mode
	AttributeDBCosmosDBConnectionModeGateway = "gateway"
	// Direct connection
	AttributeDBCosmosDBConnectionModeDirect = "direct"
)

// Enum values for db.cosmosdb.operation_type
const (
	// none
	AttributeDBCosmosDBOperationTypeBatch = "batch"
	// none
	AttributeDBCosmosDBOperationTypeCreate = "create"
	// none
	AttributeDBCosmosDBOperationTypeDelete = "delete"
	// none
	AttributeDBCosmosDBOperationTypeExecute = "execute"
	// none
	AttributeDBCosmosDBOperationTypeExecuteJavascript = "execute_javascript"
	// none
	AttributeDBCosmosDBOperationTypeInvalid = "invalid"
	// none
	AttributeDBCosmosDBOperationTypeHead = "head"
	// none
	AttributeDBCosmosDBOperationTypeHeadFeed = "head_feed"
	// none
	AttributeDBCosmosDBOperationTypePatch = "patch"
	// none
	AttributeDBCosmosDBOperationTypeQuery = "query"
	// none
	AttributeDBCosmosDBOperationTypeQueryPlan = "query_plan"
	// none
	AttributeDBCosmosDBOperationTypeRead = "read"
	// none
	AttributeDBCosmosDBOperationTypeReadFeed = "read_feed"
	// none
	AttributeDBCosmosDBOperationTypeReplace = "replace"
	// none
	AttributeDBCosmosDBOperationTypeUpsert = "upsert"
)

// Enum values for db.system
const (
	// Some other SQL database. Fallback only. See notes
	AttributeDBSystemOtherSQL = "other_sql"
	// Adabas (Adaptable Database System)
	AttributeDBSystemAdabas = "adabas"
	// Deprecated, use `intersystems_cache` instead
	AttributeDBSystemCache = "cache"
	// InterSystems Caché
	AttributeDBSystemIntersystemsCache = "intersystems_cache"
	// Apache Cassandra
	AttributeDBSystemCassandra = "cassandra"
	// ClickHouse
	AttributeDBSystemClickhouse = "clickhouse"
	// Deprecated, use `other_sql` instead
	AttributeDBSystemCloudscape = "cloudscape"
	// CockroachDB
	AttributeDBSystemCockroachdb = "cockroachdb"
	// Deprecated, no replacement at this time
	AttributeDBSystemColdfusion = "coldfusion"
	// Microsoft Azure Cosmos DB
	AttributeDBSystemCosmosDB = "cosmosdb"
	// Couchbase
	AttributeDBSystemCouchbase = "couchbase"
	// CouchDB
	AttributeDBSystemCouchDB = "couchdb"
	// IBM Db2
	AttributeDBSystemDb2 = "db2"
	// Apache Derby
	AttributeDBSystemDerby = "derby"
	// Amazon DynamoDB
	AttributeDBSystemDynamoDB = "dynamodb"
	// EnterpriseDB
	AttributeDBSystemEDB = "edb"
	// Elasticsearch
	AttributeDBSystemElasticsearch = "elasticsearch"
	// FileMaker
	AttributeDBSystemFilemaker = "filemaker"
	// Firebird
	AttributeDBSystemFirebird = "firebird"
	// Deprecated, use `other_sql` instead
	AttributeDBSystemFirstSQL = "firstsql"
	// Apache Geode
	AttributeDBSystemGeode = "geode"
	// H2
	AttributeDBSystemH2 = "h2"
	// SAP HANA
	AttributeDBSystemHanaDB = "hanadb"
	// Apache HBase
	AttributeDBSystemHBase = "hbase"
	// Apache Hive
	AttributeDBSystemHive = "hive"
	// HyperSQL DataBase
	AttributeDBSystemHSQLDB = "hsqldb"
	// InfluxDB
	AttributeDBSystemInfluxdb = "influxdb"
	// Informix
	AttributeDBSystemInformix = "informix"
	// Ingres
	AttributeDBSystemIngres = "ingres"
	// InstantDB
	AttributeDBSystemInstantDB = "instantdb"
	// InterBase
	AttributeDBSystemInterbase = "interbase"
	// MariaDB
	AttributeDBSystemMariaDB = "mariadb"
	// SAP MaxDB
	AttributeDBSystemMaxDB = "maxdb"
	// Memcached
	AttributeDBSystemMemcached = "memcached"
	// MongoDB
	AttributeDBSystemMongoDB = "mongodb"
	// Microsoft SQL Server
	AttributeDBSystemMSSQL = "mssql"
	// Deprecated, Microsoft SQL Server Compact is discontinued
	AttributeDBSystemMssqlcompact = "mssqlcompact"
	// MySQL
	AttributeDBSystemMySQL = "mysql"
	// Neo4j
	AttributeDBSystemNeo4j = "neo4j"
	// Netezza
	AttributeDBSystemNetezza = "netezza"
	// OpenSearch
	AttributeDBSystemOpensearch = "opensearch"
	// Oracle Database
	AttributeDBSystemOracle = "oracle"
	// Pervasive PSQL
	AttributeDBSystemPervasive = "pervasive"
	// PointBase
	AttributeDBSystemPointbase = "pointbase"
	// PostgreSQL
	AttributeDBSystemPostgreSQL = "postgresql"
	// Progress Database
	AttributeDBSystemProgress = "progress"
	// Redis
	AttributeDBSystemRedis = "redis"
	// Amazon Redshift
	AttributeDBSystemRedshift = "redshift"
	// Cloud Spanner
	AttributeDBSystemSpanner = "spanner"
	// SQLite
	AttributeDBSystemSqlite = "sqlite"
	// Sybase
	AttributeDBSystemSybase = "sybase"
	// Teradata
	AttributeDBSystemTeradata = "teradata"
	// Trino
	AttributeDBSystemTrino = "trino"
	// Vertica
	AttributeDBSystemVertica = "vertica"
)

// Namespace: deployment
const (

	// 'Deprecated, use `deployment.environment.name` instead.'
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Deprecated, use `deployment.environment.name` instead.
	//
	// Examples:
	// "staging",
	// "production",
	AttributeDeploymentEnvironment = "deployment.environment"

	// Name of the [deployment environment] (aka deployment tier).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "staging",
	// "production",
	//
	// Note: `deployment.environment.name` does not affect the uniqueness constraints defined through
	// the `service.namespace`, `service.name` and `service.instance.id` resource attributes.
	// This implies that resources carrying the following attribute combinations MUST be
	// considered to be identifying the same service:
	//
	//   - `service.name=frontend`, `deployment.environment.name=production`
	//   - `service.name=frontend`, `deployment.environment.name=staging`
	// [deployment environment]: https://wikipedia.org/wiki/Deployment_environment
	AttributeDeploymentEnvironmentName = "deployment.environment.name"

	// The id of the deployment.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1208",
	AttributeDeploymentID = "deployment.id"

	// The name of the deployment.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "deploy my app",
	// "deploy-frontend",
	AttributeDeploymentName = "deployment.name"

	// The status of the deployment.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeDeploymentStatus = "deployment.status"
)

// Enum values for deployment.status
const (
	// failed
	AttributeDeploymentStatusFailed = "failed"
	// succeeded
	AttributeDeploymentStatusSucceeded = "succeeded"
)

// Namespace: destination
const (

	// Destination address - domain name if available without reverse DNS lookup; otherwise, IP address or Unix domain socket name.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "destination.example.com",
	// "10.1.2.80",
	// "/tmp/my.sock",
	//
	// Note: When observed from the source side, and when communicating through an intermediary, `destination.address` SHOULD represent the destination address behind any intermediaries, for example proxies, if it's available
	AttributeDestinationAddress = "destination.address"

	// Destination port number
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 3389,
	// 2888,
	AttributeDestinationPort = "destination.port"
)

// Namespace: device
const (

	// A unique identifier representing the device
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2ab2916d-a51f-4ac8-80ee-45ac31a28092",
	//
	// Note: The device identifier MUST only be defined using the values outlined below. This value is not an advertising identifier and MUST NOT be used as such. On iOS (Swift or Objective-C), this value MUST be equal to the [vendor identifier]. On Android (Java or Kotlin), this value MUST be equal to the Firebase Installation ID or a globally unique UUID which is persisted across sessions in your application. More information can be found [here] on best practices and exact implementation details. Caution should be taken when storing personal data or anything which can identify a user. GDPR and data protection laws may apply, ensure you do your own due diligence
	//
	// [vendor identifier]: https://developer.apple.com/documentation/uikit/uidevice/1620059-identifierforvendor
	// [here]: https://developer.android.com/training/articles/user-data-ids
	AttributeDeviceID = "device.id"

	// The name of the device manufacturer
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Apple",
	// "Samsung",
	//
	// Note: The Android OS provides this field via [Build]. iOS apps SHOULD hardcode the value `Apple`
	//
	// [Build]: https://developer.android.com/reference/android/os/Build#MANUFACTURER
	AttributeDeviceManufacturer = "device.manufacturer"

	// The model identifier for the device
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "iPhone3,4",
	// "SM-G920F",
	//
	// Note: It's recommended this value represents a machine-readable version of the model identifier rather than the market or consumer-friendly name of the device
	AttributeDeviceModelIdentifier = "device.model.identifier"

	// The marketing name for the device model
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "iPhone 6s Plus",
	// "Samsung Galaxy S6",
	//
	// Note: It's recommended this value represents a human-readable version of the device model rather than a machine-readable alternative
	AttributeDeviceModelName = "device.model.name"
)

// Namespace: disk
const (

	// The disk IO operation direction.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "read",
	AttributeDiskIoDirection = "disk.io.direction"
)

// Enum values for disk.io.direction
const (
	// none
	AttributeDiskIoDirectionRead = "read"
	// none
	AttributeDiskIoDirectionWrite = "write"
)

// Namespace: dns
const (

	// The name being queried.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "www.example.com",
	// "opentelemetry.io",
	//
	// Note: If the name field contains non-printable characters (below 32 or above 126), those characters should be represented as escaped base 10 integers (\DDD). Back slashes and quotes should be escaped. Tabs, carriage returns, and line feeds should be converted to \t, \r, and \n respectively
	AttributeDNSQuestionName = "dns.question.name"
)

// Namespace: dotnet
const (

	// Name of the garbage collector managed heap generation.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "gen0",
	// "gen1",
	// "gen2",
	AttributeDotnetGcHeapGeneration = "dotnet.gc.heap.generation"
)

// Enum values for dotnet.gc.heap.generation
const (
	// Generation 0
	AttributeDotnetGcHeapGenerationGen0 = "gen0"
	// Generation 1
	AttributeDotnetGcHeapGenerationGen1 = "gen1"
	// Generation 2
	AttributeDotnetGcHeapGenerationGen2 = "gen2"
	// Large Object Heap
	AttributeDotnetGcHeapGenerationLoh = "loh"
	// Pinned Object Heap
	AttributeDotnetGcHeapGenerationPoh = "poh"
)

// Namespace: enduser
const (

	// Deprecated, use `user.id` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `user.id` attribute.
	//
	// Examples: "username"
	AttributeEnduserID = "enduser.id"

	// Deprecated, use `user.roles` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `user.roles` attribute.
	//
	// Examples: "admin"
	AttributeEnduserRole = "enduser.role"

	// Deprecated, no replacement at this time.
	// Stability: Experimental
	// Type: string
	// Deprecated: Removed.
	//
	// Examples: "read:message, write:files"
	AttributeEnduserScope = "enduser.scope"
)

// Namespace: error
const (

	// Describes a class of error the operation ended with.
	//
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "timeout",
	// "java.net.UnknownHostException",
	// "server_certificate_invalid",
	// "500",
	//
	// Note: The `error.type` SHOULD be predictable, and SHOULD have low cardinality.
	//
	// When `error.type` is set to a type (e.g., an exception type), its
	// canonical class name identifying the type within the artifact SHOULD be used.
	//
	// Instrumentations SHOULD document the list of errors they report.
	//
	// The cardinality of `error.type` within one instrumentation library SHOULD be low.
	// Telemetry consumers that aggregate data from multiple instrumentation libraries and applications
	// should be prepared for `error.type` to have high cardinality at query time when no
	// additional filters are applied.
	//
	// If the operation has completed successfully, instrumentations SHOULD NOT set `error.type`.
	//
	// If a specific domain defines its own set of error identifiers (such as HTTP or gRPC status codes),
	// it's RECOMMENDED to:
	//
	//   - Use a domain-specific attribute
	//   - Set `error.type` to capture all errors, regardless of whether they are defined within the domain-specific set or not
	AttributeErrorType = "error.type"
)

// Enum values for error.type
const (
	// A fallback error value to be used when the instrumentation doesn't define a custom value
	AttributeErrorTypeOther = "_OTHER"
)

// Namespace: event
const (

	// Identifies the class / type of event.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "browser.mouse.click",
	// "device.app.lifecycle",
	//
	// Note: Event names are subject to the same rules as [attribute names]. Notably, event names are namespaced to avoid collisions and provide a clean separation of semantics for events in separate domains like browser, mobile, and kubernetes
	//
	// [attribute names]: /docs/general/attribute-naming.md
	AttributeEventName = "event.name"
)

// Namespace: exception
const (

	// SHOULD be set to true if the exception event is recorded at a point where it is known that the exception is escaping the scope of the span.
	//
	// Stability: Stable
	// Type: boolean
	//
	// Examples: undefined
	// Note: An exception is considered to have escaped (or left) the scope of a span,
	// if that span is ended while the exception is still logically "in flight".
	// This may be actually "in flight" in some languages (e.g. if the exception
	// is passed to a Context manager's `__exit__` method in Python) but will
	// usually be caught at the point of recording the exception in most languages.
	//
	// It is usually not possible to determine at the point where an exception is thrown
	// whether it will escape the scope of a span.
	// However, it is trivial to know that an exception
	// will escape, if one checks for an active exception just before ending the span,
	// as done in the [example for recording span exceptions].
	//
	// It follows that an exception may still escape the scope of the span
	// even if the `exception.escaped` attribute was not set or set to false,
	// since the event might have been recorded at a time where it was not
	// clear whether the exception will escape
	//
	// [example for recording span exceptions]: https://opentelemetry.io/docs/specs/semconv/exceptions/exceptions-spans/#recording-an-exception
	AttributeExceptionEscaped = "exception.escaped"

	// The exception message.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "Division by zero",
	// "Can't convert 'int' object to str implicitly",
	AttributeExceptionMessage = "exception.message"

	// A stacktrace as a string in the natural representation for the language runtime. The representation is to be determined and documented by each language SIG.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples: "Exception in thread "main" java.lang.RuntimeException: Test exception\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\n at com.example.GenerateTrace.main(GenerateTrace.java:5)\n"
	AttributeExceptionStacktrace = "exception.stacktrace"

	// The type of the exception (its fully-qualified class name, if applicable). The dynamic type of the exception should be preferred over the static type in languages that support it.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "java.net.ConnectException",
	// "OSError",
	AttributeExceptionType = "exception.type"
)

// Namespace: faas
const (

	// A boolean that is true if the serverless function is executed for the first time (aka cold-start).
	//
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeFaaSColdstart = "faas.coldstart"

	// A string containing the schedule period as [Cron Expression].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "0/5 * * * ? *"
	//
	// [Cron Expression]: https://docs.oracle.com/cd/E12058_01/doc/doc.1014/e12030/cron_expressions.htm
	AttributeFaaSCron = "faas.cron"

	// The name of the source on which the triggering operation was performed. For example, in Cloud Storage or S3 corresponds to the bucket name, and in Cosmos DB to the database name.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "myBucketName",
	// "myDbName",
	AttributeFaaSDocumentCollection = "faas.document.collection"

	// The document name/table subjected to the operation. For example, in Cloud Storage or S3 is the name of the file, and in Cosmos DB the table name.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "myFile.txt",
	// "myTableName",
	AttributeFaaSDocumentName = "faas.document.name"

	// Describes the type of the operation that was performed on the data.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeFaaSDocumentOperation = "faas.document.operation"

	// A string containing the time when the data was accessed in the [ISO 8601] format expressed in [UTC].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "2020-01-23T13:47:06Z"
	//
	// [ISO 8601]: https://www.iso.org/iso-8601-date-and-time-format.html
	// [UTC]: https://www.w3.org/TR/NOTE-datetime
	AttributeFaaSDocumentTime = "faas.document.time"

	// The execution environment ID as a string, that will be potentially reused for other invocations to the same function/function version.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de",
	//
	// Note: * **AWS Lambda:** Use the (full) log stream name
	AttributeFaaSInstance = "faas.instance"

	// The invocation ID of the current function invocation.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "af9d5aa4-a685-4c5f-a22b-444f80b3cc28"
	AttributeFaaSInvocationID = "faas.invocation_id"

	// The name of the invoked function.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "my-function"
	// Note: SHOULD be equal to the `faas.name` resource attribute of the invoked function
	AttributeFaaSInvokedName = "faas.invoked_name"

	// The cloud provider of the invoked function.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: SHOULD be equal to the `cloud.provider` resource attribute of the invoked function
	AttributeFaaSInvokedProvider = "faas.invoked_provider"

	// The cloud region of the invoked function.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "eu-central-1"
	// Note: SHOULD be equal to the `cloud.region` resource attribute of the invoked function
	AttributeFaaSInvokedRegion = "faas.invoked_region"

	// The amount of memory available to the serverless function converted to Bytes.
	//
	// Stability: Experimental
	// Type: int
	//
	// Note: It's recommended to set this attribute since e.g. too little memory can easily stop a Java AWS Lambda function from working correctly. On AWS Lambda, the environment variable `AWS_LAMBDA_FUNCTION_MEMORY_SIZE` provides this information (which must be multiplied by 1,048,576)
	AttributeFaaSMaxMemory = "faas.max_memory"

	// The name of the single function that this runtime instance executes.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "my-function",
	// "myazurefunctionapp/some-function-name",
	//
	// Note: This is the name of the function as configured/deployed on the FaaS
	// platform and is usually different from the name of the callback
	// function (which may be stored in the
	// [`code.namespace`/`code.function`]
	// span attributes).
	//
	// For some cloud providers, the above definition is ambiguous. The following
	// definition of function name MUST be used for this attribute
	// (and consequently the span name) for the listed cloud providers/products:
	//
	//   - **Azure:**  The full name `<FUNCAPP>/<FUNC>`, i.e., function app name
	//     followed by a forward slash followed by the function name (this form
	//     can also be seen in the resource JSON for the function).
	//     This means that a span attribute MUST be used, as an Azure function
	//     app can host multiple functions that would usually share
	//     a TracerProvider (see also the `cloud.resource_id` attribute)
	// [`code.namespace`/`code.function`]: /docs/general/attributes.md#source-code-attributes
	AttributeFaaSName = "faas.name"

	// A string containing the function invocation time in the [ISO 8601] format expressed in [UTC].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "2020-01-23T13:47:06Z"
	//
	// [ISO 8601]: https://www.iso.org/iso-8601-date-and-time-format.html
	// [UTC]: https://www.w3.org/TR/NOTE-datetime
	AttributeFaaSTime = "faas.time"

	// Type of the trigger which caused this function invocation.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeFaaSTrigger = "faas.trigger"

	// The immutable version of the function being executed.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "26",
	// "pinkfroid-00002",
	//
	// Note: Depending on the cloud provider and platform, use:
	//
	//   - **AWS Lambda:** The [function version]
	//     (an integer represented as a decimal string).
	//   - **Google Cloud Run (Services):** The [revision]
	//     (i.e., the function name plus the revision suffix).
	//   - **Google Cloud Functions:** The value of the
	//     [`K_REVISION` environment variable].
	//   - **Azure Functions:** Not applicable. Do not set this attribute
	// [function version]: https://docs.aws.amazon.com/lambda/latest/dg/configuration-versions.html
	// [revision]: https://cloud.google.com/run/docs/managing/revisions
	// [`K_REVISION` environment variable]: https://cloud.google.com/functions/docs/env-var#runtime_environment_variables_set_automatically
	AttributeFaaSVersion = "faas.version"
)

// Enum values for faas.document.operation
const (
	// When a new object is created
	AttributeFaaSDocumentOperationInsert = "insert"
	// When an object is modified
	AttributeFaaSDocumentOperationEdit = "edit"
	// When an object is deleted
	AttributeFaaSDocumentOperationDelete = "delete"
)

// Enum values for faas.invoked_provider
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

// Enum values for faas.trigger
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

// Namespace: feature_flag
const (

	// The unique identifier of the feature flag.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "logo-color",
	AttributeFeatureFlagKey = "feature_flag.key"

	// The name of the service provider that performs the flag evaluation.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Flag Manager",
	AttributeFeatureFlagProviderName = "feature_flag.provider_name"

	// SHOULD be a semantic identifier for a value. If one is unavailable, a stringified version of the value can be used.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "red",
	// "true",
	// "on",
	//
	// Note: A semantic identifier, commonly referred to as a variant, provides a means
	// for referring to a value without including the value itself. This can
	// provide additional context for understanding the meaning behind a value.
	// For example, the variant `red` maybe be used for the value `#c05543`.
	//
	// A stringified version of the value can be used in situations where a
	// semantic identifier is unavailable. String representation of the value
	// should be determined by the implementer
	AttributeFeatureFlagVariant = "feature_flag.variant"
)

// Namespace: file
const (

	// Directory where the file is located. It should include the drive letter, when appropriate.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/home/user",
	// "C:\Program Files\MyApp",
	AttributeFileDirectory = "file.directory"

	// File extension, excluding the leading dot.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "png",
	// "gz",
	//
	// Note: When the file name has multiple extensions (example.tar.gz), only the last one should be captured ("gz", not "tar.gz")
	AttributeFileExtension = "file.extension"

	// Name of the file including the extension, without the directory.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "example.png",
	AttributeFileName = "file.name"

	// Full path to the file, including the file name. It should include the drive letter, when appropriate.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/home/alice/example.png",
	// "C:\Program Files\MyApp\myapp.exe",
	AttributeFilePath = "file.path"

	// File size in bytes.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	AttributeFileSize = "file.size"
)

// Namespace: gcp
const (

	// Identifies the Google Cloud service for which the official client library is intended.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "appengine",
	// "run",
	// "firestore",
	// "alloydb",
	// "spanner",
	//
	// Note: Intended to be a stable identifier for Google Cloud client libraries that is uniform across implementation languages. The value should be derived from the canonical service domain for the service; for example, 'foo.googleapis.com' should result in a value of 'foo'
	AttributeGCPClientService = "gcp.client.service"

	// The name of the Cloud Run [execution] being run for the Job, as set by the [`CLOUD_RUN_EXECUTION`] environment variable.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "job-name-xxxx",
	// "sample-job-mdw84",
	//
	// [execution]: https://cloud.google.com/run/docs/managing/job-executions
	// [`CLOUD_RUN_EXECUTION`]: https://cloud.google.com/run/docs/container-contract#jobs-env-vars
	AttributeGCPCloudRunJobExecution = "gcp.cloud_run.job.execution"

	// The index for a task within an execution as provided by the [`CLOUD_RUN_TASK_INDEX`] environment variable.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 0,
	// 1,
	//
	// [`CLOUD_RUN_TASK_INDEX`]: https://cloud.google.com/run/docs/container-contract#jobs-env-vars
	AttributeGCPCloudRunJobTaskIndex = "gcp.cloud_run.job.task_index"

	// The hostname of a GCE instance. This is the full value of the default or [custom hostname].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "my-host1234.example.com",
	// "sample-vm.us-west1-b.c.my-project.internal",
	//
	// [custom hostname]: https://cloud.google.com/compute/docs/instances/custom-hostname-vm
	AttributeGCPGceInstanceHostname = "gcp.gce.instance.hostname"

	// The instance name of a GCE instance. This is the value provided by `host.name`, the visible name of the instance in the Cloud Console UI, and the prefix for the default hostname of the instance as defined by the [default internal DNS name].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "instance-1",
	// "my-vm-name",
	//
	// [default internal DNS name]: https://cloud.google.com/compute/docs/internal-dns#instance-fully-qualified-domain-names
	AttributeGCPGceInstanceName = "gcp.gce.instance.name"
)

// Namespace: gen_ai
const (

	// The full response received from the GenAI model.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "[{'role': 'assistant', 'content': 'The capital of France is Paris.'}]",
	//
	// Note: It's RECOMMENDED to format completions as JSON string matching [OpenAI messages format]
	//
	// [OpenAI messages format]: https://platform.openai.com/docs/guides/text-generation
	AttributeGenAiCompletion = "gen_ai.completion"

	// The name of the operation being performed.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: If one of the predefined values applies, but specific system uses a different name it's RECOMMENDED to document it in the semantic conventions for specific GenAI system and use system-specific name in the instrumentation. If a different name is not documented, instrumentation libraries SHOULD use applicable predefined value
	AttributeGenAiOperationName = "gen_ai.operation.name"

	// The full prompt sent to the GenAI model.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "[{'role': 'user', 'content': 'What is the capital of France?'}]",
	//
	// Note: It's RECOMMENDED to format prompts as JSON string matching [OpenAI messages format]
	//
	// [OpenAI messages format]: https://platform.openai.com/docs/guides/text-generation
	AttributeGenAiPrompt = "gen_ai.prompt"

	// The frequency penalty setting for the GenAI request.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 0.1,
	AttributeGenAiRequestFrequencyPenalty = "gen_ai.request.frequency_penalty"

	// The maximum number of tokens the model generates for a request.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 100,
	AttributeGenAiRequestMaxTokens = "gen_ai.request.max_tokens"

	// The name of the GenAI model a request is being made to.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "gpt-4"
	AttributeGenAiRequestModel = "gen_ai.request.model"

	// The presence penalty setting for the GenAI request.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 0.1,
	AttributeGenAiRequestPresencePenalty = "gen_ai.request.presence_penalty"

	// List of sequences that the model will use to stop generating further tokens.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "forest",
	// "lived",
	// ],
	AttributeGenAiRequestStopSequences = "gen_ai.request.stop_sequences"

	// The temperature setting for the GenAI request.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 0.0,
	AttributeGenAiRequestTemperature = "gen_ai.request.temperature"

	// The top_k sampling setting for the GenAI request.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 1.0,
	AttributeGenAiRequestTopK = "gen_ai.request.top_k"

	// The top_p sampling setting for the GenAI request.
	// Stability: Experimental
	// Type: double
	//
	// Examples:
	// 1.0,
	AttributeGenAiRequestTopP = "gen_ai.request.top_p"

	// Array of reasons the model stopped generating tokens, corresponding to each generation received.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "stop",
	// ],
	// [
	// "stop",
	// "length",
	// ],
	AttributeGenAiResponseFinishReasons = "gen_ai.response.finish_reasons"

	// The unique identifier for the completion.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "chatcmpl-123",
	AttributeGenAiResponseID = "gen_ai.response.id"

	// The name of the model that generated the response.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "gpt-4-0613",
	AttributeGenAiResponseModel = "gen_ai.response.model"

	// The Generative AI product as identified by the client or server instrumentation.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: "openai"
	// Note: The `gen_ai.system` describes a family of GenAI models with specific model identified
	// by `gen_ai.request.model` and `gen_ai.response.model` attributes.
	//
	// The actual GenAI product may differ from the one identified by the client.
	// For example, when using OpenAI client libraries to communicate with Mistral, the `gen_ai.system`
	// is set to `openai` based on the instrumentation's best knowledge.
	//
	// For custom model, a custom friendly name SHOULD be used.
	// If none of these options apply, the `gen_ai.system` SHOULD be set to `_OTHER`
	AttributeGenAiSystem = "gen_ai.system"

	// The type of token being counted.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "input",
	// "output",
	AttributeGenAiTokenType = "gen_ai.token.type"

	// Deprecated, use `gen_ai.usage.output_tokens` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `gen_ai.usage.output_tokens` attribute.
	//
	// Examples:
	// 42,
	AttributeGenAiUsageCompletionTokens = "gen_ai.usage.completion_tokens"

	// The number of tokens used in the GenAI input (prompt).
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 100,
	AttributeGenAiUsageInputTokens = "gen_ai.usage.input_tokens"

	// The number of tokens used in the GenAI response (completion).
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 180,
	AttributeGenAiUsageOutputTokens = "gen_ai.usage.output_tokens"

	// Deprecated, use `gen_ai.usage.input_tokens` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `gen_ai.usage.input_tokens` attribute.
	//
	// Examples:
	// 42,
	AttributeGenAiUsagePromptTokens = "gen_ai.usage.prompt_tokens"
)

// Enum values for gen_ai.operation.name
const (
	// Chat completion operation such as [OpenAI Chat API]
	//
	// [OpenAI Chat API]: https://platform.openai.com/docs/api-reference/chat
	AttributeGenAiOperationNameChat = "chat"
	// Text completions operation such as [OpenAI Completions API (Legacy)]
	//
	// [OpenAI Completions API (Legacy)]: https://platform.openai.com/docs/api-reference/completions
	AttributeGenAiOperationNameTextCompletion = "text_completion"
)

// Enum values for gen_ai.system
const (
	// OpenAI
	AttributeGenAiSystemOpenai = "openai"
	// Vertex AI
	AttributeGenAiSystemVertexAi = "vertex_ai"
	// Anthropic
	AttributeGenAiSystemAnthropic = "anthropic"
	// Cohere
	AttributeGenAiSystemCohere = "cohere"
)

// Enum values for gen_ai.token.type
const (
	// Input tokens (prompt, input, etc.)
	AttributeGenAiTokenTypeInput = "input"
	// Output tokens (completion, response, etc.)
	AttributeGenAiTokenTypeCompletion = "output"
)

// Namespace: go
const (

	// The type of memory.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "other",
	// "stack",
	AttributeGoMemoryType = "go.memory.type"
)

// Enum values for go.memory.type
const (
	// Memory allocated from the heap that is reserved for stack space, whether or not it is currently in-use
	AttributeGoMemoryTypeStack = "stack"
	// Memory used by the Go runtime, excluding other categories of memory usage described in this enumeration
	AttributeGoMemoryTypeOther = "other"
)

// Namespace: graphql
const (

	// The GraphQL document being executed.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "query findBookById { bookById(id: ?) { name } }"
	// Note: The value may be sanitized to exclude sensitive information
	AttributeGraphqlDocument = "graphql.document"

	// The name of the operation being executed.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "findBookById"
	AttributeGraphqlOperationName = "graphql.operation.name"

	// The type of the operation being executed.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "query",
	// "mutation",
	// "subscription",
	AttributeGraphqlOperationType = "graphql.operation.type"
)

// Enum values for graphql.operation.type
const (
	// GraphQL query
	AttributeGraphqlOperationTypeQuery = "query"
	// GraphQL mutation
	AttributeGraphqlOperationTypeMutation = "mutation"
	// GraphQL subscription
	AttributeGraphqlOperationTypeSubscription = "subscription"
)

// Namespace: heroku
const (

	// Unique identifier for the application
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2daa2797-e42b-4624-9322-ec3f968df4da",
	AttributeHerokuAppID = "heroku.app.id"

	// Commit hash for the current release
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "e6134959463efd8966b20e75b913cafe3f5ec",
	AttributeHerokuReleaseCommit = "heroku.release.commit"

	// Time and date the release was created
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2022-10-23T18:00:42Z",
	AttributeHerokuReleaseCreationTimestamp = "heroku.release.creation_timestamp"
)

// Namespace: host
const (

	// The CPU architecture the host system is running on.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeHostArch = "host.arch"

	// The amount of level 2 memory cache available to the processor (in Bytes).
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 12288000,
	AttributeHostCPUCacheL2Size = "host.cpu.cache.l2.size"

	// Family or generation of the CPU.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "6",
	// "PA-RISC 1.1e",
	AttributeHostCPUFamily = "host.cpu.family"

	// Model identifier. It provides more granular information about the CPU, distinguishing it from other CPUs within the same family.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "6",
	// "9000/778/B180L",
	AttributeHostCPUModelID = "host.cpu.model.id"

	// Model designation of the processor.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz",
	AttributeHostCPUModelName = "host.cpu.model.name"

	// Stepping or core revisions.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1",
	// "r1p1",
	AttributeHostCPUStepping = "host.cpu.stepping"

	// Processor manufacturer identifier. A maximum 12-character string.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "GenuineIntel",
	//
	// Note: [CPUID] command returns the vendor ID string in EBX, EDX and ECX registers. Writing these to memory in this order results in a 12-character string
	//
	// [CPUID]: https://wiki.osdev.org/CPUID
	AttributeHostCPUVendorID = "host.cpu.vendor.id"

	// Unique host ID. For Cloud, this must be the instance_id assigned by the cloud provider. For non-containerized systems, this should be the `machine-id`. See the table below for the sources to use to determine the `machine-id` based on operating system.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "fdbf79e8af94cb7f9e8df36789187052",
	AttributeHostID = "host.id"

	// VM image ID or host OS image ID. For Cloud, this value is from the provider.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "ami-07b06b442921831e5",
	AttributeHostImageID = "host.image.id"

	// Name of the VM image or OS install the host was instantiated from.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "infra-ami-eks-worker-node-7d4ec78312",
	// "CentOS-8-x86_64-1905",
	AttributeHostImageName = "host.image.name"

	// The version string of the VM image or host OS as defined in [Version Attributes].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "0.1",
	//
	// [Version Attributes]: /docs/resource/README.md#version-attributes
	AttributeHostImageVersion = "host.image.version"

	// Available IP addresses of the host, excluding loopback interfaces.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "192.168.1.140",
	// "fe80::abc2:4a28:737a:609e",
	// ],
	//
	// Note: IPv4 Addresses MUST be specified in dotted-quad notation. IPv6 addresses MUST be specified in the [RFC 5952] format
	//
	// [RFC 5952]: https://www.rfc-editor.org/rfc/rfc5952.html
	AttributeHostIP = "host.ip"

	// Available MAC addresses of the host, excluding loopback interfaces.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "AC-DE-48-23-45-67",
	// "AC-DE-48-23-45-67-01-9F",
	// ],
	//
	// Note: MAC Addresses MUST be represented in [IEEE RA hexadecimal form]: as hyphen-separated octets in uppercase hexadecimal form from most to least significant
	//
	// [IEEE RA hexadecimal form]: https://standards.ieee.org/wp-content/uploads/import/documents/tutorials/eui.pdf
	AttributeHostMac = "host.mac"

	// Name of the host. On Unix systems, it may contain what the hostname command returns, or the fully qualified hostname, or another name specified by the user.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry-test",
	AttributeHostName = "host.name"

	// Type of host. For Cloud, this must be the machine type.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "n1-standard-1",
	AttributeHostType = "host.type"
)

// Enum values for host.arch
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

// Namespace: http
const (

	// Deprecated, use `client.address` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `client.address`.
	//
	// Examples: "83.164.160.102"
	AttributeHTTPClientIP = "http.client_ip"

	// State of the HTTP connection in the HTTP connection pool.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "active",
	// "idle",
	AttributeHTTPConnectionState = "http.connection.state"

	// Deprecated, use `network.protocol.name` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `network.protocol.name`.
	//
	// Examples: undefined
	AttributeHTTPFlavor = "http.flavor"

	// Deprecated, use one of `server.address`, `client.address` or `http.request.header.host` instead, depending on the usage.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by one of `server.address`, `client.address` or `http.request.header.host`, depending on the usage.
	//
	// Examples:
	// "www.example.org",
	AttributeHTTPHost = "http.host"

	// Deprecated, use `http.request.method` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `http.request.method`.
	//
	// Examples:
	// "GET",
	// "POST",
	// "HEAD",
	AttributeHTTPMethod = "http.method"

	// The size of the request payload body in bytes. This is the number of bytes transferred excluding headers and is often, but not always, present as the [Content-Length] header. For requests using transport encoding, this should be the compressed size.
	//
	// Stability: Experimental
	// Type: int
	//
	// [Content-Length]: https://www.rfc-editor.org/rfc/rfc9110.html#field.content-length
	AttributeHTTPRequestBodySize = "http.request.body.size"

	// HTTP request headers, `<key>` being the normalized HTTP Header name (lowercase), the value being the header values.
	//
	// Stability: Stable
	// Type: string[]
	//
	// Examples:
	// "http.request.header.content-type=["application/json"]",
	// "http.request.header.x-forwarded-for=["1.2.3.4", "1.2.3.5"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which headers are to be captured. Including all request headers can be a security risk - explicit configuration helps avoid leaking sensitive information.
	// The `User-Agent` header is already captured in the `user_agent.original` attribute. Users MAY explicitly configure instrumentations to capture them even though it is not recommended.
	// The attribute value MUST consist of either multiple header values as an array of strings or a single-item array containing a possibly comma-concatenated string, depending on the way the HTTP library provides access to headers
	AttributeHTTPRequestHeader = "http.request.header"

	// HTTP request method.
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "GET",
	// "POST",
	// "HEAD",
	//
	// Note: HTTP request method value SHOULD be "known" to the instrumentation.
	// By default, this convention defines "known" methods as the ones listed in [RFC9110]
	// and the PATCH method defined in [RFC5789].
	//
	// If the HTTP request method is not known to instrumentation, it MUST set the `http.request.method` attribute to `_OTHER`.
	//
	// If the HTTP instrumentation could end up converting valid HTTP request methods to `_OTHER`, then it MUST provide a way to override
	// the list of known HTTP methods. If this override is done via environment variable, then the environment variable MUST be named
	// OTEL_INSTRUMENTATION_HTTP_KNOWN_METHODS and support a comma-separated list of case-sensitive known HTTP methods
	// (this list MUST be a full override of the default known method, it is not a list of known methods in addition to the defaults).
	//
	// HTTP method names are case-sensitive and `http.request.method` attribute value MUST match a known HTTP method name exactly.
	// Instrumentations for specific web frameworks that consider HTTP methods to be case insensitive, SHOULD populate a canonical equivalent.
	// Tracing instrumentations that do so, MUST also set `http.request.method_original` to the original value
	//
	// [RFC9110]: https://www.rfc-editor.org/rfc/rfc9110.html#name-methods
	// [RFC5789]: https://www.rfc-editor.org/rfc/rfc5789.html
	AttributeHTTPRequestMethod = "http.request.method"

	// Original HTTP method sent by the client in the request line.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "GeT",
	// "ACL",
	// "foo",
	AttributeHTTPRequestMethodOriginal = "http.request.method_original"

	// The ordinal number of request resending attempt (for any reason, including redirects).
	//
	// Stability: Stable
	// Type: int
	//
	// Note: The resend count SHOULD be updated each time an HTTP request gets resent by the client, regardless of what was the cause of the resending (e.g. redirection, authorization failure, 503 Server Unavailable, network issues, or any other)
	AttributeHTTPRequestResendCount = "http.request.resend_count"

	// The total size of the request in bytes. This should be the total number of bytes sent over the wire, including the request line (HTTP/1.1), framing (HTTP/2 and HTTP/3), headers, and request body if any.
	//
	// Stability: Experimental
	// Type: int
	AttributeHTTPRequestSize = "http.request.size"

	// Deprecated, use `http.request.header.content-length` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `http.request.header.content-length`
	AttributeHTTPRequestContentLength = "http.request_content_length"

	// Deprecated, use `http.request.body.size` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `http.request.body.size`
	AttributeHTTPRequestContentLengthUncompressed = "http.request_content_length_uncompressed"

	// The size of the response payload body in bytes. This is the number of bytes transferred excluding headers and is often, but not always, present as the [Content-Length] header. For requests using transport encoding, this should be the compressed size.
	//
	// Stability: Experimental
	// Type: int
	//
	// [Content-Length]: https://www.rfc-editor.org/rfc/rfc9110.html#field.content-length
	AttributeHTTPResponseBodySize = "http.response.body.size"

	// HTTP response headers, `<key>` being the normalized HTTP Header name (lowercase), the value being the header values.
	//
	// Stability: Stable
	// Type: string[]
	//
	// Examples:
	// "http.response.header.content-type=["application/json"]",
	// "http.response.header.my-custom-header=["abc", "def"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which headers are to be captured. Including all response headers can be a security risk - explicit configuration helps avoid leaking sensitive information.
	// Users MAY explicitly configure instrumentations to capture them even though it is not recommended.
	// The attribute value MUST consist of either multiple header values as an array of strings or a single-item array containing a possibly comma-concatenated string, depending on the way the HTTP library provides access to headers
	AttributeHTTPResponseHeader = "http.response.header"

	// The total size of the response in bytes. This should be the total number of bytes sent over the wire, including the status line (HTTP/1.1), framing (HTTP/2 and HTTP/3), headers, and response body and trailers if any.
	//
	// Stability: Experimental
	// Type: int
	AttributeHTTPResponseSize = "http.response.size"

	// [HTTP response status code].
	// Stability: Stable
	// Type: int
	//
	// Examples:
	// 200,
	//
	// [HTTP response status code]: https://tools.ietf.org/html/rfc7231#section-6
	AttributeHTTPResponseStatusCode = "http.response.status_code"

	// Deprecated, use `http.response.header.content-length` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `http.response.header.content-length`
	AttributeHTTPResponseContentLength = "http.response_content_length"

	// Deprecated, use `http.response.body.size` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replace by `http.response.body.size`
	AttributeHTTPResponseContentLengthUncompressed = "http.response_content_length_uncompressed"

	// The matched route, that is, the path template in the format used by the respective server framework.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "/users/:userID?",
	// "{controller}/{action}/{id?}",
	//
	// Note: MUST NOT be populated when this is not supported by the HTTP server framework as the route attribute should have low-cardinality and the URI path can NOT substitute it.
	// SHOULD include the [application root] if there is one
	//
	// [application root]: /docs/http/http-spans.md#http-server-definitions
	AttributeHTTPRoute = "http.route"

	// Deprecated, use `url.scheme` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `url.scheme` instead.
	//
	// Examples:
	// "http",
	// "https",
	AttributeHTTPScheme = "http.scheme"

	// Deprecated, use `server.address` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `server.address`.
	//
	// Examples:
	// "example.com",
	AttributeHTTPServerName = "http.server_name"

	// Deprecated, use `http.response.status_code` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `http.response.status_code`.
	//
	// Examples:
	// 200,
	AttributeHTTPStatusCode = "http.status_code"

	// Deprecated, use `url.path` and `url.query` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Split to `url.path` and `url.query.
	//
	// Examples:
	// "/search?q=OpenTelemetry#SemConv",
	AttributeHTTPTarget = "http.target"

	// Deprecated, use `url.full` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `url.full`.
	//
	// Examples:
	// "https://www.foo.bar/search?q=OpenTelemetry#SemConv",
	AttributeHTTPURL = "http.url"

	// Deprecated, use `user_agent.original` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `user_agent.original`.
	//
	// Examples:
	// "CERN-LineMode/2.15 libwww/2.17b3",
	// "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
	AttributeHTTPUserAgent = "http.user_agent"
)

// Enum values for http.connection.state
const (
	// active state
	AttributeHTTPConnectionStateActive = "active"
	// idle state
	AttributeHTTPConnectionStateIdle = "idle"
)

// Enum values for http.flavor
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

// Enum values for http.request.method
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

// Namespace: hw
const (

	// An identifier for the hardware component, unique within the monitored host
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "win32battery_battery_testsysa33_1",
	AttributeHwID = "hw.id"

	// An easily-recognizable name for the hardware component
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "eth0",
	AttributeHwName = "hw.name"

	// Unique identifier of the parent component (typically the `hw.id` attribute of the enclosure, or disk controller)
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "dellStorage_perc_0",
	AttributeHwParent = "hw.parent"

	// The current state of the component
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeHwState = "hw.state"

	// Type of the component
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: Describes the category of the hardware component for which `hw.state` is being reported. For example, `hw.type=temperature` along with `hw.state=degraded` would indicate that the temperature of the hardware component has been reported as `degraded`
	AttributeHwType = "hw.type"
)

// Enum values for hw.state
const (
	// Ok
	AttributeHwStateOk = "ok"
	// Degraded
	AttributeHwStateDegraded = "degraded"
	// Failed
	AttributeHwStateFailed = "failed"
)

// Enum values for hw.type
const (
	// Battery
	AttributeHwTypeBattery = "battery"
	// CPU
	AttributeHwTypeCPU = "cpu"
	// Disk controller
	AttributeHwTypeDiskController = "disk_controller"
	// Enclosure
	AttributeHwTypeEnclosure = "enclosure"
	// Fan
	AttributeHwTypeFan = "fan"
	// GPU
	AttributeHwTypeGpu = "gpu"
	// Logical disk
	AttributeHwTypeLogicalDisk = "logical_disk"
	// Memory
	AttributeHwTypeMemory = "memory"
	// Network
	AttributeHwTypeNetwork = "network"
	// Physical disk
	AttributeHwTypePhysicalDisk = "physical_disk"
	// Power supply
	AttributeHwTypePowerSupply = "power_supply"
	// Tape drive
	AttributeHwTypeTapeDrive = "tape_drive"
	// Temperature
	AttributeHwTypeTemperature = "temperature"
	// Voltage
	AttributeHwTypeVoltage = "voltage"
)

// Namespace: ios
const (

	// Deprecated use the `device.app.lifecycle` event definition including `ios.state` as a payload field instead.
	//
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Moved to a payload field of `device.app.lifecycle`.
	//
	// Examples: undefined
	// Note: The iOS lifecycle states are defined in the [UIApplicationDelegate documentation], and from which the `OS terminology` column values are derived
	//
	// [UIApplicationDelegate documentation]: https://developer.apple.com/documentation/uikit/uiapplicationdelegate#1656902
	AttributeIosState = "ios.state"
)

// Enum values for ios.state
const (
	// The app has become `active`. Associated with UIKit notification `applicationDidBecomeActive`
	AttributeIosStateActive = "active"
	// The app is now `inactive`. Associated with UIKit notification `applicationWillResignActive`
	AttributeIosStateInactive = "inactive"
	// The app is now in the background. This value is associated with UIKit notification `applicationDidEnterBackground`
	AttributeIosStateBackground = "background"
	// The app is now in the foreground. This value is associated with UIKit notification `applicationWillEnterForeground`
	AttributeIosStateForeground = "foreground"
	// The app is about to terminate. Associated with UIKit notification `applicationWillTerminate`
	AttributeIosStateTerminate = "terminate"
)

// Namespace: jvm
const (

	// Name of the buffer pool.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "mapped",
	// "direct",
	//
	// Note: Pool names are generally obtained via [BufferPoolMXBean#getName()]
	//
	// [BufferPoolMXBean#getName()]: https://docs.oracle.com/en/java/javase/11/docs/api/java.management/java/lang/management/BufferPoolMXBean.html#getName()
	AttributeJvmBufferPoolName = "jvm.buffer.pool.name"

	// Name of the garbage collector action.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "end of minor GC",
	// "end of major GC",
	//
	// Note: Garbage collector action is generally obtained via [GarbageCollectionNotificationInfo#getGcAction()]
	//
	// [GarbageCollectionNotificationInfo#getGcAction()]: https://docs.oracle.com/en/java/javase/11/docs/api/jdk.management/com/sun/management/GarbageCollectionNotificationInfo.html#getGcAction()
	AttributeJvmGcAction = "jvm.gc.action"

	// Name of the garbage collector.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "G1 Young Generation",
	// "G1 Old Generation",
	//
	// Note: Garbage collector name is generally obtained via [GarbageCollectionNotificationInfo#getGcName()]
	//
	// [GarbageCollectionNotificationInfo#getGcName()]: https://docs.oracle.com/en/java/javase/11/docs/api/jdk.management/com/sun/management/GarbageCollectionNotificationInfo.html#getGcName()
	AttributeJvmGcName = "jvm.gc.name"

	// Name of the memory pool.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "G1 Old Gen",
	// "G1 Eden space",
	// "G1 Survivor Space",
	//
	// Note: Pool names are generally obtained via [MemoryPoolMXBean#getName()]
	//
	// [MemoryPoolMXBean#getName()]: https://docs.oracle.com/en/java/javase/11/docs/api/java.management/java/lang/management/MemoryPoolMXBean.html#getName()
	AttributeJvmMemoryPoolName = "jvm.memory.pool.name"

	// The type of memory.
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "heap",
	// "non_heap",
	AttributeJvmMemoryType = "jvm.memory.type"

	// Whether the thread is daemon or not.
	// Stability: Stable
	// Type: boolean
	//
	// Examples: undefined
	AttributeJvmThreadDaemon = "jvm.thread.daemon"

	// State of the thread.
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "runnable",
	// "blocked",
	AttributeJvmThreadState = "jvm.thread.state"
)

// Enum values for jvm.memory.type
const (
	// Heap memory
	AttributeJvmMemoryTypeHeap = "heap"
	// Non-heap memory
	AttributeJvmMemoryTypeNonHeap = "non_heap"
)

// Enum values for jvm.thread.state
const (
	// A thread that has not yet started is in this state
	AttributeJvmThreadStateNew = "new"
	// A thread executing in the Java virtual machine is in this state
	AttributeJvmThreadStateRunnable = "runnable"
	// A thread that is blocked waiting for a monitor lock is in this state
	AttributeJvmThreadStateBlocked = "blocked"
	// A thread that is waiting indefinitely for another thread to perform a particular action is in this state
	AttributeJvmThreadStateWaiting = "waiting"
	// A thread that is waiting for another thread to perform an action for up to a specified waiting time is in this state
	AttributeJvmThreadStateTimedWaiting = "timed_waiting"
	// A thread that has exited is in this state
	AttributeJvmThreadStateTerminated = "terminated"
)

// Namespace: k8s
const (

	// The name of the cluster.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry-cluster",
	AttributeK8SClusterName = "k8s.cluster.name"

	// A pseudo-ID for the cluster, set to the UID of the `kube-system` namespace.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "218fc5a9-a5f1-4b54-aa05-46717d0ab26d",
	//
	// Note: K8s doesn't have support for obtaining a cluster ID. If this is ever
	// added, we will recommend collecting the `k8s.cluster.uid` through the
	// official APIs. In the meantime, we are able to use the `uid` of the
	// `kube-system` namespace as a proxy for cluster ID. Read on for the
	// rationale.
	//
	// Every object created in a K8s cluster is assigned a distinct UID. The
	// `kube-system` namespace is used by Kubernetes itself and will exist
	// for the lifetime of the cluster. Using the `uid` of the `kube-system`
	// namespace is a reasonable proxy for the K8s ClusterID as it will only
	// change if the cluster is rebuilt. Furthermore, Kubernetes UIDs are
	// UUIDs as standardized by
	// [ISO/IEC 9834-8 and ITU-T X.667].
	// Which states:
	//
	// > If generated according to one of the mechanisms defined in Rec.
	// > ITU-T X.667 | ISO/IEC 9834-8, a UUID is either guaranteed to be
	// > different from all other UUIDs generated before 3603 A.D., or is
	// > extremely likely to be different (depending on the mechanism chosen).
	//
	// Therefore, UIDs between clusters should be extremely unlikely to
	// conflict
	//
	// [ISO/IEC 9834-8 and ITU-T X.667]: https://www.itu.int/ITU-T/studygroups/com17/oid.html
	AttributeK8SClusterUID = "k8s.cluster.uid"

	// The name of the Container from Pod specification, must be unique within a Pod. Container runtime usually uses different globally unique name (`container.name`).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "redis",
	AttributeK8SContainerName = "k8s.container.name"

	// Number of times the container was restarted. This attribute can be used to identify a particular container (running or stopped) within a container spec.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	AttributeK8SContainerRestartCount = "k8s.container.restart_count"

	// Last terminated reason of the Container.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Evicted",
	// "Error",
	AttributeK8SContainerStatusLastTerminatedReason = "k8s.container.status.last_terminated_reason"

	// The name of the CronJob.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SCronJobName = "k8s.cronjob.name"

	// The UID of the CronJob.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SCronJobUID = "k8s.cronjob.uid"

	// The name of the DaemonSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SDaemonSetName = "k8s.daemonset.name"

	// The UID of the DaemonSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SDaemonSetUID = "k8s.daemonset.uid"

	// The name of the Deployment.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SDeploymentName = "k8s.deployment.name"

	// The UID of the Deployment.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SDeploymentUID = "k8s.deployment.uid"

	// The name of the Job.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SJobName = "k8s.job.name"

	// The UID of the Job.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SJobUID = "k8s.job.uid"

	// The name of the namespace that the pod is running in.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "default",
	AttributeK8SNamespaceName = "k8s.namespace.name"

	// The name of the Node.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "node-1",
	AttributeK8SNodeName = "k8s.node.name"

	// The UID of the Node.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1eb3a0c6-0477-4080-a9cb-0cb7db65c6a2",
	AttributeK8SNodeUID = "k8s.node.uid"

	// The annotation key-value pairs placed on the Pod, the `<key>` being the annotation name, the value being the annotation value.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "k8s.pod.annotation.kubernetes.io/enforce-mountable-secrets=true",
	// "k8s.pod.annotation.mycompany.io/arch=x64",
	// "k8s.pod.annotation.data=",
	AttributeK8SPodAnnotation = "k8s.pod.annotation"

	// The label key-value pairs placed on the Pod, the `<key>` being the label name, the value being the label value.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "k8s.pod.label.app=my-app",
	// "k8s.pod.label.mycompany.io/arch=x64",
	// "k8s.pod.label.data=",
	AttributeK8SPodLabel = "k8s.pod.label"

	// Deprecated, use `k8s.pod.label` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `k8s.pod.label`.
	//
	// Examples:
	// "k8s.pod.label.app=my-app",
	AttributeK8SPodLabels = "k8s.pod.labels"

	// The name of the Pod.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry-pod-autoconf",
	AttributeK8SPodName = "k8s.pod.name"

	// The UID of the Pod.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SPodUID = "k8s.pod.uid"

	// The name of the ReplicaSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SReplicaSetName = "k8s.replicaset.name"

	// The UID of the ReplicaSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SReplicaSetUID = "k8s.replicaset.uid"

	// The name of the StatefulSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	AttributeK8SStatefulSetName = "k8s.statefulset.name"

	// The UID of the StatefulSet.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "275ecb36-5aa8-4c2a-9c47-d8bb681b9aff",
	AttributeK8SStatefulSetUID = "k8s.statefulset.uid"

	// The name of the K8s volume.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "volume0",
	AttributeK8SVolumeName = "k8s.volume.name"

	// The type of the K8s volume.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "emptyDir",
	// "persistentVolumeClaim",
	AttributeK8SVolumeType = "k8s.volume.type"
)

// Enum values for k8s.volume.type
const (
	// A [persistentVolumeClaim] volume
	//
	// [persistentVolumeClaim]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#persistentvolumeclaim
	AttributeK8SVolumeTypePersistentVolumeClaim = "persistentVolumeClaim"
	// A [configMap] volume
	//
	// [configMap]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#configmap
	AttributeK8SVolumeTypeConfigMap = "configMap"
	// A [downwardAPI] volume
	//
	// [downwardAPI]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#downwardapi
	AttributeK8SVolumeTypeDownwardAPI = "downwardAPI"
	// An [emptyDir] volume
	//
	// [emptyDir]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#emptydir
	AttributeK8SVolumeTypeEmptyDir = "emptyDir"
	// A [secret] volume
	//
	// [secret]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#secret
	AttributeK8SVolumeTypeSecret = "secret"
	// A [local] volume
	//
	// [local]: https://v1-29.docs.kubernetes.io/docs/concepts/storage/volumes/#local
	AttributeK8SVolumeTypeLocal = "local"
)

// Namespace: linux
const (

	// The Linux Slab memory state
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "reclaimable",
	// "unreclaimable",
	AttributeLinuxMemorySlabState = "linux.memory.slab.state"
)

// Enum values for linux.memory.slab.state
const (
	// none
	AttributeLinuxMemorySlabStateReclaimable = "reclaimable"
	// none
	AttributeLinuxMemorySlabStateUnreclaimable = "unreclaimable"
)

// Namespace: log
const (

	// The basename of the file.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "audit.log",
	AttributeLogFileName = "log.file.name"

	// The basename of the file, with symlinks resolved.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "uuid.log",
	AttributeLogFileNameResolved = "log.file.name_resolved"

	// The full path to the file.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/var/log/mysql/audit.log",
	AttributeLogFilePath = "log.file.path"

	// The full path to the file, with symlinks resolved.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/var/lib/docker/uuid.log",
	AttributeLogFilePathResolved = "log.file.path_resolved"

	// The stream associated with the log. See below for a list of well-known values.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeLogIostream = "log.iostream"

	// The complete original Log Record.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "77 <86>1 2015-08-06T21:58:59.694Z 192.168.2.133 inactive - - - Something happened",
	// "[INFO] 8/3/24 12:34:56 Something happened",
	//
	// Note: This value MAY be added when processing a Log Record which was originally transmitted as a string or equivalent data type AND the Body field of the Log Record does not contain the same value. (e.g. a syslog or a log record read from a file.)
	AttributeLogRecordOriginal = "log.record.original"

	// A unique identifier for the Log Record.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	//
	// Note: If an id is provided, other log records with the same id will be considered duplicates and can be removed safely. This means, that two distinguishable log records MUST have different values.
	// The id MAY be an [Universally Unique Lexicographically Sortable Identifier (ULID)], but other identifiers (e.g. UUID) may be used as needed
	//
	// [Universally Unique Lexicographically Sortable Identifier (ULID)]: https://github.com/ulid/spec
	AttributeLogRecordUID = "log.record.uid"
)

// Enum values for log.iostream
const (
	// Logs from stdout stream
	AttributeLogIostreamStdout = "stdout"
	// Events from stderr stream
	AttributeLogIostreamStderr = "stderr"
)

// Namespace: message
const (

	// Deprecated, use `rpc.message.compressed_size` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `rpc.message.compressed_size`.
	//
	// Examples: undefined
	AttributeMessageCompressedSize = "message.compressed_size"

	// Deprecated, use `rpc.message.id` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `rpc.message.id`.
	//
	// Examples: undefined
	AttributeMessageID = "message.id"

	// Deprecated, use `rpc.message.type` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `rpc.message.type`.
	//
	// Examples: undefined
	AttributeMessageType = "message.type"

	// Deprecated, use `rpc.message.uncompressed_size` instead.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `rpc.message.uncompressed_size`.
	//
	// Examples: undefined
	AttributeMessageUncompressedSize = "message.uncompressed_size"
)

// Enum values for message.type
const (
	// none
	AttributeMessageTypeSent = "SENT"
	// none
	AttributeMessageTypeReceived = "RECEIVED"
)

// Namespace: messaging
const (

	// The number of messages sent, received, or processed in the scope of the batching operation.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 0,
	// 1,
	// 2,
	//
	// Note: Instrumentations SHOULD NOT set `messaging.batch.message_count` on spans that operate with a single message. When a messaging client library supports both batch and single-message API for the same operation, instrumentations SHOULD use `messaging.batch.message_count` for batching APIs and SHOULD NOT use it for single-message APIs
	AttributeMessagingBatchMessageCount = "messaging.batch.message_count"

	// A unique identifier for the client that consumes or produces a message.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "client-5",
	// "myhost@8742@s8083jm",
	AttributeMessagingClientID = "messaging.client.id"

	// The name of the consumer group with which a consumer is associated.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "my-group",
	// "indexer",
	//
	// Note: Semantic conventions for individual messaging systems SHOULD document whether `messaging.consumer.group.name` is applicable and what it means in the context of that system
	AttributeMessagingConsumerGroupName = "messaging.consumer.group.name"

	// A boolean that is true if the message destination is anonymous (could be unnamed or have auto-generated name).
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeMessagingDestinationAnonymous = "messaging.destination.anonymous"

	// The message destination name
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "MyQueue",
	// "MyTopic",
	//
	// Note: Destination name SHOULD uniquely identify a specific queue, topic or other entity within the broker. If
	// the broker doesn't have such notion, the destination name SHOULD uniquely identify the broker
	AttributeMessagingDestinationName = "messaging.destination.name"

	// The identifier of the partition messages are sent to or received from, unique within the `messaging.destination.name`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "1"
	AttributeMessagingDestinationPartitionID = "messaging.destination.partition.id"

	// The name of the destination subscription from which a message is consumed.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "subscription-a",
	//
	// Note: Semantic conventions for individual messaging systems SHOULD document whether `messaging.destination.subscription.name` is applicable and what it means in the context of that system
	AttributeMessagingDestinationSubscriptionName = "messaging.destination.subscription.name"

	// Low cardinality representation of the messaging destination name
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/customers/{customerId}",
	//
	// Note: Destination names could be constructed from templates. An example would be a destination name involving a user name or product id. Although the destination name in this case is of high cardinality, the underlying template is of low cardinality and can be effectively used for grouping and aggregation
	AttributeMessagingDestinationTemplate = "messaging.destination.template"

	// A boolean that is true if the message destination is temporary and might not exist anymore after messages are processed.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeMessagingDestinationTemporary = "messaging.destination.temporary"

	// Deprecated, no replacement at this time.
	// Stability: Experimental
	// Type: boolean
	// Deprecated: No replacement at this time.
	//
	// Examples: undefined
	AttributeMessagingDestinationPublishAnonymous = "messaging.destination_publish.anonymous"

	// Deprecated, no replacement at this time.
	// Stability: Experimental
	// Type: string
	// Deprecated: No replacement at this time.
	//
	// Examples:
	// "MyQueue",
	// "MyTopic",
	AttributeMessagingDestinationPublishName = "messaging.destination_publish.name"

	// Deprecated, use `messaging.consumer.group.name` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `messaging.consumer.group.name`.
	//
	// Examples: "$Default"
	AttributeMessagingEventhubsConsumerGroup = "messaging.eventhubs.consumer.group"

	// The UTC epoch seconds at which the message has been accepted and stored in the entity.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingEventhubsMessageEnqueuedTime = "messaging.eventhubs.message.enqueued_time"

	// The ack deadline in seconds set for the modify ack deadline request.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingGCPPubsubMessageAckDeadline = "messaging.gcp_pubsub.message.ack_deadline"

	// The ack id for a given message.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "ack_id"
	AttributeMessagingGCPPubsubMessageAckID = "messaging.gcp_pubsub.message.ack_id"

	// The delivery attempt for a given message.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingGCPPubsubMessageDeliveryAttempt = "messaging.gcp_pubsub.message.delivery_attempt"

	// The ordering key for a given message. If the attribute is not present, the message does not have an ordering key.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "ordering_key"
	AttributeMessagingGCPPubsubMessageOrderingKey = "messaging.gcp_pubsub.message.ordering_key"

	// Deprecated, use `messaging.consumer.group.name` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `messaging.consumer.group.name`.
	//
	// Examples: "my-group"
	AttributeMessagingKafkaConsumerGroup = "messaging.kafka.consumer.group"

	// Deprecated, use `messaging.destination.partition.id` instead.
	//
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `messaging.destination.partition.id`
	AttributeMessagingKafkaDestinationPartition = "messaging.kafka.destination.partition"

	// Message keys in Kafka are used for grouping alike messages to ensure they're processed on the same partition. They differ from `messaging.message.id` in that they're not unique. If the key is `null`, the attribute MUST NOT be set.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "myKey"
	// Note: If the key type is not string, it's string representation has to be supplied for the attribute. If the key has no unambiguous, canonical string form, don't include its value
	AttributeMessagingKafkaMessageKey = "messaging.kafka.message.key"

	// Deprecated, use `messaging.kafka.offset` instead.
	//
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `messaging.kafka.offset`
	AttributeMessagingKafkaMessageOffset = "messaging.kafka.message.offset"

	// A boolean that is true if the message is a tombstone.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeMessagingKafkaMessageTombstone = "messaging.kafka.message.tombstone"

	// The offset of a record in the corresponding Kafka partition.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingKafkaOffset = "messaging.kafka.offset"

	// The size of the message body in bytes.
	//
	// Stability: Experimental
	// Type: int
	//
	// Note: This can refer to both the compressed or uncompressed body size. If both sizes are known, the uncompressed
	// body size should be used
	AttributeMessagingMessageBodySize = "messaging.message.body.size"

	// The conversation ID identifying the conversation to which the message belongs, represented as a string. Sometimes called "Correlation ID".
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "MyConversationId"
	AttributeMessagingMessageConversationID = "messaging.message.conversation_id"

	// The size of the message body and metadata in bytes.
	//
	// Stability: Experimental
	// Type: int
	//
	// Note: This can refer to both the compressed or uncompressed size. If both sizes are known, the uncompressed
	// size should be used
	AttributeMessagingMessageEnvelopeSize = "messaging.message.envelope.size"

	// A value used by the messaging system as an identifier for the message, represented as a string.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "452a7c7c7c7048c2f887f61572b18fc2"
	AttributeMessagingMessageID = "messaging.message.id"

	// Deprecated, use `messaging.operation.type` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `messaging.operation.type`.
	//
	// Examples:
	// "publish",
	// "create",
	// "process",
	AttributeMessagingOperation = "messaging.operation"

	// The system-specific name of the messaging operation.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "ack",
	// "nack",
	// "send",
	AttributeMessagingOperationName = "messaging.operation.name"

	// A string identifying the type of the messaging operation.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: If a custom value is used, it MUST be of low cardinality
	AttributeMessagingOperationType = "messaging.operation.type"

	// RabbitMQ message routing key.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "myKey"
	AttributeMessagingRabbitmqDestinationRoutingKey = "messaging.rabbitmq.destination.routing_key"

	// RabbitMQ message delivery tag
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingRabbitmqMessageDeliveryTag = "messaging.rabbitmq.message.delivery_tag"

	// Deprecated, use `messaging.consumer.group.name` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `messaging.consumer.group.name` on the consumer spans. No replacement for producer spans.
	//
	// Examples: "myConsumerGroup"
	AttributeMessagingRocketmqClientGroup = "messaging.rocketmq.client_group"

	// Model of message consumption. This only applies to consumer spans.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeMessagingRocketmqConsumptionModel = "messaging.rocketmq.consumption_model"

	// The delay time level for delay message, which determines the message delay time.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingRocketmqMessageDelayTimeLevel = "messaging.rocketmq.message.delay_time_level"

	// The timestamp in milliseconds that the delay message is expected to be delivered to consumer.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingRocketmqMessageDeliveryTimestamp = "messaging.rocketmq.message.delivery_timestamp"

	// It is essential for FIFO message. Messages that belong to the same message group are always processed one by one within the same consumer group.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "myMessageGroup"
	AttributeMessagingRocketmqMessageGroup = "messaging.rocketmq.message.group"

	// Key(s) of message, another way to mark message besides message id.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "keyA",
	// "keyB",
	// ],
	AttributeMessagingRocketmqMessageKeys = "messaging.rocketmq.message.keys"

	// The secondary classifier of message besides topic.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "tagA"
	AttributeMessagingRocketmqMessageTag = "messaging.rocketmq.message.tag"

	// Type of message.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeMessagingRocketmqMessageType = "messaging.rocketmq.message.type"

	// Namespace of RocketMQ resources, resources in different namespaces are individual.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "myNamespace"
	AttributeMessagingRocketmqNamespace = "messaging.rocketmq.namespace"

	// Deprecated, use `messaging.servicebus.destination.subscription_name` instead.
	//
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `messaging.servicebus.destination.subscription_name`.
	//
	// Examples: "subscription-a"
	AttributeMessagingServicebusDestinationSubscriptionName = "messaging.servicebus.destination.subscription_name"

	// Describes the [settlement type].
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [settlement type]: https://learn.microsoft.com/azure/service-bus-messaging/message-transfers-locks-settlement#peeklock
	AttributeMessagingServicebusDispositionStatus = "messaging.servicebus.disposition_status"

	// Number of deliveries that have been attempted for this message.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingServicebusMessageDeliveryCount = "messaging.servicebus.message.delivery_count"

	// The UTC epoch seconds at which the message has been accepted and stored in the entity.
	//
	// Stability: Experimental
	// Type: int
	AttributeMessagingServicebusMessageEnqueuedTime = "messaging.servicebus.message.enqueued_time"

	// The messaging system as identified by the client instrumentation.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: The actual messaging system may differ from the one known by the client. For example, when using Kafka client libraries to communicate with Azure Event Hubs, the `messaging.system` is set to `kafka` based on the instrumentation's best knowledge
	AttributeMessagingSystem = "messaging.system"
)

// Enum values for messaging.operation.type
const (
	// One or more messages are provided for publishing to an intermediary. If a single message is published, the context of the "Publish" span can be used as the creation context and no "Create" span needs to be created
	AttributeMessagingOperationTypePublish = "publish"
	// A message is created. "Create" spans always refer to a single message and are used to provide a unique creation context for messages in batch publishing scenarios
	AttributeMessagingOperationTypeCreate = "create"
	// One or more messages are requested by a consumer. This operation refers to pull-based scenarios, where consumers explicitly call methods of messaging SDKs to receive messages
	AttributeMessagingOperationTypeReceive = "receive"
	// One or more messages are processed by a consumer
	AttributeMessagingOperationTypeProcess = "process"
	// One or more messages are settled
	AttributeMessagingOperationTypeSettle = "settle"
	// Deprecated. Use `process` instead
	AttributeMessagingOperationTypeDeliver = "deliver"
)

// Enum values for messaging.rocketmq.consumption_model
const (
	// Clustering consumption model
	AttributeMessagingRocketmqConsumptionModelClustering = "clustering"
	// Broadcasting consumption model
	AttributeMessagingRocketmqConsumptionModelBroadcasting = "broadcasting"
)

// Enum values for messaging.rocketmq.message.type
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

// Enum values for messaging.servicebus.disposition_status
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

// Enum values for messaging.system
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
	// Apache Pulsar
	AttributeMessagingSystemPulsar = "pulsar"
)

// Namespace: net
const (

	// Deprecated, use `network.local.address`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.local.address`.
	//
	// Examples: "192.168.0.1"
	AttributeNetHostIP = "net.host.ip"

	// Deprecated, use `server.address`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `server.address`.
	//
	// Examples:
	// "example.com",
	AttributeNetHostName = "net.host.name"

	// Deprecated, use `server.port`.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `server.port`.
	//
	// Examples:
	// 8080,
	AttributeNetHostPort = "net.host.port"

	// Deprecated, use `network.peer.address`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.peer.address`.
	//
	// Examples: "127.0.0.1"
	AttributeNetPeerIP = "net.peer.ip"

	// Deprecated, use `server.address` on client spans and `client.address` on server spans.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `server.address` on client spans and `client.address` on server spans.
	//
	// Examples:
	// "example.com",
	AttributeNetPeerName = "net.peer.name"

	// Deprecated, use `server.port` on client spans and `client.port` on server spans.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `server.port` on client spans and `client.port` on server spans.
	//
	// Examples:
	// 8080,
	AttributeNetPeerPort = "net.peer.port"

	// Deprecated, use `network.protocol.name`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.protocol.name`.
	//
	// Examples:
	// "amqp",
	// "http",
	// "mqtt",
	AttributeNetProtocolName = "net.protocol.name"

	// Deprecated, use `network.protocol.version`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.protocol.version`.
	//
	// Examples: "3.1.1"
	AttributeNetProtocolVersion = "net.protocol.version"

	// Deprecated, use `network.transport` and `network.type`.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Split to `network.transport` and `network.type`.
	//
	// Examples: undefined
	AttributeNetSockFamily = "net.sock.family"

	// Deprecated, use `network.local.address`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.local.address`.
	//
	// Examples:
	// "/var/my.sock",
	AttributeNetSockHostAddr = "net.sock.host.addr"

	// Deprecated, use `network.local.port`.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `network.local.port`.
	//
	// Examples:
	// 8080,
	AttributeNetSockHostPort = "net.sock.host.port"

	// Deprecated, use `network.peer.address`.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `network.peer.address`.
	//
	// Examples:
	// "192.168.0.1",
	AttributeNetSockPeerAddr = "net.sock.peer.addr"

	// Deprecated, no replacement at this time.
	// Stability: Experimental
	// Type: string
	// Deprecated: Removed.
	//
	// Examples:
	// "/var/my.sock",
	AttributeNetSockPeerName = "net.sock.peer.name"

	// Deprecated, use `network.peer.port`.
	// Stability: Experimental
	// Type: int
	// Deprecated: Replaced by `network.peer.port`.
	//
	// Examples:
	// 65531,
	AttributeNetSockPeerPort = "net.sock.peer.port"

	// Deprecated, use `network.transport`.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `network.transport`.
	//
	// Examples: undefined
	AttributeNetTransport = "net.transport"
)

// Enum values for net.sock.family
const (
	// IPv4 address
	AttributeNetSockFamilyInet = "inet"
	// IPv6 address
	AttributeNetSockFamilyInet6 = "inet6"
	// Unix domain socket path
	AttributeNetSockFamilyUnix = "unix"
)

// Enum values for net.transport
const (
	// none
	AttributeNetTransportIPTCP = "ip_tcp"
	// none
	AttributeNetTransportIPUDP = "ip_udp"
	// Named or anonymous pipe
	AttributeNetTransportPipe = "pipe"
	// In-process communication
	AttributeNetTransportInProc = "inproc"
	// Something else (non IP-based)
	AttributeNetTransportOther = "other"
)

// Namespace: network
const (

	// The ISO 3166-1 alpha-2 2-character country code associated with the mobile carrier network.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "DE"
	AttributeNetworkCarrierIcc = "network.carrier.icc"

	// The mobile carrier country code.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "310"
	AttributeNetworkCarrierMcc = "network.carrier.mcc"

	// The mobile carrier network code.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "001"
	AttributeNetworkCarrierMnc = "network.carrier.mnc"

	// The name of the mobile carrier.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "sprint"
	AttributeNetworkCarrierName = "network.carrier.name"

	// This describes more details regarding the connection.type. It may be the type of cell technology connection, but it could be used for describing details about a wifi connection.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: "LTE"
	AttributeNetworkConnectionSubtype = "network.connection.subtype"

	// The internet connection type.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: "wifi"
	AttributeNetworkConnectionType = "network.connection.type"

	// The network IO operation direction.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "transmit",
	AttributeNetworkIoDirection = "network.io.direction"

	// Local address of the network connection - IP address or Unix domain socket name.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "10.1.2.80",
	// "/tmp/my.sock",
	AttributeNetworkLocalAddress = "network.local.address"

	// Local port number of the network connection.
	// Stability: Stable
	// Type: int
	//
	// Examples:
	// 65123,
	AttributeNetworkLocalPort = "network.local.port"

	// Peer address of the network connection - IP address or Unix domain socket name.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "10.1.2.80",
	// "/tmp/my.sock",
	AttributeNetworkPeerAddress = "network.peer.address"

	// Peer port number of the network connection.
	// Stability: Stable
	// Type: int
	//
	// Examples:
	// 65123,
	AttributeNetworkPeerPort = "network.peer.port"

	// [OSI application layer] or non-OSI equivalent.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "amqp",
	// "http",
	// "mqtt",
	//
	// Note: The value SHOULD be normalized to lowercase
	//
	// [OSI application layer]: https://osi-model.com/application-layer/
	AttributeNetworkProtocolName = "network.protocol.name"

	// The actual version of the protocol used for network communication.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "1.1",
	// "2",
	//
	// Note: If protocol version is subject to negotiation (for example using [ALPN]), this attribute SHOULD be set to the negotiated version. If the actual protocol version is not known, this attribute SHOULD NOT be set
	//
	// [ALPN]: https://www.rfc-editor.org/rfc/rfc7301.html
	AttributeNetworkProtocolVersion = "network.protocol.version"

	// [OSI transport layer] or [inter-process communication method].
	//
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "tcp",
	// "udp",
	//
	// Note: The value SHOULD be normalized to lowercase.
	//
	// Consider always setting the transport when setting a port number, since
	// a port number is ambiguous without knowing the transport. For example
	// different processes could be listening on TCP port 12345 and UDP port 12345
	//
	// [OSI transport layer]: https://osi-model.com/transport-layer/
	// [inter-process communication method]: https://wikipedia.org/wiki/Inter-process_communication
	AttributeNetworkTransport = "network.transport"

	// [OSI network layer] or non-OSI equivalent.
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "ipv4",
	// "ipv6",
	//
	// Note: The value SHOULD be normalized to lowercase
	//
	// [OSI network layer]: https://osi-model.com/network-layer/
	AttributeNetworkType = "network.type"
)

// Enum values for network.connection.subtype
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

// Enum values for network.connection.type
const (
	// none
	AttributeNetworkConnectionTypeWifi = "wifi"
	// none
	AttributeNetworkConnectionTypeWired = "wired"
	// none
	AttributeNetworkConnectionTypeCell = "cell"
	// none
	AttributeNetworkConnectionTypeUnavailable = "unavailable"
	// none
	AttributeNetworkConnectionTypeUnknown = "unknown"
)

// Enum values for network.io.direction
const (
	// none
	AttributeNetworkIoDirectionTransmit = "transmit"
	// none
	AttributeNetworkIoDirectionReceive = "receive"
)

// Enum values for network.transport
const (
	// TCP
	AttributeNetworkTransportTCP = "tcp"
	// UDP
	AttributeNetworkTransportUDP = "udp"
	// Named or anonymous pipe
	AttributeNetworkTransportPipe = "pipe"
	// Unix domain socket
	AttributeNetworkTransportUnix = "unix"
	// QUIC
	AttributeNetworkTransportQUIC = "quic"
)

// Enum values for network.type
const (
	// IPv4
	AttributeNetworkTypeIpv4 = "ipv4"
	// IPv6
	AttributeNetworkTypeIpv6 = "ipv6"
)

// Namespace: nodejs
const (

	// The state of event loop time.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeNodejsEventloopState = "nodejs.eventloop.state"
)

// Enum values for nodejs.eventloop.state
const (
	// Active time
	AttributeNodejsEventloopStateActive = "active"
	// Idle time
	AttributeNodejsEventloopStateIdle = "idle"
)

// Namespace: oci
const (

	// The digest of the OCI image manifest. For container images specifically is the digest by which the container image is known.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "sha256:e4ca62c0d62f3e886e684806dfe9d4e0cda60d54986898173c1083856cfda0f4",
	//
	// Note: Follows [OCI Image Manifest Specification], and specifically the [Digest property].
	// An example can be found in [Example Image Manifest]
	//
	// [OCI Image Manifest Specification]: https://github.com/opencontainers/image-spec/blob/main/manifest.md
	// [Digest property]: https://github.com/opencontainers/image-spec/blob/main/descriptor.md#digests
	// [Example Image Manifest]: https://docs.docker.com/registry/spec/manifest-v2-2/#example-image-manifest
	AttributeOciManifestDigest = "oci.manifest.digest"
)

// Namespace: opentracing
const (

	// Parent-child Reference type
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: The causal relationship between a child Span and a parent Span
	AttributeOpentracingRefType = "opentracing.ref_type"
)

// Enum values for opentracing.ref_type
const (
	// The parent Span depends on the child Span in some capacity
	AttributeOpentracingRefTypeChildOf = "child_of"
	// The parent Span doesn't depend in any way on the result of the child Span
	AttributeOpentracingRefTypeFollowsFrom = "follows_from"
)

// Namespace: os
const (

	// Unique identifier for a particular build or compilation of the operating system.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "TQ3C.230805.001.B2",
	// "20E247",
	// "22621",
	AttributeOSBuildID = "os.build_id"

	// Human readable (not intended to be parsed) OS version information, like e.g. reported by `ver` or `lsb_release -a` commands.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Microsoft Windows [Version 10.0.18363.778]",
	// "Ubuntu 18.04.1 LTS",
	AttributeOSDescription = "os.description"

	// Human readable operating system name.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "iOS",
	// "Android",
	// "Ubuntu",
	AttributeOSName = "os.name"

	// The operating system type.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeOSType = "os.type"

	// The version string of the operating system as defined in [Version Attributes].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "14.2.1",
	// "18.04.1",
	//
	// [Version Attributes]: /docs/resource/README.md#version-attributes
	AttributeOSVersion = "os.version"
)

// Enum values for os.type
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

// Namespace: otel
const (

	// Deprecated. Use the `otel.scope.name` attribute
	// Stability: Experimental
	// Type: string
	// Deprecated: Use the `otel.scope.name` attribute.
	//
	// Examples:
	// "io.opentelemetry.contrib.mongodb",
	AttributeOTelLibraryName = "otel.library.name"

	// Deprecated. Use the `otel.scope.version` attribute.
	// Stability: Experimental
	// Type: string
	// Deprecated: Use the `otel.scope.version` attribute.
	//
	// Examples:
	// "1.0.0",
	AttributeOTelLibraryVersion = "otel.library.version"

	// The name of the instrumentation scope - (`InstrumentationScope.Name` in OTLP).
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "io.opentelemetry.contrib.mongodb",
	AttributeOTelScopeName = "otel.scope.name"

	// The version of the instrumentation scope - (`InstrumentationScope.Version` in OTLP).
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "1.0.0",
	AttributeOTelScopeVersion = "otel.scope.version"

	// Name of the code, either "OK" or "ERROR". MUST NOT be set if the status code is UNSET.
	// Stability: Stable
	// Type: Enum
	//
	// Examples: undefined
	AttributeOTelStatusCode = "otel.status_code"

	// Description of the Status if it has a value, otherwise not set.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "resource not found",
	AttributeOTelStatusDescription = "otel.status_description"
)

// Enum values for otel.status_code
const (
	// The operation has been validated by an Application developer or Operator to have completed successfully
	AttributeOTelStatusCodeOk = "OK"
	// The operation contains an error
	AttributeOTelStatusCodeError = "ERROR"
)

// Namespace: other
const (

	// Deprecated, use `db.client.connection.state` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `db.client.connection.state`.
	//
	// Examples:
	// "idle",
	AttributeState = "state"
)

// Enum values for state
const (
	// none
	AttributeStateIdle = "idle"
	// none
	AttributeStateUsed = "used"
)

// Namespace: peer
const (

	// The [`service.name`] of the remote service. SHOULD be equal to the actual `service.name` resource attribute of the remote service if any.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "AuthTokenCache"
	//
	// [`service.name`]: /docs/resource/README.md#service
	AttributePeerService = "peer.service"
)

// Namespace: pool
const (

	// Deprecated, use `db.client.connection.pool.name` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `db.client.connection.pool.name`.
	//
	// Examples:
	// "myDataSource",
	AttributePoolName = "pool.name"
)

// Namespace: process
const (

	// Length of the process.command_args array
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 4,
	//
	// Note: This field can be useful for querying or performing bucket analysis on how many arguments were provided to start a process. More arguments may be an indication of suspicious activity
	AttributeProcessArgsCount = "process.args_count"

	// The command used to launch the process (i.e. the command name). On Linux based systems, can be set to the zeroth string in `proc/[pid]/cmdline`. On Windows, can be set to the first parameter extracted from `GetCommandLineW`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "cmd/otelcol",
	AttributeProcessCommand = "process.command"

	// All the command arguments (including the command/executable itself) as received by the process. On Linux-based systems (and some other Unixoid systems supporting procfs), can be set according to the list of null-delimited strings extracted from `proc/[pid]/cmdline`. For libc-based executables, this would be the full argv vector passed to `main`.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "cmd/otecol",
	// "--config=config.yaml",
	// ],
	AttributeProcessCommandArgs = "process.command_args"

	// The full command used to launch the process as a single string representing the full command. On Windows, can be set to the result of `GetCommandLineW`. Do not set this if you have to assemble it just for monitoring; use `process.command_args` instead.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "C:\cmd\otecol --config="my directory\config.yaml"",
	AttributeProcessCommandLine = "process.command_line"

	// Specifies whether the context switches for this data point were voluntary or involuntary.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeProcessContextSwitchType = "process.context_switch_type"

	// Deprecated, use `cpu.mode` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `cpu.mode`
	//
	// Examples: undefined
	AttributeProcessCPUState = "process.cpu.state"

	// The date and time the process was created, in ISO 8601 format.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2023-11-21T09:25:34.853Z",
	AttributeProcessCreationTime = "process.creation.time"

	// The name of the process executable. On Linux based systems, can be set to the `Name` in `proc/[pid]/status`. On Windows, can be set to the base name of `GetProcessImageFileNameW`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "otelcol",
	AttributeProcessExecutableName = "process.executable.name"

	// The full path to the process executable. On Linux based systems, can be set to the target of `proc/[pid]/exe`. On Windows, can be set to the result of `GetProcessImageFileNameW`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/usr/bin/cmd/otelcol",
	AttributeProcessExecutablePath = "process.executable.path"

	// The exit code of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 127,
	AttributeProcessExitCode = "process.exit.code"

	// The date and time the process exited, in ISO 8601 format.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2023-11-21T09:26:12.315Z",
	AttributeProcessExitTime = "process.exit.time"

	// The PID of the process's group leader. This is also the process group ID (PGID) of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 23,
	AttributeProcessGroupLeaderPID = "process.group_leader.pid"

	// Whether the process is connected to an interactive shell.
	//
	// Stability: Experimental
	// Type: boolean
	//
	// Examples: undefined
	AttributeProcessInteractive = "process.interactive"

	// The username of the user that owns the process.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "root",
	AttributeProcessOwner = "process.owner"

	// The type of page fault for this data point. Type `major` is for major/hard page faults, and `minor` is for minor/soft page faults.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeProcessPagingFaultType = "process.paging.fault_type"

	// Parent Process identifier (PPID).
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 111,
	AttributeProcessParentPID = "process.parent_pid"

	// Process identifier (PID).
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1234,
	AttributeProcessPID = "process.pid"

	// The real user ID (RUID) of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1000,
	AttributeProcessRealUserID = "process.real_user.id"

	// The username of the real user of the process.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "operator",
	AttributeProcessRealUserName = "process.real_user.name"

	// An additional description about the runtime of the process, for example a specific vendor customization of the runtime environment.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "Eclipse OpenJ9 Eclipse OpenJ9 VM openj9-0.21.0"
	AttributeProcessRuntimeDescription = "process.runtime.description"

	// The name of the runtime of this process.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "OpenJDK Runtime Environment",
	AttributeProcessRuntimeName = "process.runtime.name"

	// The version of the runtime of this process, as returned by the runtime without modification.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "14.0.2"
	AttributeProcessRuntimeVersion = "process.runtime.version"

	// The saved user ID (SUID) of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1002,
	AttributeProcessSavedUserID = "process.saved_user.id"

	// The username of the saved user.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "operator",
	AttributeProcessSavedUserName = "process.saved_user.name"

	// The PID of the process's session leader. This is also the session ID (SID) of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 14,
	AttributeProcessSessionLeaderPID = "process.session_leader.pid"

	// Process title (proctitle)
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "cat /etc/hostname",
	// "xfce4-session",
	// "bash",
	//
	// Note: In many Unix-like systems, process title (proctitle), is the string that represents the name or command line of a running process, displayed by system monitoring tools like ps, top, and htop
	AttributeProcessTitle = "process.title"

	// The effective user ID (EUID) of the process.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1001,
	AttributeProcessUserID = "process.user.id"

	// The username of the effective user of the process.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "root",
	AttributeProcessUserName = "process.user.name"

	// Virtual process identifier.
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 12,
	//
	// Note: The process ID within a PID namespace. This is not necessarily unique across all processes on the host but it is unique within the process namespace that the process exists within
	AttributeProcessVpid = "process.vpid"

	// The working directory of the process.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/root",
	AttributeProcessWorkingDirectory = "process.working_directory"
)

// Enum values for process.context_switch_type
const (
	// none
	AttributeProcessContextSwitchTypeVoluntary = "voluntary"
	// none
	AttributeProcessContextSwitchTypeInvoluntary = "involuntary"
)

// Enum values for process.cpu.state
const (
	// none
	AttributeProcessCPUStateSystem = "system"
	// none
	AttributeProcessCPUStateUser = "user"
	// none
	AttributeProcessCPUStateWait = "wait"
)

// Enum values for process.paging.fault_type
const (
	// none
	AttributeProcessPagingFaultTypeMajor = "major"
	// none
	AttributeProcessPagingFaultTypeMinor = "minor"
)

// Namespace: profile
const (

	// Describes the interpreter or compiler of a single frame.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "cpython",
	AttributeProfileFrameType = "profile.frame.type"
)

// Enum values for profile.frame.type
const (
	// [.NET]
	//
	// [.NET]: https://wikipedia.org/wiki/.NET
	AttributeProfileFrameTypeDotnet = "dotnet"
	// [JVM]
	//
	// [JVM]: https://wikipedia.org/wiki/Java_virtual_machine
	AttributeProfileFrameTypeJvm = "jvm"
	// [Kernel]
	//
	// [Kernel]: https://wikipedia.org/wiki/Kernel_(operating_system)
	AttributeProfileFrameTypeKernel = "kernel"
	// [C], [C++], [Go], [Rust]
	//
	// [C]: https://wikipedia.org/wiki/C_(programming_language)
	// [C++]: https://wikipedia.org/wiki/C%2B%2B
	// [Go]: https://wikipedia.org/wiki/Go_(programming_language)
	// [Rust]: https://wikipedia.org/wiki/Rust_(programming_language)
	AttributeProfileFrameTypeNative = "native"
	// [Perl]
	//
	// [Perl]: https://wikipedia.org/wiki/Perl
	AttributeProfileFrameTypePerl = "perl"
	// [PHP]
	//
	// [PHP]: https://wikipedia.org/wiki/PHP
	AttributeProfileFrameTypePHP = "php"
	// [Python]
	//
	// [Python]: https://wikipedia.org/wiki/Python_(programming_language)
	AttributeProfileFrameTypeCpython = "cpython"
	// [Ruby]
	//
	// [Ruby]: https://wikipedia.org/wiki/Ruby_(programming_language)
	AttributeProfileFrameTypeRuby = "ruby"
	// [V8JS]
	//
	// [V8JS]: https://wikipedia.org/wiki/V8_(JavaScript_engine)
	AttributeProfileFrameTypeV8js = "v8js"
)

// Namespace: rpc
const (

	// The [error codes] of the Connect request. Error codes are always string values.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [error codes]: https://connect.build/docs/protocol/#error-codes
	AttributeRPCConnectRPCErrorCode = "rpc.connect_rpc.error_code"

	// Connect request metadata, `<key>` being the normalized Connect Metadata key (lowercase), the value being the metadata values.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// "rpc.request.metadata.my-custom-metadata-attribute=["1.2.3.4", "1.2.3.5"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which metadata values are to be captured. Including all request metadata values can be a security risk - explicit configuration helps avoid leaking sensitive information
	AttributeRPCConnectRPCRequestMetadata = "rpc.connect_rpc.request.metadata"

	// Connect response metadata, `<key>` being the normalized Connect Metadata key (lowercase), the value being the metadata values.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// "rpc.response.metadata.my-custom-metadata-attribute=["attribute_value"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which metadata values are to be captured. Including all response metadata values can be a security risk - explicit configuration helps avoid leaking sensitive information
	AttributeRPCConnectRPCResponseMetadata = "rpc.connect_rpc.response.metadata"

	// gRPC request metadata, `<key>` being the normalized gRPC Metadata key (lowercase), the value being the metadata values.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// "rpc.grpc.request.metadata.my-custom-metadata-attribute=["1.2.3.4", "1.2.3.5"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which metadata values are to be captured. Including all request metadata values can be a security risk - explicit configuration helps avoid leaking sensitive information
	AttributeRPCGRPCRequestMetadata = "rpc.grpc.request.metadata"

	// gRPC response metadata, `<key>` being the normalized gRPC Metadata key (lowercase), the value being the metadata values.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// "rpc.grpc.response.metadata.my-custom-metadata-attribute=["attribute_value"]",
	//
	// Note: Instrumentations SHOULD require an explicit configuration of which metadata values are to be captured. Including all response metadata values can be a security risk - explicit configuration helps avoid leaking sensitive information
	AttributeRPCGRPCResponseMetadata = "rpc.grpc.response.metadata"

	// The [numeric status code] of the gRPC request.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [numeric status code]: https://github.com/grpc/grpc/blob/v1.33.2/doc/statuscodes.md
	AttributeRPCGRPCStatusCode = "rpc.grpc.status_code"

	// `error.code` property of response if it is an error response.
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// -32700,
	// 100,
	AttributeRPCJsonrpcErrorCode = "rpc.jsonrpc.error_code"

	// `error.message` property of response if it is an error response.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Parse error",
	// "User already exists",
	AttributeRPCJsonrpcErrorMessage = "rpc.jsonrpc.error_message"

	// `id` property of request or response. Since protocol allows id to be int, string, `null` or missing (for notifications), value is expected to be cast to string for simplicity. Use empty string in case of `null` value. Omit entirely if this is a notification.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "10",
	// "request-7",
	// "",
	AttributeRPCJsonrpcRequestID = "rpc.jsonrpc.request_id"

	// Protocol version as in `jsonrpc` property of request/response. Since JSON-RPC 1.0 doesn't specify this, the value can be omitted.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2.0",
	// "1.0",
	AttributeRPCJsonrpcVersion = "rpc.jsonrpc.version"

	// Compressed size of the message in bytes.
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	AttributeRPCMessageCompressedSize = "rpc.message.compressed_size"

	// MUST be calculated as two different counters starting from `1` one for sent messages and one for received message.
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	// Note: This way we guarantee that the values will be consistent between different implementations
	AttributeRPCMessageID = "rpc.message.id"

	// Whether this is a received or sent message.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeRPCMessageType = "rpc.message.type"

	// Uncompressed size of the message in bytes.
	// Stability: Experimental
	// Type: int
	//
	// Examples: undefined
	AttributeRPCMessageUncompressedSize = "rpc.message.uncompressed_size"

	// The name of the (logical) method being called, must be equal to the $method part in the span name.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "exampleMethod"
	// Note: This is the logical name of the method from the RPC interface perspective, which can be different from the name of any implementing method/function. The `code.function` attribute may be used to store the latter (e.g., method actually executing the call on the server side, RPC client stub method on the client side)
	AttributeRPCMethod = "rpc.method"

	// The full (logical) name of the service being called, including its package name, if applicable.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "myservice.EchoService"
	// Note: This is the logical name of the service from the RPC interface perspective, which can be different from the name of any implementing class. The `code.namespace` attribute may be used to store the latter (despite the attribute name, it may include a class name; e.g., class with method actually executing the call on the server side, RPC client stub class on the client side)
	AttributeRPCService = "rpc.service"

	// A string identifying the remoting system. See below for a list of well-known identifiers.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeRPCSystem = "rpc.system"
)

// Enum values for rpc.connect_rpc.error_code
const (
	// none
	AttributeRPCConnectRPCErrorCodeCancelled = "cancelled"
	// none
	AttributeRPCConnectRPCErrorCodeUnknown = "unknown"
	// none
	AttributeRPCConnectRPCErrorCodeInvalidArgument = "invalid_argument"
	// none
	AttributeRPCConnectRPCErrorCodeDeadlineExceeded = "deadline_exceeded"
	// none
	AttributeRPCConnectRPCErrorCodeNotFound = "not_found"
	// none
	AttributeRPCConnectRPCErrorCodeAlreadyExists = "already_exists"
	// none
	AttributeRPCConnectRPCErrorCodePermissionDenied = "permission_denied"
	// none
	AttributeRPCConnectRPCErrorCodeResourceExhausted = "resource_exhausted"
	// none
	AttributeRPCConnectRPCErrorCodeFailedPrecondition = "failed_precondition"
	// none
	AttributeRPCConnectRPCErrorCodeAborted = "aborted"
	// none
	AttributeRPCConnectRPCErrorCodeOutOfRange = "out_of_range"
	// none
	AttributeRPCConnectRPCErrorCodeUnimplemented = "unimplemented"
	// none
	AttributeRPCConnectRPCErrorCodeInternal = "internal"
	// none
	AttributeRPCConnectRPCErrorCodeUnavailable = "unavailable"
	// none
	AttributeRPCConnectRPCErrorCodeDataLoss = "data_loss"
	// none
	AttributeRPCConnectRPCErrorCodeUnauthenticated = "unauthenticated"
)

// Enum values for rpc.message.type
const (
	// none
	AttributeRPCMessageTypeSent = "SENT"
	// none
	AttributeRPCMessageTypeReceived = "RECEIVED"
)

// Enum values for rpc.system
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

// Namespace: server
const (

	// Server domain name if available without reverse DNS lookup; otherwise, IP address or Unix domain socket name.
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "example.com",
	// "10.1.2.80",
	// "/tmp/my.sock",
	//
	// Note: When observed from the client side, and when communicating through an intermediary, `server.address` SHOULD represent the server address behind any intermediaries, for example proxies, if it's available
	AttributeServerAddress = "server.address"

	// Server port number.
	// Stability: Stable
	// Type: int
	//
	// Examples:
	// 80,
	// 8080,
	// 443,
	//
	// Note: When observed from the client side, and when communicating through an intermediary, `server.port` SHOULD represent the server port behind any intermediaries, for example proxies, if it's available
	AttributeServerPort = "server.port"
)

// Namespace: service
const (

	// The string ID of the service instance.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "627cc493-f310-47de-96bd-71410b7dec09",
	//
	// Note: MUST be unique for each instance of the same `service.namespace,service.name` pair (in other words
	// `service.namespace,service.name,service.instance.id` triplet MUST be globally unique). The ID helps to
	// distinguish instances of the same service that exist at the same time (e.g. instances of a horizontally scaled
	// service).
	//
	// Implementations, such as SDKs, are recommended to generate a random Version 1 or Version 4 [RFC
	// 4122] UUID, but are free to use an inherent unique ID as the source of
	// this value if stability is desirable. In that case, the ID SHOULD be used as source of a UUID Version 5 and
	// SHOULD use the following UUID as the namespace: `4d63009a-8d0f-11ee-aad7-4c796ed8e320`.
	//
	// UUIDs are typically recommended, as only an opaque value for the purposes of identifying a service instance is
	// needed. Similar to what can be seen in the man page for the
	// [`/etc/machine-id`] file, the underlying
	// data, such as pod name and namespace should be treated as confidential, being the user's choice to expose it
	// or not via another resource attribute.
	//
	// For applications running behind an application server (like unicorn), we do not recommend using one identifier
	// for all processes participating in the application. Instead, it's recommended each division (e.g. a worker
	// thread in unicorn) to have its own instance.id.
	//
	// It's not recommended for a Collector to set `service.instance.id` if it can't unambiguously determine the
	// service instance that is generating that telemetry. For instance, creating an UUID based on `pod.name` will
	// likely be wrong, as the Collector might not know from which container within that pod the telemetry originated.
	// However, Collectors can set the `service.instance.id` if they can unambiguously determine the service instance
	// for that telemetry. This is typically the case for scraping receivers, as they know the target address and
	// port
	//
	// [RFC
	// 4122]: https://www.ietf.org/rfc/rfc4122.txt
	// [`/etc/machine-id`]: https://www.freedesktop.org/software/systemd/man/machine-id.html
	AttributeServiceInstanceID = "service.instance.id"

	// Logical name of the service.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "shoppingcart",
	//
	// Note: MUST be the same for all instances of horizontally scaled services. If the value was not specified, SDKs MUST fallback to `unknown_service:` concatenated with [`process.executable.name`], e.g. `unknown_service:bash`. If `process.executable.name` is not available, the value MUST be set to `unknown_service`
	//
	// [`process.executable.name`]: process.md
	AttributeServiceName = "service.name"

	// A namespace for `service.name`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Shop",
	//
	// Note: A string value having a meaning that helps to distinguish a group of services, for example the team name that owns a group of services. `service.name` is expected to be unique within the same namespace. If `service.namespace` is not specified in the Resource then `service.name` is expected to be unique for all services that have no explicit namespace defined (so the empty/unspecified namespace is simply one more valid namespace). Zero-length namespace string is assumed equal to unspecified namespace
	AttributeServiceNamespace = "service.namespace"

	// The version string of the service API or implementation. The format is not defined by these conventions.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "2.0.0",
	// "a01dbef8a",
	AttributeServiceVersion = "service.version"
)

// Namespace: session
const (

	// A unique id to identify a session.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "00112233-4455-6677-8899-aabbccddeeff"
	AttributeSessionID = "session.id"

	// The previous `session.id` for this user, when known.
	// Stability: Experimental
	// Type: string
	//
	// Examples: "00112233-4455-6677-8899-aabbccddeeff"
	AttributeSessionPreviousID = "session.previous_id"
)

// Namespace: signalr
const (

	// SignalR HTTP connection closure status.
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "app_shutdown",
	// "timeout",
	AttributeSignalrConnectionStatus = "signalr.connection.status"

	// [SignalR transport type]
	// Stability: Stable
	// Type: Enum
	//
	// Examples:
	// "web_sockets",
	// "long_polling",
	//
	// [SignalR transport type]: https://github.com/dotnet/aspnetcore/blob/main/src/SignalR/docs/specs/TransportProtocols.md
	AttributeSignalrTransport = "signalr.transport"
)

// Enum values for signalr.connection.status
const (
	// The connection was closed normally
	AttributeSignalrConnectionStatusNormalClosure = "normal_closure"
	// The connection was closed due to a timeout
	AttributeSignalrConnectionStatusTimeout = "timeout"
	// The connection was closed because the app is shutting down
	AttributeSignalrConnectionStatusAppShutdown = "app_shutdown"
)

// Enum values for signalr.transport
const (
	// ServerSentEvents protocol
	AttributeSignalrTransportServerSentEvents = "server_sent_events"
	// LongPolling protocol
	AttributeSignalrTransportLongPolling = "long_polling"
	// WebSockets protocol
	AttributeSignalrTransportWebSockets = "web_sockets"
)

// Namespace: source
const (

	// Source address - domain name if available without reverse DNS lookup; otherwise, IP address or Unix domain socket name.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "source.example.com",
	// "10.1.2.80",
	// "/tmp/my.sock",
	//
	// Note: When observed from the destination side, and when communicating through an intermediary, `source.address` SHOULD represent the source address behind any intermediaries, for example proxies, if it's available
	AttributeSourceAddress = "source.address"

	// Source port number
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 3389,
	// 2888,
	AttributeSourcePort = "source.port"
)

// Namespace: system
const (

	// The logical CPU number [0..n-1]
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 1,
	AttributeSystemCPULogicalNumber = "system.cpu.logical_number"

	// Deprecated, use `cpu.mode` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `cpu.mode`
	//
	// Examples:
	// "idle",
	// "interrupt",
	AttributeSystemCPUState = "system.cpu.state"

	// The device identifier
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "(identifier)",
	AttributeSystemDevice = "system.device"

	// The filesystem mode
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "rw, ro",
	AttributeSystemFilesystemMode = "system.filesystem.mode"

	// The filesystem mount path
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/mnt/data",
	AttributeSystemFilesystemMountpoint = "system.filesystem.mountpoint"

	// The filesystem state
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "used",
	AttributeSystemFilesystemState = "system.filesystem.state"

	// The filesystem type
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "ext4",
	AttributeSystemFilesystemType = "system.filesystem.type"

	// The memory state
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "free",
	// "cached",
	AttributeSystemMemoryState = "system.memory.state"

	// A stateless protocol MUST NOT set this attribute
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "close_wait",
	AttributeSystemNetworkState = "system.network.state"

	// The paging access direction
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "in",
	AttributeSystemPagingDirection = "system.paging.direction"

	// The memory paging state
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "free",
	AttributeSystemPagingState = "system.paging.state"

	// The memory paging type
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "minor",
	AttributeSystemPagingType = "system.paging.type"

	// The process state, e.g., [Linux Process State Codes]
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "running",
	//
	// [Linux Process State Codes]: https://man7.org/linux/man-pages/man1/ps.1.html#PROCESS_STATE_CODES
	AttributeSystemProcessStatus = "system.process.status"

	// Deprecated, use `system.process.status` instead.
	// Stability: Experimental
	// Type: Enum
	// Deprecated: Replaced by `system.process.status`.
	//
	// Examples:
	// "running",
	AttributeSystemProcessesStatus = "system.processes.status"
)

// Enum values for system.cpu.state
const (
	// none
	AttributeSystemCPUStateUser = "user"
	// none
	AttributeSystemCPUStateSystem = "system"
	// none
	AttributeSystemCPUStateNice = "nice"
	// none
	AttributeSystemCPUStateIdle = "idle"
	// none
	AttributeSystemCPUStateIowait = "iowait"
	// none
	AttributeSystemCPUStateInterrupt = "interrupt"
	// none
	AttributeSystemCPUStateSteal = "steal"
)

// Enum values for system.filesystem.state
const (
	// none
	AttributeSystemFilesystemStateUsed = "used"
	// none
	AttributeSystemFilesystemStateFree = "free"
	// none
	AttributeSystemFilesystemStateReserved = "reserved"
)

// Enum values for system.filesystem.type
const (
	// none
	AttributeSystemFilesystemTypeFat32 = "fat32"
	// none
	AttributeSystemFilesystemTypeExfat = "exfat"
	// none
	AttributeSystemFilesystemTypeNtfs = "ntfs"
	// none
	AttributeSystemFilesystemTypeRefs = "refs"
	// none
	AttributeSystemFilesystemTypeHfsplus = "hfsplus"
	// none
	AttributeSystemFilesystemTypeExt4 = "ext4"
)

// Enum values for system.memory.state
const (
	// none
	AttributeSystemMemoryStateUsed = "used"
	// none
	AttributeSystemMemoryStateFree = "free"
	// none
	AttributeSystemMemoryStateShared = "shared"
	// none
	AttributeSystemMemoryStateBuffers = "buffers"
	// none
	AttributeSystemMemoryStateCached = "cached"
)

// Enum values for system.network.state
const (
	// none
	AttributeSystemNetworkStateClose = "close"
	// none
	AttributeSystemNetworkStateCloseWait = "close_wait"
	// none
	AttributeSystemNetworkStateClosing = "closing"
	// none
	AttributeSystemNetworkStateDelete = "delete"
	// none
	AttributeSystemNetworkStateEstablished = "established"
	// none
	AttributeSystemNetworkStateFinWait1 = "fin_wait_1"
	// none
	AttributeSystemNetworkStateFinWait2 = "fin_wait_2"
	// none
	AttributeSystemNetworkStateLastAck = "last_ack"
	// none
	AttributeSystemNetworkStateListen = "listen"
	// none
	AttributeSystemNetworkStateSynRecv = "syn_recv"
	// none
	AttributeSystemNetworkStateSynSent = "syn_sent"
	// none
	AttributeSystemNetworkStateTimeWait = "time_wait"
)

// Enum values for system.paging.direction
const (
	// none
	AttributeSystemPagingDirectionIn = "in"
	// none
	AttributeSystemPagingDirectionOut = "out"
)

// Enum values for system.paging.state
const (
	// none
	AttributeSystemPagingStateUsed = "used"
	// none
	AttributeSystemPagingStateFree = "free"
)

// Enum values for system.paging.type
const (
	// none
	AttributeSystemPagingTypeMajor = "major"
	// none
	AttributeSystemPagingTypeMinor = "minor"
)

// Enum values for system.process.status
const (
	// none
	AttributeSystemProcessStatusRunning = "running"
	// none
	AttributeSystemProcessStatusSleeping = "sleeping"
	// none
	AttributeSystemProcessStatusStopped = "stopped"
	// none
	AttributeSystemProcessStatusDefunct = "defunct"
)

// Enum values for system.processes.status
const (
	// none
	AttributeSystemProcessesStatusRunning = "running"
	// none
	AttributeSystemProcessesStatusSleeping = "sleeping"
	// none
	AttributeSystemProcessesStatusStopped = "stopped"
	// none
	AttributeSystemProcessesStatusDefunct = "defunct"
)

// Namespace: telemetry
const (

	// The name of the auto instrumentation agent or distribution, if used.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "parts-unlimited-java",
	//
	// Note: Official auto instrumentation agents and distributions SHOULD set the `telemetry.distro.name` attribute to
	// a string starting with `opentelemetry-`, e.g. `opentelemetry-java-instrumentation`
	AttributeTelemetryDistroName = "telemetry.distro.name"

	// The version string of the auto instrumentation agent or distribution, if used.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1.2.3",
	AttributeTelemetryDistroVersion = "telemetry.distro.version"

	// The language of the telemetry SDK.
	//
	// Stability: Stable
	// Type: Enum
	//
	// Examples: undefined
	AttributeTelemetrySDKLanguage = "telemetry.sdk.language"

	// The name of the telemetry SDK as defined above.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "opentelemetry",
	//
	// Note: The OpenTelemetry SDK MUST set the `telemetry.sdk.name` attribute to `opentelemetry`.
	// If another SDK, like a fork or a vendor-provided implementation, is used, this SDK MUST set the
	// `telemetry.sdk.name` attribute to the fully-qualified class or module name of this SDK's main entry point
	// or another suitable identifier depending on the language.
	// The identifier `opentelemetry` is reserved and MUST NOT be used in this case.
	// All custom identifiers SHOULD be stable across different versions of an implementation
	AttributeTelemetrySDKName = "telemetry.sdk.name"

	// The version string of the telemetry SDK.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "1.2.3",
	AttributeTelemetrySDKVersion = "telemetry.sdk.version"
)

// Enum values for telemetry.sdk.language
const (
	// none
	AttributeTelemetrySDKLanguageCPP = "cpp"
	// none
	AttributeTelemetrySDKLanguageDotnet = "dotnet"
	// none
	AttributeTelemetrySDKLanguageErlang = "erlang"
	// none
	AttributeTelemetrySDKLanguageGo = "go"
	// none
	AttributeTelemetrySDKLanguageJava = "java"
	// none
	AttributeTelemetrySDKLanguageNodejs = "nodejs"
	// none
	AttributeTelemetrySDKLanguagePHP = "php"
	// none
	AttributeTelemetrySDKLanguagePython = "python"
	// none
	AttributeTelemetrySDKLanguageRuby = "ruby"
	// none
	AttributeTelemetrySDKLanguageRust = "rust"
	// none
	AttributeTelemetrySDKLanguageSwift = "swift"
	// none
	AttributeTelemetrySDKLanguageWebjs = "webjs"
)

// Namespace: test
const (

	// The fully qualified human readable name of the [test case].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "org.example.TestCase1.test1",
	// "example/tests/TestCase1.test1",
	// "ExampleTestCase1_test1",
	//
	// [test case]: https://en.wikipedia.org/wiki/Test_case
	AttributeTestCaseName = "test.case.name"

	// The status of the actual test case result from test execution.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "pass",
	// "fail",
	AttributeTestCaseResultStatus = "test.case.result.status"

	// The human readable name of a [test suite].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "TestSuite1",
	//
	// [test suite]: https://en.wikipedia.org/wiki/Test_suite
	AttributeTestSuiteName = "test.suite.name"

	// The status of the test suite run.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "success",
	// "failure",
	// "skipped",
	// "aborted",
	// "timed_out",
	// "in_progress",
	AttributeTestSuiteRunStatus = "test.suite.run.status"
)

// Enum values for test.case.result.status
const (
	// pass
	AttributeTestCaseResultStatusPass = "pass"
	// fail
	AttributeTestCaseResultStatusFail = "fail"
)

// Enum values for test.suite.run.status
const (
	// success
	AttributeTestSuiteRunStatusSuccess = "success"
	// failure
	AttributeTestSuiteRunStatusFailure = "failure"
	// skipped
	AttributeTestSuiteRunStatusSkipped = "skipped"
	// aborted
	AttributeTestSuiteRunStatusAborted = "aborted"
	// timed_out
	AttributeTestSuiteRunStatusTimedOut = "timed_out"
	// in_progress
	AttributeTestSuiteRunStatusInProgress = "in_progress"
)

// Namespace: thread
const (

	// Current "managed" thread ID (as opposed to OS thread ID).
	//
	// Stability: Experimental
	// Type: int
	AttributeThreadID = "thread.id"

	// Current thread name.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples: "main"
	AttributeThreadName = "thread.name"
)

// Namespace: tls
const (

	// String indicating the [cipher] used during the current connection.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	// "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	//
	// Note: The values allowed for `tls.cipher` MUST be one of the `Descriptions` of the [registered TLS Cipher Suits]
	//
	// [cipher]: https://datatracker.ietf.org/doc/html/rfc5246#appendix-A.5
	// [registered TLS Cipher Suits]: https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#table-tls-parameters-4
	AttributeTLSCipher = "tls.cipher"

	// PEM-encoded stand-alone certificate offered by the client. This is usually mutually-exclusive of `client.certificate_chain` since this value also exists in that list.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "MII...",
	AttributeTLSClientCertificate = "tls.client.certificate"

	// Array of PEM-encoded certificates that make up the certificate chain offered by the client. This is usually mutually-exclusive of `client.certificate` since that value should be the first certificate in the chain.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "MII...",
	// "MI...",
	// ],
	AttributeTLSClientCertificateChain = "tls.client.certificate_chain"

	// Certificate fingerprint using the MD5 digest of DER-encoded version of certificate offered by the client. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "0F76C7F2C55BFD7D8E8B8F4BFBF0C9EC",
	AttributeTLSClientHashMd5 = "tls.client.hash.md5"

	// Certificate fingerprint using the SHA1 digest of DER-encoded version of certificate offered by the client. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "9E393D93138888D288266C2D915214D1D1CCEB2A",
	AttributeTLSClientHashSha1 = "tls.client.hash.sha1"

	// Certificate fingerprint using the SHA256 digest of DER-encoded version of certificate offered by the client. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "0687F666A054EF17A08E2F2162EAB4CBC0D265E1D7875BE74BF3C712CA92DAF0",
	AttributeTLSClientHashSha256 = "tls.client.hash.sha256"

	// Distinguished name of [subject] of the issuer of the x.509 certificate presented by the client.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "CN=Example Root CA, OU=Infrastructure Team, DC=example, DC=com",
	//
	// [subject]: https://datatracker.ietf.org/doc/html/rfc5280#section-4.1.2.6
	AttributeTLSClientIssuer = "tls.client.issuer"

	// A hash that identifies clients based on how they perform an SSL/TLS handshake.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "d4e5b18d6b55c71272893221c96ba240",
	AttributeTLSClientJa3 = "tls.client.ja3"

	// Date/Time indicating when client certificate is no longer considered valid.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2021-01-01T00:00:00.000Z",
	AttributeTLSClientNotAfter = "tls.client.not_after"

	// Date/Time indicating when client certificate is first considered valid.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1970-01-01T00:00:00.000Z",
	AttributeTLSClientNotBefore = "tls.client.not_before"

	// Deprecated, use `server.address` instead.
	// Stability: Experimental
	// Type: string
	// Deprecated: Replaced by `server.address`.
	//
	// Examples:
	// "opentelemetry.io",
	AttributeTLSClientServerName = "tls.client.server_name"

	// Distinguished name of subject of the x.509 certificate presented by the client.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "CN=myclient, OU=Documentation Team, DC=example, DC=com",
	AttributeTLSClientSubject = "tls.client.subject"

	// Array of ciphers offered by the client during the client hello.
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	// "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	// ],
	AttributeTLSClientSupportedCiphers = "tls.client.supported_ciphers"

	// String indicating the curve used for the given cipher, when applicable
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "secp256r1",
	AttributeTLSCurve = "tls.curve"

	// Boolean flag indicating if the TLS negotiation was successful and transitioned to an encrypted tunnel.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples:
	// true,
	AttributeTLSEstablished = "tls.established"

	// String indicating the protocol being tunneled. Per the values in the [IANA registry], this string should be lower case.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "http/1.1",
	//
	// [IANA registry]: https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values.xhtml#alpn-protocol-ids
	AttributeTLSNextProtocol = "tls.next_protocol"

	// Normalized lowercase protocol name parsed from original string of the negotiated [SSL/TLS protocol version]
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	//
	// [SSL/TLS protocol version]: https://www.openssl.org/docs/man1.1.1/man3/SSL_get_version.html#RETURN-VALUES
	AttributeTLSProtocolName = "tls.protocol.name"

	// Numeric part of the version parsed from the original string of the negotiated [SSL/TLS protocol version]
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1.2",
	// "3",
	//
	// [SSL/TLS protocol version]: https://www.openssl.org/docs/man1.1.1/man3/SSL_get_version.html#RETURN-VALUES
	AttributeTLSProtocolVersion = "tls.protocol.version"

	// Boolean flag indicating if this TLS connection was resumed from an existing TLS negotiation.
	// Stability: Experimental
	// Type: boolean
	//
	// Examples:
	// true,
	AttributeTLSResumed = "tls.resumed"

	// PEM-encoded stand-alone certificate offered by the server. This is usually mutually-exclusive of `server.certificate_chain` since this value also exists in that list.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "MII...",
	AttributeTLSServerCertificate = "tls.server.certificate"

	// Array of PEM-encoded certificates that make up the certificate chain offered by the server. This is usually mutually-exclusive of `server.certificate` since that value should be the first certificate in the chain.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "MII...",
	// "MI...",
	// ],
	AttributeTLSServerCertificateChain = "tls.server.certificate_chain"

	// Certificate fingerprint using the MD5 digest of DER-encoded version of certificate offered by the server. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "0F76C7F2C55BFD7D8E8B8F4BFBF0C9EC",
	AttributeTLSServerHashMd5 = "tls.server.hash.md5"

	// Certificate fingerprint using the SHA1 digest of DER-encoded version of certificate offered by the server. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "9E393D93138888D288266C2D915214D1D1CCEB2A",
	AttributeTLSServerHashSha1 = "tls.server.hash.sha1"

	// Certificate fingerprint using the SHA256 digest of DER-encoded version of certificate offered by the server. For consistency with other hash values, this value should be formatted as an uppercase hash.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "0687F666A054EF17A08E2F2162EAB4CBC0D265E1D7875BE74BF3C712CA92DAF0",
	AttributeTLSServerHashSha256 = "tls.server.hash.sha256"

	// Distinguished name of [subject] of the issuer of the x.509 certificate presented by the client.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "CN=Example Root CA, OU=Infrastructure Team, DC=example, DC=com",
	//
	// [subject]: https://datatracker.ietf.org/doc/html/rfc5280#section-4.1.2.6
	AttributeTLSServerIssuer = "tls.server.issuer"

	// A hash that identifies servers based on how they perform an SSL/TLS handshake.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "d4e5b18d6b55c71272893221c96ba240",
	AttributeTLSServerJa3s = "tls.server.ja3s"

	// Date/Time indicating when server certificate is no longer considered valid.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "2021-01-01T00:00:00.000Z",
	AttributeTLSServerNotAfter = "tls.server.not_after"

	// Date/Time indicating when server certificate is first considered valid.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "1970-01-01T00:00:00.000Z",
	AttributeTLSServerNotBefore = "tls.server.not_before"

	// Distinguished name of subject of the x.509 certificate presented by the server.
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "CN=myserver, OU=Documentation Team, DC=example, DC=com",
	AttributeTLSServerSubject = "tls.server.subject"
)

// Enum values for tls.protocol.name
const (
	// none
	AttributeTLSProtocolNameSsl = "ssl"
	// none
	AttributeTLSProtocolNameTLS = "tls"
)

// Namespace: url
const (

	// Domain extracted from the `url.full`, such as "opentelemetry.io".
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "www.foo.bar",
	// "opentelemetry.io",
	// "3.12.167.2",
	// "[1080:0:0:0:8:800:200C:417A]",
	//
	// Note: In some cases a URL may refer to an IP and/or port directly, without a domain name. In this case, the IP address would go to the domain field. If the URL contains a [literal IPv6 address] enclosed by `[` and `]`, the `[` and `]` characters should also be captured in the domain field
	//
	// [literal IPv6 address]: https://www.rfc-editor.org/rfc/rfc2732#section-2
	AttributeURLDomain = "url.domain"

	// The file extension extracted from the `url.full`, excluding the leading dot.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "png",
	// "gz",
	//
	// Note: The file extension is only set if it exists, as not every url has a file extension. When the file name has multiple extensions `example.tar.gz`, only the last one should be captured `gz`, not `tar.gz`
	AttributeURLExtension = "url.extension"

	// The [URI fragment] component
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "SemConv",
	//
	// [URI fragment]: https://www.rfc-editor.org/rfc/rfc3986#section-3.5
	AttributeURLFragment = "url.fragment"

	// Absolute URL describing a network resource according to [RFC3986]
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "https://www.foo.bar/search?q=OpenTelemetry#SemConv",
	// "//localhost",
	//
	// Note: For network calls, URL usually has `scheme://host[:port][path][?query][#fragment]` format, where the fragment is not transmitted over HTTP, but if it is known, it SHOULD be included nevertheless.
	// `url.full` MUST NOT contain credentials passed via URL in form of `https://username:password@www.example.com/`. In such case username and password SHOULD be redacted and attribute's value SHOULD be `https://REDACTED:REDACTED@www.example.com/`.
	// `url.full` SHOULD capture the absolute URL when it is available (or can be reconstructed). Sensitive content provided in `url.full` SHOULD be scrubbed when instrumentations can identify it
	//
	// [RFC3986]: https://www.rfc-editor.org/rfc/rfc3986
	AttributeURLFull = "url.full"

	// Unmodified original URL as seen in the event source.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "https://www.foo.bar/search?q=OpenTelemetry#SemConv",
	// "search?q=OpenTelemetry",
	//
	// Note: In network monitoring, the observed URL may be a full URL, whereas in access logs, the URL is often just represented as a path. This field is meant to represent the URL as it was observed, complete or not.
	// `url.original` might contain credentials passed via URL in form of `https://username:password@www.example.com/`. In such case password and username SHOULD NOT be redacted and attribute's value SHOULD remain the same
	AttributeURLOriginal = "url.original"

	// The [URI path] component
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "/search",
	//
	// Note: Sensitive content provided in `url.path` SHOULD be scrubbed when instrumentations can identify it
	//
	// [URI path]: https://www.rfc-editor.org/rfc/rfc3986#section-3.3
	AttributeURLPath = "url.path"

	// Port extracted from the `url.full`
	//
	// Stability: Experimental
	// Type: int
	//
	// Examples:
	// 443,
	AttributeURLPort = "url.port"

	// The [URI query] component
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "q=OpenTelemetry",
	//
	// Note: Sensitive content provided in `url.query` SHOULD be scrubbed when instrumentations can identify it
	//
	// [URI query]: https://www.rfc-editor.org/rfc/rfc3986#section-3.4
	AttributeURLQuery = "url.query"

	// The highest registered url domain, stripped of the subdomain.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "example.com",
	// "foo.co.uk",
	//
	// Note: This value can be determined precisely with the [public suffix list]. For example, the registered domain for `foo.example.com` is `example.com`. Trying to approximate this by simply taking the last two labels will not work well for TLDs such as `co.uk`
	//
	// [public suffix list]: http://publicsuffix.org
	AttributeURLRegisteredDomain = "url.registered_domain"

	// The [URI scheme] component identifying the used protocol.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "https",
	// "ftp",
	// "telnet",
	//
	// [URI scheme]: https://www.rfc-editor.org/rfc/rfc3986#section-3.1
	AttributeURLScheme = "url.scheme"

	// The subdomain portion of a fully qualified domain name includes all of the names except the host name under the registered_domain. In a partially qualified domain, or if the qualification level of the full name cannot be determined, subdomain contains all of the names below the registered domain.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "east",
	// "sub2.sub1",
	//
	// Note: The subdomain portion of `www.east.mydomain.co.uk` is `east`. If the domain has multiple levels of subdomain, such as `sub2.sub1.example.com`, the subdomain field should contain `sub2.sub1`, with no trailing period
	AttributeURLSubdomain = "url.subdomain"

	// The low-cardinality template of an [absolute path reference].
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "/users/{id}",
	// "/users/:id",
	// "/users?id={id}",
	//
	// [absolute path reference]: https://www.rfc-editor.org/rfc/rfc3986#section-4.2
	AttributeURLTemplate = "url.template"

	// The effective top level domain (eTLD), also known as the domain suffix, is the last part of the domain name. For example, the top level domain for example.com is `com`.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "com",
	// "co.uk",
	//
	// Note: This value can be determined precisely with the [public suffix list]
	//
	// [public suffix list]: http://publicsuffix.org
	AttributeURLTopLevelDomain = "url.top_level_domain"
)

// Namespace: user
const (

	// User email address.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "a.einstein@example.com",
	AttributeUserEmail = "user.email"

	// User's full name
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Albert Einstein",
	AttributeUserFullName = "user.full_name"

	// Unique user hash to correlate information for a user in anonymized form.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "364fc68eaf4c8acec74a4e52d7d1feaa",
	//
	// Note: Useful if `user.id` or `user.name` contain confidential information and cannot be used
	AttributeUserHash = "user.hash"

	// Unique identifier of the user.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "S-1-5-21-202424912787-2692429404-2351956786-1000",
	AttributeUserID = "user.id"

	// Short name or login/username of the user.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "a.einstein",
	AttributeUserName = "user.name"

	// Array of user roles at the time of the event.
	//
	// Stability: Experimental
	// Type: string[]
	//
	// Examples:
	// [
	// "admin",
	// "reporting_user",
	// ],
	AttributeUserRoles = "user.roles"
)

// Namespace: user_agent
const (

	// Name of the user-agent extracted from original. Usually refers to the browser's name.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Safari",
	// "YourApp",
	//
	// Note: [Example] of extracting browser's name from original string. In the case of using a user-agent for non-browser products, such as microservices with multiple names/versions inside the `user_agent.original`, the most significant name SHOULD be selected. In such a scenario it should align with `user_agent.version`
	//
	// [Example]: https://www.whatsmyua.info
	AttributeUserAgentName = "user_agent.name"

	// Value of the [HTTP User-Agent] header sent by the client.
	//
	// Stability: Stable
	// Type: string
	//
	// Examples:
	// "CERN-LineMode/2.15 libwww/2.17b3",
	// "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
	// "YourApp/1.0.0 grpc-java-okhttp/1.27.2",
	//
	// [HTTP User-Agent]: https://www.rfc-editor.org/rfc/rfc9110.html#field.user-agent
	AttributeUserAgentOriginal = "user_agent.original"

	// Version of the user-agent extracted from original. Usually refers to the browser's version
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "14.1.2",
	// "1.0.0",
	//
	// Note: [Example] of extracting browser's version from original string. In the case of using a user-agent for non-browser products, such as microservices with multiple names/versions inside the `user_agent.original`, the most significant version SHOULD be selected. In such a scenario it should align with `user_agent.name`
	//
	// [Example]: https://www.whatsmyua.info
	AttributeUserAgentVersion = "user_agent.version"
)

// Namespace: v8js
const (

	// The type of garbage collection.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	AttributeV8jsGcType = "v8js.gc.type"

	// The name of the space type of heap memory.
	// Stability: Experimental
	// Type: Enum
	//
	// Examples: undefined
	// Note: Value can be retrieved from value `space_name` of [`v8.getHeapSpaceStatistics()`]
	//
	// [`v8.getHeapSpaceStatistics()`]: https://nodejs.org/api/v8.html#v8getheapspacestatistics
	AttributeV8jsHeapSpaceName = "v8js.heap.space.name"
)

// Enum values for v8js.gc.type
const (
	// Major (Mark Sweep Compact)
	AttributeV8jsGcTypeMajor = "major"
	// Minor (Scavenge)
	AttributeV8jsGcTypeMinor = "minor"
	// Incremental (Incremental Marking)
	AttributeV8jsGcTypeIncremental = "incremental"
	// Weak Callbacks (Process Weak Callbacks)
	AttributeV8jsGcTypeWeakcb = "weakcb"
)

// Enum values for v8js.heap.space.name
const (
	// New memory space
	AttributeV8jsHeapSpaceNameNewSpace = "new_space"
	// Old memory space
	AttributeV8jsHeapSpaceNameOldSpace = "old_space"
	// Code memory space
	AttributeV8jsHeapSpaceNameCodeSpace = "code_space"
	// Map memory space
	AttributeV8jsHeapSpaceNameMapSpace = "map_space"
	// Large object memory space
	AttributeV8jsHeapSpaceNameLargeObjectSpace = "large_object_space"
)

// Namespace: vcs
const (

	// The ID of the change (pull request/merge request) if applicable. This is usually a unique (within repository) identifier generated by the VCS system.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "123",
	AttributeVcsRepositoryChangeID = "vcs.repository.change.id"

	// The human readable title of the change (pull request/merge request). This title is often a brief summary of the change and may get merged in to a ref as the commit summary.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "Fixes broken thing",
	// "feat: add my new feature",
	// "[chore] update dependency",
	AttributeVcsRepositoryChangeTitle = "vcs.repository.change.title"

	// The name of the [reference] such as **branch** or **tag** in the repository.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "my-feature-branch",
	// "tag-1-test",
	//
	// [reference]: https://git-scm.com/docs/gitglossary#def_ref
	AttributeVcsRepositoryRefName = "vcs.repository.ref.name"

	// The revision, literally [revised version], The revision most often refers to a commit object in Git, or a revision number in SVN.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "9d59409acf479dfa0df1aa568182e43e43df8bbe28d60fcf2bc52e30068802cc",
	// "main",
	// "123",
	// "HEAD",
	//
	// Note: The revision can be a full [hash value (see glossary)],
	// of the recorded change to a ref within a repository pointing to a
	// commit [commit] object. It does
	// not necessarily have to be a hash; it can simply define a
	// [revision number]
	// which is an integer that is monotonically increasing. In cases where
	// it is identical to the `ref.name`, it SHOULD still be included. It is
	// up to the implementer to decide which value to set as the revision
	// based on the VCS system and situational context
	//
	// [revised version]: https://www.merriam-webster.com/dictionary/revision
	// [hash value (see glossary)]: https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf
	// [commit]: https://git-scm.com/docs/git-commit
	// [revision number]: https://svnbook.red-bean.com/en/1.7/svn.tour.revs.specifiers.html
	AttributeVcsRepositoryRefRevision = "vcs.repository.ref.revision"

	// The type of the [reference] in the repository.
	//
	// Stability: Experimental
	// Type: Enum
	//
	// Examples:
	// "branch",
	// "tag",
	//
	// [reference]: https://git-scm.com/docs/gitglossary#def_ref
	AttributeVcsRepositoryRefType = "vcs.repository.ref.type"

	// The [URL] of the repository providing the complete address in order to locate and identify the repository.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "https://github.com/opentelemetry/open-telemetry-collector-contrib",
	// "https://gitlab.com/my-org/my-project/my-projects-project/repo",
	//
	// [URL]: https://en.wikipedia.org/wiki/URL
	AttributeVcsRepositoryURLFull = "vcs.repository.url.full"
)

// Enum values for vcs.repository.ref.type
const (
	// [branch]
	//
	// [branch]: https://git-scm.com/docs/gitglossary#Documentation/gitglossary.txt-aiddefbranchabranch
	AttributeVcsRepositoryRefTypeBranch = "branch"
	// [tag]
	//
	// [tag]: https://git-scm.com/docs/gitglossary#Documentation/gitglossary.txt-aiddeftagatag
	AttributeVcsRepositoryRefTypeTag = "tag"
)

// Namespace: webengine
const (

	// Additional description of the web engine (e.g. detailed version and edition information).
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "WildFly Full 21.0.0.Final (WildFly Core 13.0.1.Final) - 2.2.2.Final",
	AttributeWebEngineDescription = "webengine.description"

	// The name of the web engine.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "WildFly",
	AttributeWebEngineName = "webengine.name"

	// The version of the web engine.
	//
	// Stability: Experimental
	// Type: string
	//
	// Examples:
	// "21.0.0",
	AttributeWebEngineVersion = "webengine.version"
)

func GetAttribute_groupSemanticConventionAttributeNames() []string {
	return []string{
		AttributeAndroidOSAPILevel,
		AttributeAndroidState,
		AttributeArtifactAttestationFilename,
		AttributeArtifactAttestationHash,
		AttributeArtifactAttestationID,
		AttributeArtifactFilename,
		AttributeArtifactHash,
		AttributeArtifactPurl,
		AttributeArtifactVersion,
		AttributeAspnetcoreDiagnosticsExceptionResult,
		AttributeAspnetcoreDiagnosticsHandlerType,
		AttributeAspnetcoreRateLimitingPolicy,
		AttributeAspnetcoreRateLimitingResult,
		AttributeAspnetcoreRequestIsUnhandled,
		AttributeAspnetcoreRoutingIsFallback,
		AttributeAspnetcoreRoutingMatchStatus,
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
		AttributeAWSECSClusterARN,
		AttributeAWSECSContainerARN,
		AttributeAWSECSLaunchtype,
		AttributeAWSECSTaskARN,
		AttributeAWSECSTaskFamily,
		AttributeAWSECSTaskID,
		AttributeAWSECSTaskRevision,
		AttributeAWSEKSClusterARN,
		AttributeAWSLambdaInvokedARN,
		AttributeAWSLogGroupARNs,
		AttributeAWSLogGroupNames,
		AttributeAWSLogStreamARNs,
		AttributeAWSLogStreamNames,
		AttributeAWSRequestID,
		AttributeAWSS3Bucket,
		AttributeAWSS3CopySource,
		AttributeAWSS3Delete,
		AttributeAWSS3Key,
		AttributeAWSS3PartNumber,
		AttributeAWSS3UploadID,
		AttributeAzNamespace,
		AttributeAzServiceRequestID,
		AttributeBrowserBrands,
		AttributeBrowserLanguage,
		AttributeBrowserMobile,
		AttributeBrowserPlatform,
		AttributeCicdPipelineName,
		AttributeCicdPipelineRunID,
		AttributeCicdPipelineTaskName,
		AttributeCicdPipelineTaskRunID,
		AttributeCicdPipelineTaskRunURLFull,
		AttributeCicdPipelineTaskType,
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
		AttributeCodeLineno,
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
		AttributeContainerLabel,
		AttributeContainerLabels,
		AttributeContainerName,
		AttributeContainerRuntime,
		AttributeCPUMode,
		AttributeDBCassandraConsistencyLevel,
		AttributeDBCassandraCoordinatorDC,
		AttributeDBCassandraCoordinatorID,
		AttributeDBCassandraIdempotence,
		AttributeDBCassandraPageSize,
		AttributeDBCassandraSpeculativeExecutionCount,
		AttributeDBCassandraTable,
		AttributeDBClientConnectionPoolName,
		AttributeDBClientConnectionState,
		AttributeDBClientConnectionsPoolName,
		AttributeDBClientConnectionsState,
		AttributeDBCollectionName,
		AttributeDBConnectionString,
		AttributeDBCosmosDBClientID,
		AttributeDBCosmosDBConnectionMode,
		AttributeDBCosmosDBContainer,
		AttributeDBCosmosDBOperationType,
		AttributeDBCosmosDBRequestCharge,
		AttributeDBCosmosDBRequestContentLength,
		AttributeDBCosmosDBStatusCode,
		AttributeDBCosmosDBSubStatusCode,
		AttributeDBElasticsearchClusterName,
		AttributeDBElasticsearchNodeName,
		AttributeDBElasticsearchPathParts,
		AttributeDBInstanceID,
		AttributeDBJDBCDriverClassname,
		AttributeDBMongoDBCollection,
		AttributeDBMSSQLInstanceName,
		AttributeDBName,
		AttributeDBNamespace,
		AttributeDBOperation,
		AttributeDBOperationBatchSize,
		AttributeDBOperationName,
		AttributeDBQueryParameter,
		AttributeDBQueryText,
		AttributeDBRedisDatabaseIndex,
		AttributeDBSQLTable,
		AttributeDBStatement,
		AttributeDBSystem,
		AttributeDBUser,
		AttributeDeploymentEnvironment,
		AttributeDeploymentEnvironmentName,
		AttributeDeploymentID,
		AttributeDeploymentName,
		AttributeDeploymentStatus,
		AttributeDestinationAddress,
		AttributeDestinationPort,
		AttributeDeviceID,
		AttributeDeviceManufacturer,
		AttributeDeviceModelIdentifier,
		AttributeDeviceModelName,
		AttributeDiskIoDirection,
		AttributeDNSQuestionName,
		AttributeDotnetGcHeapGeneration,
		AttributeEnduserID,
		AttributeEnduserRole,
		AttributeEnduserScope,
		AttributeErrorType,
		AttributeEventName,
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
		AttributeGCPClientService,
		AttributeGCPCloudRunJobExecution,
		AttributeGCPCloudRunJobTaskIndex,
		AttributeGCPGceInstanceHostname,
		AttributeGCPGceInstanceName,
		AttributeGenAiCompletion,
		AttributeGenAiOperationName,
		AttributeGenAiPrompt,
		AttributeGenAiRequestFrequencyPenalty,
		AttributeGenAiRequestMaxTokens,
		AttributeGenAiRequestModel,
		AttributeGenAiRequestPresencePenalty,
		AttributeGenAiRequestStopSequences,
		AttributeGenAiRequestTemperature,
		AttributeGenAiRequestTopK,
		AttributeGenAiRequestTopP,
		AttributeGenAiResponseFinishReasons,
		AttributeGenAiResponseID,
		AttributeGenAiResponseModel,
		AttributeGenAiSystem,
		AttributeGenAiTokenType,
		AttributeGenAiUsageCompletionTokens,
		AttributeGenAiUsageInputTokens,
		AttributeGenAiUsageOutputTokens,
		AttributeGenAiUsagePromptTokens,
		AttributeGoMemoryType,
		AttributeGraphqlDocument,
		AttributeGraphqlOperationName,
		AttributeGraphqlOperationType,
		AttributeHerokuAppID,
		AttributeHerokuReleaseCommit,
		AttributeHerokuReleaseCreationTimestamp,
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
		AttributeHTTPClientIP,
		AttributeHTTPConnectionState,
		AttributeHTTPFlavor,
		AttributeHTTPHost,
		AttributeHTTPMethod,
		AttributeHTTPRequestBodySize,
		AttributeHTTPRequestHeader,
		AttributeHTTPRequestMethod,
		AttributeHTTPRequestMethodOriginal,
		AttributeHTTPRequestResendCount,
		AttributeHTTPRequestSize,
		AttributeHTTPRequestContentLength,
		AttributeHTTPRequestContentLengthUncompressed,
		AttributeHTTPResponseBodySize,
		AttributeHTTPResponseHeader,
		AttributeHTTPResponseSize,
		AttributeHTTPResponseStatusCode,
		AttributeHTTPResponseContentLength,
		AttributeHTTPResponseContentLengthUncompressed,
		AttributeHTTPRoute,
		AttributeHTTPScheme,
		AttributeHTTPServerName,
		AttributeHTTPStatusCode,
		AttributeHTTPTarget,
		AttributeHTTPURL,
		AttributeHTTPUserAgent,
		AttributeHwID,
		AttributeHwName,
		AttributeHwParent,
		AttributeHwState,
		AttributeHwType,
		AttributeIosState,
		AttributeJvmBufferPoolName,
		AttributeJvmGcAction,
		AttributeJvmGcName,
		AttributeJvmMemoryPoolName,
		AttributeJvmMemoryType,
		AttributeJvmThreadDaemon,
		AttributeJvmThreadState,
		AttributeK8SClusterName,
		AttributeK8SClusterUID,
		AttributeK8SContainerName,
		AttributeK8SContainerRestartCount,
		AttributeK8SContainerStatusLastTerminatedReason,
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
		AttributeK8SPodAnnotation,
		AttributeK8SPodLabel,
		AttributeK8SPodLabels,
		AttributeK8SPodName,
		AttributeK8SPodUID,
		AttributeK8SReplicaSetName,
		AttributeK8SReplicaSetUID,
		AttributeK8SStatefulSetName,
		AttributeK8SStatefulSetUID,
		AttributeK8SVolumeName,
		AttributeK8SVolumeType,
		AttributeLinuxMemorySlabState,
		AttributeLogFileName,
		AttributeLogFileNameResolved,
		AttributeLogFilePath,
		AttributeLogFilePathResolved,
		AttributeLogIostream,
		AttributeLogRecordOriginal,
		AttributeLogRecordUID,
		AttributeMessageCompressedSize,
		AttributeMessageID,
		AttributeMessageType,
		AttributeMessageUncompressedSize,
		AttributeMessagingBatchMessageCount,
		AttributeMessagingClientID,
		AttributeMessagingConsumerGroupName,
		AttributeMessagingDestinationAnonymous,
		AttributeMessagingDestinationName,
		AttributeMessagingDestinationPartitionID,
		AttributeMessagingDestinationSubscriptionName,
		AttributeMessagingDestinationTemplate,
		AttributeMessagingDestinationTemporary,
		AttributeMessagingDestinationPublishAnonymous,
		AttributeMessagingDestinationPublishName,
		AttributeMessagingEventhubsConsumerGroup,
		AttributeMessagingEventhubsMessageEnqueuedTime,
		AttributeMessagingGCPPubsubMessageAckDeadline,
		AttributeMessagingGCPPubsubMessageAckID,
		AttributeMessagingGCPPubsubMessageDeliveryAttempt,
		AttributeMessagingGCPPubsubMessageOrderingKey,
		AttributeMessagingKafkaConsumerGroup,
		AttributeMessagingKafkaDestinationPartition,
		AttributeMessagingKafkaMessageKey,
		AttributeMessagingKafkaMessageOffset,
		AttributeMessagingKafkaMessageTombstone,
		AttributeMessagingKafkaOffset,
		AttributeMessagingMessageBodySize,
		AttributeMessagingMessageConversationID,
		AttributeMessagingMessageEnvelopeSize,
		AttributeMessagingMessageID,
		AttributeMessagingOperation,
		AttributeMessagingOperationName,
		AttributeMessagingOperationType,
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
		AttributeNetHostIP,
		AttributeNetHostName,
		AttributeNetHostPort,
		AttributeNetPeerIP,
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
		AttributeNodejsEventloopState,
		AttributeOciManifestDigest,
		AttributeOpentracingRefType,
		AttributeOSBuildID,
		AttributeOSDescription,
		AttributeOSName,
		AttributeOSType,
		AttributeOSVersion,
		AttributeOTelLibraryName,
		AttributeOTelLibraryVersion,
		AttributeOTelScopeName,
		AttributeOTelScopeVersion,
		AttributeOTelStatusCode,
		AttributeOTelStatusDescription,
		AttributeState,
		AttributePeerService,
		AttributePoolName,
		AttributeProcessArgsCount,
		AttributeProcessCommand,
		AttributeProcessCommandArgs,
		AttributeProcessCommandLine,
		AttributeProcessContextSwitchType,
		AttributeProcessCPUState,
		AttributeProcessCreationTime,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessExitCode,
		AttributeProcessExitTime,
		AttributeProcessGroupLeaderPID,
		AttributeProcessInteractive,
		AttributeProcessOwner,
		AttributeProcessPagingFaultType,
		AttributeProcessParentPID,
		AttributeProcessPID,
		AttributeProcessRealUserID,
		AttributeProcessRealUserName,
		AttributeProcessRuntimeDescription,
		AttributeProcessRuntimeName,
		AttributeProcessRuntimeVersion,
		AttributeProcessSavedUserID,
		AttributeProcessSavedUserName,
		AttributeProcessSessionLeaderPID,
		AttributeProcessTitle,
		AttributeProcessUserID,
		AttributeProcessUserName,
		AttributeProcessVpid,
		AttributeProcessWorkingDirectory,
		AttributeProfileFrameType,
		AttributeRPCConnectRPCErrorCode,
		AttributeRPCConnectRPCRequestMetadata,
		AttributeRPCConnectRPCResponseMetadata,
		AttributeRPCGRPCRequestMetadata,
		AttributeRPCGRPCResponseMetadata,
		AttributeRPCGRPCStatusCode,
		AttributeRPCJsonrpcErrorCode,
		AttributeRPCJsonrpcErrorMessage,
		AttributeRPCJsonrpcRequestID,
		AttributeRPCJsonrpcVersion,
		AttributeRPCMessageCompressedSize,
		AttributeRPCMessageID,
		AttributeRPCMessageType,
		AttributeRPCMessageUncompressedSize,
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
		AttributeSignalrConnectionStatus,
		AttributeSignalrTransport,
		AttributeSourceAddress,
		AttributeSourcePort,
		AttributeSystemCPULogicalNumber,
		AttributeSystemCPUState,
		AttributeSystemDevice,
		AttributeSystemFilesystemMode,
		AttributeSystemFilesystemMountpoint,
		AttributeSystemFilesystemState,
		AttributeSystemFilesystemType,
		AttributeSystemMemoryState,
		AttributeSystemNetworkState,
		AttributeSystemPagingDirection,
		AttributeSystemPagingState,
		AttributeSystemPagingType,
		AttributeSystemProcessStatus,
		AttributeSystemProcessesStatus,
		AttributeTelemetryDistroName,
		AttributeTelemetryDistroVersion,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKVersion,
		AttributeTestCaseName,
		AttributeTestCaseResultStatus,
		AttributeTestSuiteName,
		AttributeTestSuiteRunStatus,
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
		AttributeURLTemplate,
		AttributeURLTopLevelDomain,
		AttributeUserEmail,
		AttributeUserFullName,
		AttributeUserHash,
		AttributeUserID,
		AttributeUserName,
		AttributeUserRoles,
		AttributeUserAgentName,
		AttributeUserAgentOriginal,
		AttributeUserAgentVersion,
		AttributeV8jsGcType,
		AttributeV8jsHeapSpaceName,
		AttributeVcsRepositoryChangeID,
		AttributeVcsRepositoryChangeTitle,
		AttributeVcsRepositoryRefName,
		AttributeVcsRepositoryRefRevision,
		AttributeVcsRepositoryRefType,
		AttributeVcsRepositoryURLFull,
		AttributeWebEngineDescription,
		AttributeWebEngineName,
		AttributeWebEngineVersion,
	}
}
