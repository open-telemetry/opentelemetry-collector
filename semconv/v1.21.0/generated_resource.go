// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// The web browser in which the application represented by the resource is
// running. The `browser.*` attributes MUST be used only for resources that
// represent applications running in a web browser (regardless of whether
// running on a mobile or desktop device).
const (
	// Array of brand name and version separated by a space
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: ' Not A;Brand 99', 'Chromium 99', 'Chrome 99'
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.brands).
	AttributeBrowserBrands = "browser.brands"
	// Preferred language of the user using the browser
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'en', 'en-US', 'fr', 'fr-FR'
	// Note: This value is intended to be taken from the Navigator API
	// navigator.language.
	AttributeBrowserLanguage = "browser.language"
	// A boolean that is true if the browser is running on a mobile device
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: experimental
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.mobile). If unavailable, this attribute SHOULD be left
	// unset.
	AttributeBrowserMobile = "browser.mobile"
	// The platform on which the browser is running
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Windows', 'macOS', 'Android'
	// Note: This value is intended to be taken from the UA client hints API
	// (navigator.userAgentData.platform). If unavailable, the legacy
	// navigator.platform API SHOULD NOT be used instead and this attribute SHOULD be
	// left unset in order for the values to be consistent.
	// The list of possible values is defined in the W3C User-Agent Client Hints
	// specification. Note that some (but not all) of these values can overlap with
	// values in the os.type and os.name attributes. However, for consistency, the
	// values in the browser.platform attribute should capture the exact value that
	// the user agent provides.
	AttributeBrowserPlatform = "browser.platform"
)

// A cloud environment (e.g. GCP, Azure, AWS)
const (
	// The cloud account ID the resource is assigned to.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '111111111111', 'opentelemetry'
	AttributeCloudAccountID = "cloud.account.id"
	// Cloud regions often have multiple, isolated locations known as zones to
	// increase availability. Availability zone represents the zone where the resource
	// is running.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'us-east-1c'
	// Note: Availability zones are called &quot;zones&quot; on Alibaba Cloud and
	// Google Cloud.
	AttributeCloudAvailabilityZone = "cloud.availability_zone"
	// The cloud platform in use.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	// Note: The prefix of the service SHOULD match the one specified in
	// cloud.provider.
	AttributeCloudPlatform = "cloud.platform"
	// Name of the cloud provider.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeCloudProvider = "cloud.provider"
	// The geographical region the resource is running.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'us-central1', 'us-east-1'
	// Note: Refer to your provider's docs to see the available regions, for example
	// Alibaba Cloud regions, AWS regions, Azure regions, Google Cloud regions, or
	// Tencent Cloud regions.
	AttributeCloudRegion = "cloud.region"
	// Cloud provider-specific native identifier of the monitored cloud resource (e.g.
	// an ARN on AWS, a fully qualified resource ID on Azure, a full resource name on
	// GCP)
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:lambda:REGION:ACCOUNT_ID:function:my-function', '//run.googl
	// eapis.com/projects/PROJECT_ID/locations/LOCATION_ID/services/SERVICE_ID', '/sub
	// scriptions/<SUBSCRIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/sites
	// /<FUNCAPP>/functions/<FUNC>'
	// Note: On some cloud providers, it may not be possible to determine the full ID
	// at startup,
	// so it may be necessary to set cloud.resource_id as a span attribute instead.The
	// exact value to use for cloud.resource_id depends on the cloud provider.
	// The following well-known definitions MUST be used if you set this attribute and
	// they apply:<ul>
	// <li>AWS Lambda: The function ARN.
	// Take care not to use the &quot;invoked ARN&quot; directly but replace any
	// alias suffix
	// with the resolved function version, as the same runtime instance may be
	// invocable with
	// multiple different aliases.</li>
	// <li>GCP: The URI of the resource</li>
	// <li>Azure: The Fully Qualified Resource ID of the invoked function,
	// not the function app, having the form
	// /subscriptions/<SUBSCRIPTION_GUID>/resourceGroups/<RG>/providers/Microsoft.Web/s
	// ites/<FUNCAPP>/functions/<FUNC>.
	// This means that a span attribute MUST be used, as an Azure function app can
	// host multiple functions that would usually share
	// a TracerProvider.</li>
	// </ul>
	AttributeCloudResourceID = "cloud.resource_id"
)

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

// Resources used by AWS Elastic Container Service (ECS).
const (
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
	// The ARN of an ECS task definition.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'arn:aws:ecs:us-
	// west-1:123456789123:task/10838bed-421f-43ef-870a-f43feacbbb5b'
	AttributeAWSECSTaskARN = "aws.ecs.task.arn"
	// The task definition family this task definition is a member of.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-family'
	AttributeAWSECSTaskFamily = "aws.ecs.task.family"
	// The revision for this task definition.
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

// Resource used by Google Cloud Run.
const (
	// The name of the Cloud Run execution being run for the Job, as set by the
	// CLOUD_RUN_EXECUTION environment variable.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'job-name-xxxx', 'sample-job-mdw84'
	AttributeGCPCloudRunJobExecution = "gcp.cloud_run.job.execution"
	// The index for a task within an execution as provided by the
	// CLOUD_RUN_TASK_INDEX environment variable.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 1
	AttributeGCPCloudRunJobTaskIndex = "gcp.cloud_run.job.task_index"
)

// Resources used by Google Compute Engine (GCE).
const (
	// The hostname of a GCE instance. This is the full value of the default or custom
	// hostname.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-host1234.example.com', 'sample-vm.us-west1-b.c.my-
	// project.internal'
	AttributeGCPGceInstanceHostname = "gcp.gce.instance.hostname"
	// The instance name of a GCE instance. This is the value provided by host.name,
	// the visible name of the instance in the Cloud Console UI, and the prefix for
	// the default hostname of the instance as defined by the default internal DNS
	// name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'instance-1', 'my-vm-name'
	AttributeGCPGceInstanceName = "gcp.gce.instance.name"
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

// A container instance.
const (
	// The command used to run the container (i.e. the command name).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol'
	// Note: If using embedded credentials or sensitive data, it is recommended to
	// remove them to prevent potential leakage.
	AttributeContainerCommand = "container.command"
	// All the command arguments (including the command/executable itself) run by the
	// container. [2]
	//
	// Type: string[]
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol, --config, config.yaml'
	AttributeContainerCommandArgs = "container.command_args"
	// The full command run by the container as a single string representing the full
	// command. [2]
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'otelcontribcol --config config.yaml'
	AttributeContainerCommandLine = "container.command_line"
	// Container ID. Usually a UUID, as for example used to identify Docker
	// containers. The UUID might be abbreviated.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'a3bf90e006b2'
	AttributeContainerID = "container.id"
	// Runtime specific image identifier. Usually a hash algorithm followed by a UUID.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples:
	// 'sha256:19c92d0a00d1b66d897bceaa7319bee0dd38a10a851c60bcec9474aa3f01e50f'
	// Note: Docker defines a sha256 of the image id; container.image.id corresponds
	// to the Image field from the Docker container inspect API endpoint.
	// K8S defines a link to the container registry repository with digest "imageID":
	// "registry.azurecr.io /namespace/service/dockerfile@sha256:bdeabd40c3a8a492eaf9e
	// 8e44d0ebbb84bac7ee25ac0cf8a7159d25f62555625".
	// OCI defines a digest of manifest.
	AttributeContainerImageID = "container.image.id"
	// Name of the image the container was built on.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'gcr.io/opentelemetry/operator'
	AttributeContainerImageName = "container.image.name"
	// Container image tag.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0.1'
	AttributeContainerImageTag = "container.image.tag"
	// Container name used by container runtime.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-autoconf'
	AttributeContainerName = "container.name"
	// The container runtime managing this container.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'docker', 'containerd', 'rkt'
	AttributeContainerRuntime = "container.runtime"
)

// The software deployment.
const (
	// Name of the deployment environment (aka deployment tier).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'staging', 'production'
	AttributeDeploymentEnvironment = "deployment.environment"
)

// The device on which the process represented by this resource is running.
const (
	// A unique identifier representing the device
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
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
	AttributeDeviceID = "device.id"
	// The name of the device manufacturer
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Apple', 'Samsung'
	// Note: The Android OS provides this field via Build. iOS apps SHOULD hardcode
	// the value Apple.
	AttributeDeviceManufacturer = "device.manufacturer"
	// The model identifier for the device
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iPhone3,4', 'SM-G920F'
	// Note: It's recommended this value represents a machine readable version of the
	// model identifier rather than the market or consumer-friendly name of the
	// device.
	AttributeDeviceModelIdentifier = "device.model.identifier"
	// The marketing name for the device model
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iPhone 6s Plus', 'Samsung Galaxy S6'
	// Note: It's recommended this value represents a human readable version of the
	// device model rather than a machine readable alternative.
	AttributeDeviceModelName = "device.model.name"
)

// A serverless instance.
const (
	// The execution environment ID as a string, that will be potentially reused for
	// other invocations to the same function/function version.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2021/06/28/[$LATEST]2f399eb14537447da05ab2a2e39309de'
	// Note: <ul>
	// <li>AWS Lambda: Use the (full) log stream name.</li>
	// </ul>
	AttributeFaaSInstance = "faas.instance"
	// The amount of memory available to the serverless function converted to Bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 134217728
	// Note: It's recommended to set this attribute since e.g. too little memory can
	// easily stop a Java AWS Lambda function from working correctly. On AWS Lambda,
	// the environment variable AWS_LAMBDA_FUNCTION_MEMORY_SIZE provides this
	// information (which must be multiplied by 1,048,576).
	AttributeFaaSMaxMemory = "faas.max_memory"
	// The name of the single function that this runtime instance executes.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'my-function', 'myazurefunctionapp/some-function-name'
	// Note: This is the name of the function as configured/deployed on the FaaS
	// platform and is usually different from the name of the callback
	// function (which may be stored in the
	// code.namespace/code.function
	// span attributes).For some cloud providers, the above definition is ambiguous.
	// The following
	// definition of function name MUST be used for this attribute
	// (and consequently the span name) for the listed cloud providers/products:<ul>
	// <li>Azure:  The full name <FUNCAPP>/<FUNC>, i.e., function app name
	// followed by a forward slash followed by the function name (this form
	// can also be seen in the resource JSON for the function).
	// This means that a span attribute MUST be used, as an Azure function
	// app can host multiple functions that would usually share
	// a TracerProvider (see also the cloud.resource_id attribute).</li>
	// </ul>
	AttributeFaaSName = "faas.name"
	// The immutable version of the function being executed.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '26', 'pinkfroid-00002'
	// Note: Depending on the cloud provider and platform, use:<ul>
	// <li>AWS Lambda: The function version
	// (an integer represented as a decimal string).</li>
	// <li>Google Cloud Run (Services): The revision
	// (i.e., the function name plus the revision suffix).</li>
	// <li>Google Cloud Functions: The value of the
	// K_REVISION environment variable.</li>
	// <li>Azure Functions: Not applicable. Do not set this attribute.</li>
	// </ul>
	AttributeFaaSVersion = "faas.version"
)

// A host is defined as a computing instance. For example, physical servers,
// virtual machines, switches or disk array.
const (
	// The CPU architecture the host system is running on.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeHostArch = "host.arch"
	// Unique host ID. For Cloud, this must be the instance_id assigned by the cloud
	// provider. For non-containerized systems, this should be the machine-id. See the
	// table below for the sources to use to determine the machine-id based on
	// operating system.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'fdbf79e8af94cb7f9e8df36789187052'
	AttributeHostID = "host.id"
	// VM image ID or host OS image ID. For Cloud, this value is from the provider.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'ami-07b06b442921831e5'
	AttributeHostImageID = "host.image.id"
	// Name of the VM image or OS install the host was instantiated from.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'infra-ami-eks-worker-node-7d4ec78312', 'CentOS-8-x86_64-1905'
	AttributeHostImageName = "host.image.name"
	// The version string of the VM image or host OS as defined in Version Attributes.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '0.1'
	AttributeHostImageVersion = "host.image.version"
	// Name of the host. On Unix systems, it may contain what the hostname command
	// returns, or the fully qualified hostname, or another name specified by the
	// user.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-test'
	AttributeHostName = "host.name"
	// Type of host. For Cloud, this must be the machine type.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'n1-standard-1'
	AttributeHostType = "host.type"
)

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

// A Kubernetes Cluster.
const (
	// The name of the cluster.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-cluster'
	AttributeK8SClusterName = "k8s.cluster.name"
	// A pseudo-ID for the cluster, set to the UID of the kube-system namespace.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '218fc5a9-a5f1-4b54-aa05-46717d0ab26d'
	// Note: K8S does not have support for obtaining a cluster ID. If this is ever
	// added, we will recommend collecting the k8s.cluster.uid through the
	// official APIs. In the meantime, we are able to use the uid of the
	// kube-system namespace as a proxy for cluster ID. Read on for the
	// rationale.Every object created in a K8S cluster is assigned a distinct UID. The
	// kube-system namespace is used by Kubernetes itself and will exist
	// for the lifetime of the cluster. Using the uid of the kube-system
	// namespace is a reasonable proxy for the K8S ClusterID as it will only
	// change if the cluster is rebuilt. Furthermore, Kubernetes UIDs are
	// UUIDs as standardized by
	// ISO/IEC 9834-8 and ITU-T X.667.
	// Which states:<blockquote>
	// If generated according to one of the mechanisms defined in Rec.</blockquote>
	// ITU-T X.667 | ISO/IEC 9834-8, a UUID is either guaranteed to be
	//   different from all other UUIDs generated before 3603 A.D., or is
	//   extremely likely to be different (depending on the mechanism
	// chosen).Therefore, UIDs between clusters should be extremely unlikely to
	// conflict.
	AttributeK8SClusterUID = "k8s.cluster.uid"
)

// A Kubernetes Node object.
const (
	// The name of the Node.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'node-1'
	AttributeK8SNodeName = "k8s.node.name"
	// The UID of the Node.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1eb3a0c6-0477-4080-a9cb-0cb7db65c6a2'
	AttributeK8SNodeUID = "k8s.node.uid"
)

// A Kubernetes Namespace.
const (
	// The name of the namespace that the pod is running in.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'default'
	AttributeK8SNamespaceName = "k8s.namespace.name"
)

// A Kubernetes Pod object.
const (
	// The name of the Pod.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry-pod-autoconf'
	AttributeK8SPodName = "k8s.pod.name"
	// The UID of the Pod.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SPodUID = "k8s.pod.uid"
)

// A container in a
// [PodTemplate](https://kubernetes.io/docs/concepts/workloads/pods/#pod-templates).
const (
	// The name of the Container from Pod specification, must be unique within a Pod.
	// Container runtime usually uses different globally unique name (container.name).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'redis'
	AttributeK8SContainerName = "k8s.container.name"
	// Number of times the container was restarted. This attribute can be used to
	// identify a particular container (running or stopped) within a container spec.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 0, 2
	AttributeK8SContainerRestartCount = "k8s.container.restart_count"
)

// A Kubernetes ReplicaSet object.
const (
	// The name of the ReplicaSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SReplicaSetName = "k8s.replicaset.name"
	// The UID of the ReplicaSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SReplicaSetUID = "k8s.replicaset.uid"
)

// A Kubernetes Deployment object.
const (
	// The name of the Deployment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SDeploymentName = "k8s.deployment.name"
	// The UID of the Deployment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SDeploymentUID = "k8s.deployment.uid"
)

// A Kubernetes StatefulSet object.
const (
	// The name of the StatefulSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SStatefulSetName = "k8s.statefulset.name"
	// The UID of the StatefulSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SStatefulSetUID = "k8s.statefulset.uid"
)

// A Kubernetes DaemonSet object.
const (
	// The name of the DaemonSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SDaemonSetName = "k8s.daemonset.name"
	// The UID of the DaemonSet.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SDaemonSetUID = "k8s.daemonset.uid"
)

// A Kubernetes Job object.
const (
	// The name of the Job.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SJobName = "k8s.job.name"
	// The UID of the Job.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SJobUID = "k8s.job.uid"
)

// A Kubernetes CronJob object.
const (
	// The name of the CronJob.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'opentelemetry'
	AttributeK8SCronJobName = "k8s.cronjob.name"
	// The UID of the CronJob.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '275ecb36-5aa8-4c2a-9c47-d8bb681b9aff'
	AttributeK8SCronJobUID = "k8s.cronjob.uid"
)

// The operating system (OS) on which the process represented by this resource
// is running.
const (
	// Human readable (not intended to be parsed) OS version information, like e.g.
	// reported by ver or lsb_release -a commands.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Microsoft Windows [Version 10.0.18363.778]', 'Ubuntu 18.04.1 LTS'
	AttributeOSDescription = "os.description"
	// Human readable operating system name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'iOS', 'Android', 'Ubuntu'
	AttributeOSName = "os.name"
	// The operating system type.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	AttributeOSType = "os.type"
	// The version string of the operating system as defined in Version Attributes.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '14.2.1', '18.04.1'
	AttributeOSVersion = "os.version"
)

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

// An operating system process.
const (
	// The command used to launch the process (i.e. the command name). On Linux based
	// systems, can be set to the zeroth string in proc/[pid]/cmdline. On Windows, can
	// be set to the first parameter extracted from GetCommandLineW.
	//
	// Type: string
	// Requirement Level: Conditionally Required - See alternative attributes below.
	// Stability: experimental
	// Examples: 'cmd/otelcol'
	AttributeProcessCommand = "process.command"
	// All the command arguments (including the command/executable itself) as received
	// by the process. On Linux-based systems (and some other Unixoid systems
	// supporting procfs), can be set according to the list of null-delimited strings
	// extracted from proc/[pid]/cmdline. For libc-based executables, this would be
	// the full argv vector passed to main.
	//
	// Type: string[]
	// Requirement Level: Conditionally Required - See alternative attributes below.
	// Stability: experimental
	// Examples: 'cmd/otecol', '--config=config.yaml'
	AttributeProcessCommandArgs = "process.command_args"
	// The full command used to launch the process as a single string representing the
	// full command. On Windows, can be set to the result of GetCommandLineW. Do not
	// set this if you have to assemble it just for monitoring; use
	// process.command_args instead.
	//
	// Type: string
	// Requirement Level: Conditionally Required - See alternative attributes below.
	// Stability: experimental
	// Examples: 'C:\\cmd\\otecol --config="my directory\\config.yaml"'
	AttributeProcessCommandLine = "process.command_line"
	// The name of the process executable. On Linux based systems, can be set to the
	// Name in proc/[pid]/status. On Windows, can be set to the base name of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Requirement Level: Conditionally Required - See alternative attributes below.
	// Stability: experimental
	// Examples: 'otelcol'
	AttributeProcessExecutableName = "process.executable.name"
	// The full path to the process executable. On Linux based systems, can be set to
	// the target of proc/[pid]/exe. On Windows, can be set to the result of
	// GetProcessImageFileNameW.
	//
	// Type: string
	// Requirement Level: Conditionally Required - See alternative attributes below.
	// Stability: experimental
	// Examples: '/usr/bin/cmd/otelcol'
	AttributeProcessExecutablePath = "process.executable.path"
	// The username of the user that owns the process.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'root'
	AttributeProcessOwner = "process.owner"
	// Parent Process identifier (PID).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 111
	AttributeProcessParentPID = "process.parent_pid"
	// Process identifier (PID).
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 1234
	AttributeProcessPID = "process.pid"
)

// The single (language) runtime instance which is monitored.
const (
	// An additional description about the runtime of the process, for example a
	// specific vendor customization of the runtime environment.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Eclipse OpenJ9 Eclipse OpenJ9 VM openj9-0.21.0'
	AttributeProcessRuntimeDescription = "process.runtime.description"
	// The name of the runtime of this process. For compiled native binaries, this
	// SHOULD be the name of the compiler.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'OpenJDK Runtime Environment'
	AttributeProcessRuntimeName = "process.runtime.name"
	// The version of the runtime of this process, as returned by the runtime without
	// modification.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '14.0.2'
	AttributeProcessRuntimeVersion = "process.runtime.version"
)

// A service instance.
const (
	// Logical name of the service.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'shoppingcart'
	// Note: MUST be the same for all instances of horizontally scaled services. If
	// the value was not specified, SDKs MUST fallback to unknown_service:
	// concatenated with process.executable.name, e.g. unknown_service:bash. If
	// process.executable.name is not available, the value MUST be set to
	// unknown_service.
	AttributeServiceName = "service.name"
	// The version string of the service API or implementation. The format is not
	// defined by these conventions.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '2.0.0', 'a01dbef8a'
	AttributeServiceVersion = "service.version"
)

// A service instance.
const (
	// The string ID of the service instance.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'my-k8s-pod-deployment-1', '627cc493-f310-47de-96bd-71410b7dec09'
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
	AttributeServiceInstanceID = "service.instance.id"
	// A namespace for service.name.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'Shop'
	// Note: A string value having a meaning that helps to distinguish a group of
	// services, for example the team name that owns a group of services. service.name
	// is expected to be unique within the same namespace. If service.namespace is not
	// specified in the Resource then service.name is expected to be unique for all
	// services that have no explicit namespace defined (so the empty/unspecified
	// namespace is simply one more valid namespace). Zero-length namespace string is
	// assumed equal to unspecified namespace.
	AttributeServiceNamespace = "service.namespace"
)

// The telemetry SDK used to capture data recorded by the instrumentation
// libraries.
const (
	// The language of the telemetry SDK.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	AttributeTelemetrySDKLanguage = "telemetry.sdk.language"
	// The name of the telemetry SDK as defined above.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'opentelemetry'
	// Note: The OpenTelemetry SDK MUST set the telemetry.sdk.name attribute to
	// opentelemetry.
	// If another SDK, like a fork or a vendor-provided implementation, is used, this
	// SDK MUST set the
	// telemetry.sdk.name attribute to the fully-qualified class or module name of
	// this SDK's main entry point
	// or another suitable identifier depending on the language.
	// The identifier opentelemetry is reserved and MUST NOT be used in this case.
	// All custom identifiers SHOULD be stable across different versions of an
	// implementation.
	AttributeTelemetrySDKName = "telemetry.sdk.name"
	// The version string of the telemetry SDK.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: '1.2.3'
	AttributeTelemetrySDKVersion = "telemetry.sdk.version"
)

const (
	// cpp
	AttributeTelemetrySDKLanguageCPP = "cpp"
	// dotnet
	AttributeTelemetrySDKLanguageDotnet = "dotnet"
	// erlang
	AttributeTelemetrySDKLanguageErlang = "erlang"
	// go
	AttributeTelemetrySDKLanguageGo = "go"
	// java
	AttributeTelemetrySDKLanguageJava = "java"
	// nodejs
	AttributeTelemetrySDKLanguageNodejs = "nodejs"
	// php
	AttributeTelemetrySDKLanguagePHP = "php"
	// python
	AttributeTelemetrySDKLanguagePython = "python"
	// ruby
	AttributeTelemetrySDKLanguageRuby = "ruby"
	// rust
	AttributeTelemetrySDKLanguageRust = "rust"
	// swift
	AttributeTelemetrySDKLanguageSwift = "swift"
	// webjs
	AttributeTelemetrySDKLanguageWebjs = "webjs"
)

// The telemetry SDK used to capture data recorded by the instrumentation
// libraries.
const (
	// The version string of the auto instrumentation agent, if used.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1.2.3'
	AttributeTelemetryAutoVersion = "telemetry.auto.version"
)

// Resource describing the packaged software running the application code. Web
// engines are typically executed using process.runtime.
const (
	// Additional description of the web engine (e.g. detailed version and edition
	// information).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: 'WildFly Full 21.0.0.Final (WildFly Core 13.0.1.Final) - 2.2.2.Final'
	AttributeWebEngineDescription = "webengine.description"
	// The name of the web engine.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: experimental
	// Examples: 'WildFly'
	AttributeWebEngineName = "webengine.name"
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
	// Stability: experimental
	// Examples: 'io.opentelemetry.contrib.mongodb'
	AttributeOTelScopeName = "otel.scope.name"
	// The version of the instrumentation scope - (InstrumentationScope.Version in
	// OTLP).
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: experimental
	// Examples: '1.0.0'
	AttributeOTelScopeVersion = "otel.scope.version"
)

// Span attributes used by non-OTLP exporters to represent OpenTelemetry
// Scope's concepts.
const (
	// Deprecated, use the otel.scope.name attribute.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: 'io.opentelemetry.contrib.mongodb'
	AttributeOTelLibraryName = "otel.library.name"
	// Deprecated, use the otel.scope.version attribute.
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: deprecated
	// Examples: '1.0.0'
	AttributeOTelLibraryVersion = "otel.library.version"
)

func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeBrowserBrands,
		AttributeBrowserLanguage,
		AttributeBrowserMobile,
		AttributeBrowserPlatform,
		AttributeCloudAccountID,
		AttributeCloudAvailabilityZone,
		AttributeCloudPlatform,
		AttributeCloudProvider,
		AttributeCloudRegion,
		AttributeCloudResourceID,
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
		AttributeGCPCloudRunJobExecution,
		AttributeGCPCloudRunJobTaskIndex,
		AttributeGCPGceInstanceHostname,
		AttributeGCPGceInstanceName,
		AttributeHerokuAppID,
		AttributeHerokuReleaseCommit,
		AttributeHerokuReleaseCreationTimestamp,
		AttributeContainerCommand,
		AttributeContainerCommandArgs,
		AttributeContainerCommandLine,
		AttributeContainerID,
		AttributeContainerImageID,
		AttributeContainerImageName,
		AttributeContainerImageTag,
		AttributeContainerName,
		AttributeContainerRuntime,
		AttributeDeploymentEnvironment,
		AttributeDeviceID,
		AttributeDeviceManufacturer,
		AttributeDeviceModelIdentifier,
		AttributeDeviceModelName,
		AttributeFaaSInstance,
		AttributeFaaSMaxMemory,
		AttributeFaaSName,
		AttributeFaaSVersion,
		AttributeHostArch,
		AttributeHostID,
		AttributeHostImageID,
		AttributeHostImageName,
		AttributeHostImageVersion,
		AttributeHostName,
		AttributeHostType,
		AttributeK8SClusterName,
		AttributeK8SClusterUID,
		AttributeK8SNodeName,
		AttributeK8SNodeUID,
		AttributeK8SNamespaceName,
		AttributeK8SPodName,
		AttributeK8SPodUID,
		AttributeK8SContainerName,
		AttributeK8SContainerRestartCount,
		AttributeK8SReplicaSetName,
		AttributeK8SReplicaSetUID,
		AttributeK8SDeploymentName,
		AttributeK8SDeploymentUID,
		AttributeK8SStatefulSetName,
		AttributeK8SStatefulSetUID,
		AttributeK8SDaemonSetName,
		AttributeK8SDaemonSetUID,
		AttributeK8SJobName,
		AttributeK8SJobUID,
		AttributeK8SCronJobName,
		AttributeK8SCronJobUID,
		AttributeOSDescription,
		AttributeOSName,
		AttributeOSType,
		AttributeOSVersion,
		AttributeProcessCommand,
		AttributeProcessCommandArgs,
		AttributeProcessCommandLine,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessOwner,
		AttributeProcessParentPID,
		AttributeProcessPID,
		AttributeProcessRuntimeDescription,
		AttributeProcessRuntimeName,
		AttributeProcessRuntimeVersion,
		AttributeServiceName,
		AttributeServiceVersion,
		AttributeServiceInstanceID,
		AttributeServiceNamespace,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKVersion,
		AttributeTelemetryAutoVersion,
		AttributeWebEngineDescription,
		AttributeWebEngineName,
		AttributeWebEngineVersion,
		AttributeOTelScopeName,
		AttributeOTelScopeVersion,
		AttributeOTelLibraryName,
		AttributeOTelLibraryVersion,
	}
}
