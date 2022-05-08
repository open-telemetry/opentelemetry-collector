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

// A cloud environment (e.g. GCP, Azure, AWS)
const (
	// Name of the cloud provider.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudProvider = "cloud.provider"
	// The cloud account ID the resource is assigned to.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '111111111111', 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudAccountID = "cloud.account.id"
	// The geographical region the resource is running. Refer to your provider's docs
	// to see the available regions, for example AWS regions, Azure regions, or Google
	// Cloud regions.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'us-central1', 'us-east-1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudRegion = "cloud.region"
	// Cloud regions often have multiple, isolated locations known as zones to
	// increase availability. Availability zone represents the zone where the resource
	// is running.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'us-east-1c'
	// Note: Availability zones are called &quot;zones&quot; on Google Cloud.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudAvailabilityZone = "cloud.availability_zone"
	// The cloud platform in use.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Note: The prefix of the service SHOULD match the one specified in
	// cloud.provider.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatform = "cloud.platform"
)

const (
	// Amazon Web Services
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudProviderAWS = "aws"
	// Microsoft Azure
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudProviderAzure = "azure"
	// Google Cloud Platform
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudProviderGCP = "gcp"
)

const (
	// AWS Elastic Compute Cloud
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAWSEC2 = "aws_ec2"
	// AWS Elastic Container Service
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAWSECS = "aws_ecs"
	// AWS Elastic Kubernetes Service
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAWSEKS = "aws_eks"
	// AWS Lambda
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAWSLambda = "aws_lambda"
	// AWS Elastic Beanstalk
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAWSElasticBeanstalk = "aws_elastic_beanstalk"
	// Azure Virtual Machines
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAzureVM = "azure_vm"
	// Azure Container Instances
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAzureContainerInstances = "azure_container_instances"
	// Azure Kubernetes Service
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAzureAKS = "azure_aks"
	// Azure Functions
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAzureFunctions = "azure_functions"
	// Azure App Service
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformAzureAppService = "azure_app_service"
	// Google Cloud Compute Engine (GCE)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformGCPComputeEngine = "gcp_compute_engine"
	// Google Cloud Run
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformGCPCloudRun = "gcp_cloud_run"
	// Google Cloud Kubernetes Engine (GKE)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformGCPKubernetesEngine = "gcp_kubernetes_engine"
	// Google Cloud Functions (GCF)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformGCPCloudFunctions = "gcp_cloud_functions"
	// Google Cloud App Engine (GAE)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeCloudPlatformGCPAppEngine = "gcp_app_engine"
)

// Resources used by AWS Elastic Container Service (ECS).
const (
	// The Amazon Resource Name (ARN) of an ECS container instance.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:ecs:us-
	// west-1:123456789123:container/32624152-9086-4f0e-acae-1a75b14fe4d9'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSContainerARN = "aws.ecs.container.arn"
	// The ARN of an ECS cluster.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSClusterARN = "aws.ecs.cluster.arn"
	// The launch type for an ECS task.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSLaunchtype = "aws.ecs.launchtype"
	// The ARN of an ECS task definition.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:ecs:us-
	// west-1:123456789123:task/10838bed-421f-43ef-870a-f43feacbbb5b'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSTaskARN = "aws.ecs.task.arn"
	// The task definition family this task definition is a member of.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-family'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSTaskFamily = "aws.ecs.task.family"
	// The revision for this task definition.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '8', '26'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSTaskRevision = "aws.ecs.task.revision"
)

const (
	// ec2
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSLaunchtypeEC2 = "ec2"
	// fargate
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSECSLaunchtypeFargate = "fargate"
)

// Resources used by AWS Elastic Kubernetes Service (EKS).
const (
	// The ARN of an EKS cluster.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:ecs:us-west-2:123456789123:cluster/my-cluster'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSEKSClusterARN = "aws.eks.cluster.arn"
)

// Resources specific to Amazon Web Services.
const (
	// The name(s) of the AWS log group(s) an application is writing to.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: '/aws/lambda/my-function', 'opentelemetry-service'
	// Note: Multiple log groups must be supported for cases like multi-container
	// applications, where a single application has sidecar containers, and each write
	// to their own log group.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSLogGroupNames = "aws.log.group.names"
	// The Amazon Resource Name(s) (ARN) of the AWS log group(s).
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:*'
	// Note: See the log group ARN format documentation.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSLogGroupARNs = "aws.log.group.arns"
	// The name(s) of the AWS log stream(s) an application is writing to.
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: 'logs/main/10838bed-421f-43ef-870a-f43feacbbb5b'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSLogStreamNames = "aws.log.stream.names"
	// The ARN(s) of the AWS log stream(s).
	//
	// Type: string[]
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:logs:us-west-1:123456789012:log-group:/aws/my/group:log-
	// stream:logs/main/10838bed-421f-43ef-870a-f43feacbbb5b'
	// Note: See the log stream ARN format documentation. One log group can contain
	// several log streams, so these ARNs necessarily identify both a log group and a
	// log stream.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeAWSLogStreamARNs = "aws.log.stream.arns"
)

// A container instance.
const (
	// Container name.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-autoconf'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeContainerName = "container.name"
	// Container ID. Usually a UUID, as for example used to identify Docker
	// containers. The UUID might be abbreviated.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'a3bf90e006b2'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeContainerID = "container.id"
	// The container runtime managing this container.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'docker', 'containerd', 'rkt'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeContainerRuntime = "container.runtime"
	// Name of the image the container was built on.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'gcr.io/opentelemetry/operator'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeContainerImageName = "container.image.name"
	// Container image tag.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '0.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeContainerImageTag = "container.image.tag"
)

// The software deployment.
const (
	// Name of the deployment environment (aka deployment tier).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'staging', 'production'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDeploymentEnvironment = "deployment.environment"
)

// The device on which the process represented by this resource is running.
const (
	// A unique identifier representing the device
	//
	// Type: string
	// Required: No
	// Stability: stable
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
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDeviceID = "device.id"
	// The model identifier for the device
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'iPhone3,4', 'SM-G920F'
	// Note: It's recommended this value represents a machine readable version of the
	// model identifier rather than the market or consumer-friendly name of the
	// device.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDeviceModelIdentifier = "device.model.identifier"
	// The marketing name for the device model
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'iPhone 6s Plus', 'Samsung Galaxy S6'
	// Note: It's recommended this value represents a human readable version of the
	// device model rather than a machine readable alternative.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeDeviceModelName = "device.model.name"
)

// A serverless instance.
const (
	// The name of the single function that this runtime instance executes.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'my-function'
	// Note: This is the name of the function as configured/deployed on the FaaS
	// platform and is usually different from the name of the callback function (which
	// may be stored in the code.namespace/code.function span attributes).
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSName = "faas.name"
	// The unique ID of the single function that this runtime instance executes.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'arn:aws:lambda:us-west-2:123456789012:function:my-function'
	// Note: Depending on the cloud provider, use:<ul>
	// <li>AWS Lambda: The function ARN.</li>
	// </ul>
	// Take care not to use the &quot;invoked ARN&quot; directly but replace any
	// alias suffix with the resolved function version, as the same runtime instance
	// may be invokable with multiple
	// different aliases.<ul>
	// <li>GCP: The URI of the resource</li>
	// <li>Azure: The Fully Qualified Resource ID.</li>
	// </ul>
	// On some providers, it may not be possible to determine the full ID at startup,
	// which is why this field cannot be made required. For example, on AWS the
	// account ID
	// part of the ARN is not available without calling another AWS API
	// which may be deemed too slow for a short-running lambda function.
	// As an alternative, consider setting faas.id as a span attribute instead.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSID = "faas.id"
	// The immutable version of the function being executed.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '26', 'pinkfroid-00002'
	// Note: Depending on the cloud provider and platform, use:<ul>
	// <li>AWS Lambda: The function version
	// (an integer represented as a decimal string).</li>
	// <li>Google Cloud Run: The revision
	// (i.e., the function name plus the revision suffix).</li>
	// <li>Google Cloud Functions: The value of the
	// K_REVISION environment variable.</li>
	// <li>Azure Functions: Not applicable. Do not set this attribute.</li>
	// </ul>
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSVersion = "faas.version"
	// The execution environment ID as a string, that will be potentially reused for
	// other invocations to the same function/function version.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de'
	// Note: <ul>
	// <li>AWS Lambda: Use the (full) log stream name.</li>
	// </ul>
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSInstance = "faas.instance"
	// The amount of memory available to the serverless function in MiB.
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 128
	// Note: It's recommended to set this attribute since e.g. too little memory can
	// easily stop a Java AWS Lambda function from working correctly. On AWS Lambda,
	// the environment variable AWS_LAMBDA_FUNCTION_MEMORY_SIZE provides this
	// information.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeFaaSMaxMemory = "faas.max_memory"
)

// A host is defined as a general computing instance.
const (
	// Unique host ID. For Cloud, this must be the instance_id assigned by the cloud
	// provider.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-test'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostID = "host.id"
	// Name of the host. On Unix systems, it may contain what the hostname command
	// returns, or the fully qualified hostname, or another name specified by the
	// user.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-test'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostName = "host.name"
	// Type of host. For Cloud, this must be the machine type.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'n1-standard-1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostType = "host.type"
	// The CPU architecture the host system is running on.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArch = "host.arch"
	// Name of the VM image or OS install the host was instantiated from.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'infra-ami-eks-worker-node-7d4ec78312', 'CentOS-8-x86_64-1905'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostImageName = "host.image.name"
	// VM image ID. For Cloud, this value is from the provider.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'ami-07b06b442921831e5'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostImageID = "host.image.id"
	// The version string of the VM image as defined in Version Attributes.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '0.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostImageVersion = "host.image.version"
)

const (
	// AMD64
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchAMD64 = "amd64"
	// ARM32
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchARM32 = "arm32"
	// ARM64
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchARM64 = "arm64"
	// Itanium
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchIA64 = "ia64"
	// 32-bit PowerPC
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchPPC32 = "ppc32"
	// 64-bit PowerPC
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchPPC64 = "ppc64"
	// 32-bit x86
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeHostArchX86 = "x86"
)

// A Kubernetes Cluster.
const (
	// The name of the cluster.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-cluster'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SClusterName = "k8s.cluster.name"
)

// A Kubernetes Node object.
const (
	// The name of the Node.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'node-1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SNodeName = "k8s.node.name"
	// The UID of the Node.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '1eb3a0c6-0477-4080-a9cb-0cb7db65c6a2'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SNodeUID = "k8s.node.uid"
)

// A Kubernetes Namespace.
const (
	// The name of the namespace that the pod is running in.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'default'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SNamespaceName = "k8s.namespace.name"
)

// A Kubernetes Pod object.
const (
	// The UID of the Pod.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SPodUID = "k8s.pod.uid"
	// The name of the Pod.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry-pod-autoconf'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SPodName = "k8s.pod.name"
)

// A container in a [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/#pod-templates).
const (
	// The name of the Container in a Pod template.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'redis'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SContainerName = "k8s.container.name"
)

// A Kubernetes ReplicaSet object.
const (
	// The UID of the ReplicaSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SReplicaSetUID = "k8s.replicaset.uid"
	// The name of the ReplicaSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SReplicaSetName = "k8s.replicaset.name"
)

// A Kubernetes Deployment object.
const (
	// The UID of the Deployment.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SDeploymentUID = "k8s.deployment.uid"
	// The name of the Deployment.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SDeploymentName = "k8s.deployment.name"
)

// A Kubernetes StatefulSet object.
const (
	// The UID of the StatefulSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SStatefulSetUID = "k8s.statefulset.uid"
	// The name of the StatefulSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SStatefulSetName = "k8s.statefulset.name"
)

// A Kubernetes DaemonSet object.
const (
	// The UID of the DaemonSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SDaemonSetUID = "k8s.daemonset.uid"
	// The name of the DaemonSet.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SDaemonSetName = "k8s.daemonset.name"
)

// A Kubernetes Job object.
const (
	// The UID of the Job.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SJobUID = "k8s.job.uid"
	// The name of the Job.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SJobName = "k8s.job.name"
)

// A Kubernetes CronJob object.
const (
	// The UID of the CronJob.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SCronJobUID = "k8s.cronjob.uid"
	// The name of the CronJob.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeK8SCronJobName = "k8s.cronjob.name"
)

// The operating system (OS) on which the process represented by this resource is running.
const (
	// The operating system type.
	//
	// Type: Enum
	// Required: Always
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSType = "os.type"
	// Human readable (not intended to be parsed) OS version information, like e.g.
	// reported by ver or lsb_release -a commands.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Microsoft Windows [Version 10.0.18363.778]', 'Ubuntu 18.04.1 LTS'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSDescription = "os.description"
	// Human readable operating system name.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'iOS', 'Android', 'Ubuntu'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSName = "os.name"
	// The version string of the operating system as defined in Version Attributes.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '14.2.1', '18.04.1'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSVersion = "os.version"
)

const (
	// Microsoft Windows
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeWindows = "windows"
	// Linux
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeLinux = "linux"
	// Apple Darwin
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeDarwin = "darwin"
	// FreeBSD
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeFreeBSD = "freebsd"
	// NetBSD
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeNetBSD = "netbsd"
	// OpenBSD
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeOpenBSD = "openbsd"
	// DragonFly BSD
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeDragonflyBSD = "dragonflybsd"
	// HP-UX (Hewlett Packard Unix)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeHPUX = "hpux"
	// AIX (Advanced Interactive eXecutive)
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeAIX = "aix"
	// Oracle Solaris
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeSolaris = "solaris"
	// IBM z/OS
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeOSTypeZOS = "z_os"
)

// An operating system process.
const (
	// Process identifier (PID).
	//
	// Type: int
	// Required: No
	// Stability: stable
	// Examples: 1234
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessPID = "process.pid"
	// The name of the process executable. On Linux based systems, can be set to the
	// Name in proc/[pid]/status. On Windows, can be set to the base name of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Required: See below
	// Stability: stable
	// Examples: 'otelcol'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessExecutableName = "process.executable.name"
	// The full path to the process executable. On Linux based systems, can be set to
	// the target of proc/[pid]/exe. On Windows, can be set to the result of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Required: See below
	// Stability: stable
	// Examples: '/usr/bin/cmd/otelcol'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessExecutablePath = "process.executable.path"
	// The command used to launch the process (i.e. the command name). On Linux based
	// systems, can be set to the zeroth string in proc/[pid]/cmdline. On Windows, can
	// be set to the first parameter extracted from GetCommandLineW.
	//
	// Type: string
	// Required: See below
	// Stability: stable
	// Examples: 'cmd/otelcol'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessCommand = "process.command"
	// The full command used to launch the process as a single string representing the
	// full command. On Windows, can be set to the result of GetCommandLineW. Do not
	// set this if you have to assemble it just for monitoring; use
	// process.command_args instead.
	//
	// Type: string
	// Required: See below
	// Stability: stable
	// Examples: 'C:\\cmd\\otecol --config="my directory\\config.yaml"'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessCommandLine = "process.command_line"
	// All the command arguments (including the command/executable itself) as received
	// by the process. On Linux-based systems (and some other Unixoid systems
	// supporting procfs), can be set according to the list of null-delimited strings
	// extracted from proc/[pid]/cmdline. For libc-based executables, this would be
	// the full argv vector passed to main.
	//
	// Type: string[]
	// Required: See below
	// Stability: stable
	// Examples: 'cmd/otecol', '--config=config.yaml'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessCommandArgs = "process.command_args"
	// The username of the user that owns the process.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'root'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessOwner = "process.owner"
)

// The single (language) runtime instance which is monitored.
const (
	// The name of the runtime of this process. For compiled native binaries, this
	// SHOULD be the name of the compiler.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'OpenJDK Runtime Environment'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessRuntimeName = "process.runtime.name"
	// The version of the runtime of this process, as returned by the runtime without
	// modification.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '14.0.2'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessRuntimeVersion = "process.runtime.version"
	// An additional description about the runtime of the process, for example a
	// specific vendor customization of the runtime environment.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Eclipse OpenJ9 Eclipse OpenJ9 VM openj9-0.21.0'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeProcessRuntimeDescription = "process.runtime.description"
)

// A service instance.
const (
	// Logical name of the service.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'shoppingcart'
	// Note: MUST be the same for all instances of horizontally scaled services. If
	// the value was not specified, SDKs MUST fallback to unknown_service:
	// concatenated with process.executable.name, e.g. unknown_service:bash. If
	// process.executable.name is not available, the value MUST be set to
	// unknown_service.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeServiceName = "service.name"
	// A namespace for service.name.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'Shop'
	// Note: A string value having a meaning that helps to distinguish a group of
	// services, for example the team name that owns a group of services. service.name
	// is expected to be unique within the same namespace. If service.namespace is not
	// specified in the Resource then service.name is expected to be unique for all
	// services that have no explicit namespace defined (so the empty/unspecified
	// namespace is simply one more valid namespace). Zero-length namespace string is
	// assumed equal to unspecified namespace.
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeServiceNamespace = "service.namespace"
	// The string ID of the service instance.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '627cc493-f310-47de-96bd-71410b7dec09'
	// Note: MUST be unique for each instance of the same
	// service.namespace,service.name pair (in other words
	// service.namespace,service.name,service.instance.id triplet MUST be globally
	// unique). The ID helps to distinguish instances of the same service that exist
	// at the same time (e.g. instances of a horizontally scaled service). It is
	// preferable for the ID to be persistent and stay the same for the lifetime of
	// the service instance, however it is acceptable that the ID is ephemeral and
	// changes during important lifetime events for the service (e.g. service
	// restarts). If the service has no inherent unique ID that can be used as the
	// value of this attribute it is recommended to generate a random Version 1 or
	// Version 4 RFC 4122 UUID (services aiming for reproducible UUIDs may also use
	// Version 5, see RFC 4122 for more recommendations).
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeServiceInstanceID = "service.instance.id"
	// The version string of the service API or implementation.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '2.0.0'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeServiceVersion = "service.version"
)

// The telemetry SDK used to capture data recorded by the instrumentation libraries.
const (
	// The name of the telemetry SDK as defined above.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'opentelemetry'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKName = "telemetry.sdk.name"
	// The language of the telemetry SDK.
	//
	// Type: Enum
	// Required: No
	// Stability: stable
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguage = "telemetry.sdk.language"
	// The version string of the telemetry SDK.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '1.2.3'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKVersion = "telemetry.sdk.version"
	// The version string of the auto instrumentation agent, if used.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '1.2.3'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetryAutoVersion = "telemetry.auto.version"
)

const (
	// cpp
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageCPP = "cpp"
	// dotnet
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageDotnet = "dotnet"
	// erlang
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageErlang = "erlang"
	// go
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageGo = "go"
	// java
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageJava = "java"
	// nodejs
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageNodejs = "nodejs"
	// php
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguagePHP = "php"
	// python
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguagePython = "python"
	// ruby
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageRuby = "ruby"
	// webjs
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeTelemetrySDKLanguageWebjs = "webjs"
)

// Resource describing the packaged software running the application code. Web engines are typically executed using process.runtime.
const (
	// The name of the web engine.
	//
	// Type: string
	// Required: Always
	// Stability: stable
	// Examples: 'WildFly'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeWebEngineName = "webengine.name"
	// The version of the web engine.
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: '21.0.0'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeWebEngineVersion = "webengine.version"
	// Additional description of the web engine (e.g. detailed version and edition
	// information).
	//
	// Type: string
	// Required: No
	// Stability: stable
	// Examples: 'WildFly Full 21.0.0.Final (WildFly Core 13.0.1.Final) - 2.2.2.Final'
	// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
	AttributeWebEngineDescription = "webengine.description"
)

// Deprecated: [v0.50.0] Use go.opentelemetry.io/collector/semconv module.
func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeCloudProvider,
		AttributeCloudAccountID,
		AttributeCloudRegion,
		AttributeCloudAvailabilityZone,
		AttributeCloudPlatform,
		AttributeAWSECSContainerARN,
		AttributeAWSECSClusterARN,
		AttributeAWSECSLaunchtype,
		AttributeAWSECSTaskARN,
		AttributeAWSECSTaskFamily,
		AttributeAWSECSTaskRevision,
		AttributeAWSEKSClusterARN,
		AttributeAWSLogGroupNames,
		AttributeAWSLogGroupARNs,
		AttributeAWSLogStreamNames,
		AttributeAWSLogStreamARNs,
		AttributeContainerName,
		AttributeContainerID,
		AttributeContainerRuntime,
		AttributeContainerImageName,
		AttributeContainerImageTag,
		AttributeDeploymentEnvironment,
		AttributeDeviceID,
		AttributeDeviceModelIdentifier,
		AttributeDeviceModelName,
		AttributeFaaSName,
		AttributeFaaSID,
		AttributeFaaSVersion,
		AttributeFaaSInstance,
		AttributeFaaSMaxMemory,
		AttributeHostID,
		AttributeHostName,
		AttributeHostType,
		AttributeHostArch,
		AttributeHostImageName,
		AttributeHostImageID,
		AttributeHostImageVersion,
		AttributeK8SClusterName,
		AttributeK8SNodeName,
		AttributeK8SNodeUID,
		AttributeK8SNamespaceName,
		AttributeK8SPodUID,
		AttributeK8SPodName,
		AttributeK8SContainerName,
		AttributeK8SReplicaSetUID,
		AttributeK8SReplicaSetName,
		AttributeK8SDeploymentUID,
		AttributeK8SDeploymentName,
		AttributeK8SStatefulSetUID,
		AttributeK8SStatefulSetName,
		AttributeK8SDaemonSetUID,
		AttributeK8SDaemonSetName,
		AttributeK8SJobUID,
		AttributeK8SJobName,
		AttributeK8SCronJobUID,
		AttributeK8SCronJobName,
		AttributeOSType,
		AttributeOSDescription,
		AttributeOSName,
		AttributeOSVersion,
		AttributeProcessPID,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessCommand,
		AttributeProcessCommandLine,
		AttributeProcessCommandArgs,
		AttributeProcessOwner,
		AttributeProcessRuntimeName,
		AttributeProcessRuntimeVersion,
		AttributeProcessRuntimeDescription,
		AttributeServiceName,
		AttributeServiceNamespace,
		AttributeServiceInstanceID,
		AttributeServiceVersion,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKVersion,
		AttributeTelemetryAutoVersion,
		AttributeWebEngineName,
		AttributeWebEngineVersion,
		AttributeWebEngineDescription,
	}
}
