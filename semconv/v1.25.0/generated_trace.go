// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

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
	// The parent Span doesn't depend in any way on the result of the child Span
	AttributeOpentracingRefTypeFollowsFrom = "follows_from"
)

// Span attributes used by non-OTLP exporters to represent OpenTelemetry Span's
// concepts.
const (
	// Name of the code, either &quot;OK&quot; or &quot;ERROR&quot;. MUST NOT be set
	// if the status code is UNSET.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	AttributeOTelStatusCode = "otel.status_code"
	// Description of the Status if it has a value, otherwise not set.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'resource not found'
	AttributeOTelStatusDescription = "otel.status_description"
)

const (
	// The operation has been validated by an Application developer or Operator to have completed successfully
	AttributeOTelStatusCodeOk = "OK"
	// The operation contains an error
	AttributeOTelStatusCodeError = "ERROR"
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

func GetTraceSemanticConventionAttributeNames() []string {
	return []string{
		AttributePeerService,
		AttributeAWSLambdaInvokedARN,
		AttributeOpentracingRefType,
		AttributeOTelStatusCode,
		AttributeOTelStatusDescription,
		AttributeAWSRequestID,
		AttributeAWSS3Bucket,
		AttributeAWSS3CopySource,
		AttributeAWSS3Delete,
		AttributeAWSS3Key,
		AttributeAWSS3PartNumber,
		AttributeAWSS3UploadID,
		AttributeGraphqlDocument,
		AttributeGraphqlOperationName,
		AttributeGraphqlOperationType,
	}
}
