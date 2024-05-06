// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// Resources used by AWS Elastic Container Service (ECS).
const (
	// The ID of a running ECS task. The ID MUST be extracted from task.arn.
	//
	// Type: string
	// Requirement Level: Conditionally Required - If and only if `task.arn` is
	// populated.
	// Stability: experimental
	// Examples: '10838bed-421f-43ef-870a-f43feacbbb5b',
	// '23ebb8ac-c18f-46c6-8bbe-d55d0e37cfbd'
	AttributeAWSECSTaskID = "aws.ecs.task.id"
	// The ARN of an ECS cluster.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster'
	AttributeAWSECSClusterARN = "aws.ecs.cluster.arn"
	// The Amazon Resource Name (ARN) of an ECS container instance.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:ecs:us-
	// west-1:123456789123:container/32624152-9086-4f0e-acae-1a75b14fe4d9'
	AttributeAWSECSContainerARN = "aws.ecs.container.arn"
	// The launch type for an ECS task.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeAWSECSLaunchtype = "aws.ecs.launchtype"
	// The ARN of a running ECS task.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:ecs:us-
	// west-1:123456789123:task/10838bed-421f-43ef-870a-f43feacbbb5b',
	// 'arn:aws:ecs:us-west-1:123456789123:task/my-cluster/task-
	// id/23ebb8ac-c18f-46c6-8bbe-d55d0e37cfbd'
	AttributeAWSECSTaskARN = "aws.ecs.task.arn"
	// The family name of the ECS task definition used to create the ECS task.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-family'
	AttributeAWSECSTaskFamily = "aws.ecs.task.family"
	// The revision for the task definition used to create the ECS task.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '8', '26'
	AttributeAWSECSTaskRevision = "aws.ecs.task.revision"
)

const (
	// ec2
	AttributeAWSECSLaunchtypeEC2 = "ec2"
	// fargate
	AttributeAWSECSLaunchtypeFargate = "fargate"
)

// Resources used by AWS Elastic Kubernetes Service (EKS).
const (
	// The ARN of an EKS cluster.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster'
	AttributeAWSEKSClusterARN = "aws.eks.cluster.arn"
)

// Resources specific to Amazon Web Services.
const (
	// The Amazon Resource Name(s) (ARN) of the AWS log group(s).
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:*'
	// Note: See the log group ARN format documentation.
	AttributeAWSLogGroupARNs = "aws.log.group.arns"
	// The name(s) of the AWS log group(s) an application is writing to.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '/aws/lambda/my-function', 'opentelemetry-service'
	// Note: Multiple log groups must be supported for cases like multi-container
	// applications, where a single application has sidecar containers, and each write
	// to their own log group.
	AttributeAWSLogGroupNames = "aws.log.group.names"
	// The ARN(s) of the AWS log stream(s).
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:log-
	// stream:logs/main/10838bed-421f-43ef-870a-f43feacbbb5b'
	// Note: See the log stream ARN format documentation. One log group can contain
	// several log streams, so these ARNs necessarily identify both a log group and a
	// log stream.
	AttributeAWSLogStreamARNs = "aws.log.stream.arns"
	// The name(s) of the AWS log stream(s) an application is writing to.
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'logs/main/10838bed-421f-43ef-870a-f43feacbbb5b'
	AttributeAWSLogStreamNames = "aws.log.stream.names"
)

// Heroku dyno metadata
const (
	// Unique identifier for the application
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2daa2797-e42b-4624-9322-ec3f968df4da'
	AttributeHerokuAppID = "heroku.app.id"
	// Commit hash for the current release
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'e6134959463efd8966b20e75b913cafe3f5ec'
	AttributeHerokuReleaseCommit = "heroku.release.commit"
	// Time and date the release was created
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2022-10-23T18:00:42Z'
	AttributeHerokuReleaseCreationTimestamp = "heroku.release.creation_timestamp"
)

// Resource describing the packaged software running the application code. Web
// engines are typically executed using process.runtime.
const (
	// The name of the web engine.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'WildFly'
	AttributeWebEngineName = "webengine.name"
	// Additional description of the web engine (e.g. detailed version and edition
	// information).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'WildFly Full 21.0.0.Final (WildFly Core 13.0.1.Final) - 2.2.2.Final'
	AttributeWebEngineDescription = "webengine.description"
	// The version of the web engine.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '21.0.0'
	AttributeWebEngineVersion = "webengine.version"
)

// Attributes used by non-OTLP exporters to represent OpenTelemetry Scope's
// concepts.
const (
	// The name of the instrumentation scope - (InstrumentationScope.Name in OTLP).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: 'io.opentelemetry.contrib.mongodb'
	AttributeOTelScopeName = "otel.scope.name"
	// The version of the instrumentation scope - (InstrumentationScope.Version in
	// OTLP).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: stable
	// Examples: '1.0.0'
	AttributeOTelScopeVersion = "otel.scope.version"
)

// Span attributes used by non-OTLP exporters to represent OpenTelemetry
// Scope's concepts.
const (
	// None
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: use the `otel.scope.name` attribute.
	// Examples: 'io.opentelemetry.contrib.mongodb'
	AttributeOTelLibraryName = "otel.library.name"
	// None
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Deprecated: use the `otel.scope.version` attribute.
	// Examples: '1.0.0'
	AttributeOTelLibraryVersion = "otel.library.version"
)

func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeAWSECSTaskID,
		AttributeAWSECSClusterARN,
		AttributeAWSECSContainerARN,
		AttributeAWSECSLaunchtype,
		AttributeAWSECSTaskARN,
		AttributeAWSECSTaskFamily,
		AttributeAWSECSTaskRevision,
		AttributeAWSEKSClusterARN,
		AttributeAWSLogGroupARNs,
		AttributeAWSLogGroupNames,
		AttributeAWSLogStreamARNs,
		AttributeAWSLogStreamNames,
		AttributeHerokuAppID,
		AttributeHerokuReleaseCommit,
		AttributeHerokuReleaseCreationTimestamp,
		AttributeWebEngineName,
		AttributeWebEngineDescription,
		AttributeWebEngineVersion,
		AttributeOTelScopeName,
		AttributeOTelScopeVersion,
		AttributeOTelLibraryName,
		AttributeOTelLibraryVersion,
	}
}
