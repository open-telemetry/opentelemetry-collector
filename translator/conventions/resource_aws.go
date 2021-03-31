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

// OpenTelemetry semantic convention values for AWS-specific resource attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud_provider/aws/README.md
const (
	AttributeAWSECSContainerARN = "aws.ecs.container.arn"
	AttributeAWSECSClusterARN   = "aws.ecs.cluster.arn"
	AttributeAWSECSLaunchType   = "aws.ecs.launchtype"
	AttributeAWSECSTaskARN      = "aws.ecs.task.arn"
	AttributeAWSECSTaskFamily   = "aws.ecs.task.family"
	AttributeAWSECSTaskRevision = "aws.ecs.task.revision"
	AttributeAWSLogGroupNames   = "aws.log.group.names"
	AttributeAWSLogGroupARNs    = "aws.log.group.arns"
	AttributeAWSLogStreamNames  = "aws.log.stream.names"
	AttributeAWSLogStreamARNs   = "aws.log.stream.arns"
)

// OpenTelemetry Semantic Convention values for Resource attribute "aws.ecs.launchtype" values.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud_provider/aws/ecs.md
const (
	AttributeAWSECSLaunchTypeEC2     = "ec2"
	AttributeAWSECSLaunchTypeFargate = "fargate"
)
