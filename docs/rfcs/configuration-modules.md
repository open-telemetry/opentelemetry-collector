# Shareable modular configuration

Add support for shareable modular configuration to collect signals from specific
services or applications.

## Motivation

Distributing high level configurations focused on monitoring specific services
or applications is a feature commonly found in observability solutions. These
features help day to day users to reduce learning curve and maintenance burden,
and allow knowledgeable users to share opinionated configuration that can be
quickly adopted.

Three options are discussed on this RFC, they all come from previous existing
discussions. None of these options gained enough traction to reach a final
state. The expected outcome of this RFC is to decide on the approach to follow
for the implementation.

## Non-goals

It is not a goal of this functionality to offer general templating for the OTel
collector configuration. Scenarios where general templating is needed are
probably better covered by configuration management tools. General templating
based on other tools could still be combined with configurations using modules.

## Terminology

A note on terminology. We will use the term "modules" for this reusable
configuration on this RFC. This is a term used in the industry for this kind of
feature. Other name commonly used is "integrations", but this term is [already
used](https://opentelemetry.io/ecosystem/integrations/) in the OTel ecosystem.
Previous discussions use the term "templates". We are not using this term here
to avoid assumptions related with general templating of configuration.

## Explanation

From the user perspective, this feature should allow to use high-level modules
by their name and parameterize them with a set of variables. These modules will
contain receivers and processors configured for signal collection from an
specific service. Each module can contain any number of receivers, and any
number of pipelines including at least one processor. Pipelines will only define
receivers and processors. Modules won't include other components such as extensions
or exporters.

It should be possible to configure the source of the modules. For example they
could be included in the configuration itself, in external files, in some hosted
service or in K8s config maps.

This feature should play well with autodiscovery features such as the receiver
creator, so it is possible to apply modules for autodiscovered loads. It can be
very handy when [defining configurations in annotations](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/34427).

See [open-telemetry/opentelemetry-collector-contrib#36116](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/36116)
for more details from the user point of view.

### Summary of user interface

Configurations will be defined in modules as templates, these templates will
look pretty similar to usual OTel collector configurations, but they will only
allow the definition of receivers and processors, and pipelines containing them.
Specific templating language is TBD.

Something like this:
```
receivers:
  prometheus/someservice:
    config:
      scrape_configs:
        - job_name: 'someservice'
          static_configs:
            - targets: [${var:endpoint}]
          basic_auth:
            username: ${var:username}
            password: ${var:password}
          metric_relabel_configs:
            ...
processors:
  filter/something:
    metrics:
      exclude:
        match_type: strict
        metric_names: ...
pipelines:
  metrics/somepipeline:
    receiver: prometheus/someservice
    processors: [filter/something]
```

In a configuration file, a module could be used with something like this:
```
receivers:
  module/somemodule:
    name: somemodule
    parameters:
      endpoint: https://localhost:1234
      username: someuser
      password: somepassword
...
service:
  pipelines:
    metrics:
      receivers: [module/somemodule] 
      exporters: [...]
```

Processors pipelines could be used also independently, with something like this:
```
processors:
  module/somemodule:
    name: somemodule
    pipeline: metrics/somepipeline
...
service:
  pipeline:
    metrics:
      receivers: [...]
      processors: [module/somemodule]
      exporters: [...]
```

A module can contain multiple pipelines of multiple signal types. The
implementation should be aware of this and select the pipelines to create
depending on the type of the pipeline.

## Technical options

Some options are described here, and a general summary of pros and cons can be
found below, to help making decisions.

### Option 1 (Recommended): Module components

New receiver and processor components are implemented. They can instantiate the
pipelines defined in the modules internally, by calling the subcomponent factories
and chaining them with the provided consumers.

This approach was originally proposed in [open-telemetry/opentelemetry-collector-contrib#26312](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/26312).

#### Internal details

The module components create their subcomponents on `Start()`, by getting the
factories from the `component.Host`. They keep record of all their subcomponents
so they can be stopped on `Shutdown()`.

Modules can be used this way in any place where the components can be used, what
in principle provides a more natural user experience. With this they also have
synergies with any feature that accepts them. For example the receiver
creator could use the module receiver directly, supporting autodiscovery use
cases.

Module sources are provided via extensions that components can use to discover
modules by their name. For example the following extension could be used to
provide templates from a local directory:
```
extensions:
  file_modules:
    path: "./modules"
```

As all pieces are implemented as independent components, each of them can be
optionally used in distributions.

POC for this approach is available in [elastic/opentelemetry-collector-components#96](https://github.com/elastic/opentelemetry-collector-components/pull/96)

#### Trade-offs and mitigations

* Are factory getters always going to be available in the `component.Host`? They
  are not in the current interface and could be a blocker if there are plans to
  remove them.
* Subcomponents are built on `Start()`, while components are usually created when
  unmarshalling the configuration.
* Subcomponents are not available on the internal graph, so it is going to be
  difficult to access the effective configuration.
* Module receiver also instantiates processors, this cannot be represented with
  usual configuration without using connectors.

Mitigating these trade-offs can be complex. They would imply making the
factories available to the factories themselves, and/or providing some internal
API for instantiating subcomponents while updating the internal graph.

Most trade-offs of this approach also exist on the receiver creator, that
uses a similar approach to create receivers. Mitigating them for one would
mitigate them for both, and could also help in other features such as
configuration reload.

### Option 2: Module converter

Modules can be used in the configuration as any other component, but they don't
correspond to any actual component. A new converter is introduced to expand these
modules while loading configuration.

#### Internal details

A new converter is introduced for templates expansion. It is executed as any other
converter when [resolving configuration](https://github.com/open-telemetry/opentelemetry-collector/blob/main/confmap/README.md#configuration-resolving). After it is resolved, it is unmarshalled as any other configuration.

The expansion process removes the modules and replaces them by receivers and/or
processors. New pipelines are added using the expanded receivers and processors,
using a forward connector as exporter. This connector is then used as receiver
in the pipelines defined in the configuration by the user.

Configuration of this feature can be done as a new top-level entry. This entry
needs to be unmarshaled and removed from the config by the converter.
```
modules:
  path: "./modules"
```
Different source implementations would need to be part of the converter itself.

There is an implementation of this approach in [open-telemetry/opentelemetry-collector#8507](https://github.com/open-telemetry/opentelemetry-collector/pull/8507).

#### Trade-offs and mitigations

* The forward connector is an additional dependency that distributions must include
  for modules to work. This can be mitigated by documentation and/or runtime checks.
* To modify components and pipelines in the configuration, this converter needs
  to be aware of the structure of the configuration. This is out of the scope
  for a converter, that are more intended for small replacements not dependant
  on the configuration format. This is mitigated by option 3, that introduces a
  higher level approach for configuration processing.
* The converter needs to take care of unmarshalling its own configuration. This
  would be also mitigated by option 3.
* Using modules with the autodiscovery features provided by the receiver creator
  needs explicit support in the converter or in the receiver creator.
* Configuring modules [from k8s annotations](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/34427#issuecomment-2413401158) may not be possible.
  As this configuration would not be known when unmarshalling the collector
  configuration. This can be difficult to mitigate with this option.

### Option 3: Config processor / Recursive unmarshalling

This is a variation of option 2. Modules can be used in the configuration as any
other component. A new extension point is added in the collector that allows
higher-level modification of the configuration as part of the unmarshalling
process. A new config processor is added to expand modules.

#### Internal details

A new extension point is added to the OTel collector, for config processors.
These processors are executed on partially parsed configuration, taking the
opportunity to modify any part of the configuration. After all the config
processors have been executed, a valid `otelcol.Config` must result.

This approach is described in [open-telemetry/opentelemetry-collector#8940](https://github.com/open-telemetry/opentelemetry-collector/issues/8940),
and could be leveraged also in other requested features, such as the
one for [component groups](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/18509).

Once this extension point is added, a new config provider would be implemented
that would take care of expanding module subcomponents. Expansion would work in
a similar fashion to option 2, creating new pipelines and plugging them to the
pipelines in the configuration file using the forward connector. In contrast to
option 2, configuration unmarshalling would be provided by the collector config
provider.

Configuration of module sources would be done the same way as in option 2.

#### Trade-offs and mitigations

* Potentially risky and complex implementation, as it is a significant change in
  the unmarshalling process. The extension point will require its own
  design and implementation process and in the meantime it can block progress on modules.
  This could be mitigated by temporarily using the converter approach (option 2), and replace
  it when the extension point is available. 
* The forward connector is an additional dependency that distributions must include
  for modules to work. This can be mitigated by documentation and runtime checks.
* Using modules with the autodiscovery features provided by the receiver creator
  needs explicit support in the converter or in the receiver creator.
* Configuring modules [from k8s annotations](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/34427#issuecomment-2413401158) may not be possible.
  As this configuration would not be known when unmarshalling the collector
  configuration. This can be difficult to mitigate with this option.

### Summary of options

Some of the decision points which trade-offs orbit around are the observability
options for the effective configuration, its integration with the receiver
creator for autodiscovery use cases or dynamic configuration in general, and
its user experience.

We recommend Option 1 because it provides a more natural user experience, based
on optional components, and it works better when combined with the receiver
creator, what can be important in autodiscovery use cases. Using Options 2 and 3
with the receiver creator will require additional developments, and may be
challenging to support configurations [based on annotations](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/34427#issuecomment-2413401158), as these annotations
cannot be known when unmarshalling the collector configuration.

The main reasons to avoid Option 1 are that it instantiates components on `Start()`,
what could be breaking architectural boundaries, and that it would be difficult to
give visibility on the "effective configuration". Though these trade-offs are also
present to some point in the receiver creator, that is already used in the OTel
collector ecosystem, and we consider them less relevant than the usability
trade-offs of Options 2 and 3,

## Prior art and alternatives

An alternative, proposed in [open-telemetry/opentelemetry-collector#8372](https://github.com/open-telemetry/opentelemetry-collector/issues/8372), could be to add a confmap provider that expands configuration files as templates.
This has the problem of coupling template functionality with configuration
sourcing, and it also has a less intuitive user experience than the other
options. Given these limitations, it is not discussed here.

Also, without adding anything, the current config resolver is already able to
combine multiple configuration files. This approach is discussed in
[some comments](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/26312#issuecomment-1702391029)
in one of the previous issues discussing templates. Even when functional, this
approach depends on the order of configuration flags, and doesn't provide a
defined abstraction for sharing configuration, so we consider it would be better
to define this abstraction.

Other alternatives could be based on external tools already available in the CNCF
ecosystem, such as Kustomize or Helm. They could cover some use cases, but at
least for the mentioned tools, they would be limited to Kubernetes. Given the
relevance of these tools, their relationship with this feature is detailed below.

### Kustomize

Kustomize is a tool for declarative management of Kubernetes configuration. In
that regard, it could be used to organize the OTel Collector configuration when
deployed on this environment, and provide similar benefits to the proposal in
this document. But it would not be useful in other environments.

It would be also difficult to use. Shareable modular configuration based on
Kustomize would be based on composition of resources, where user-specific
settings would be provided as [JSON patches](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/patchesjson6902/)
or as [replacements](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/replacements/).
In both cases it is needed to know the structure of the final configuration,
what may not be possible for shared configuration, specially if it can be used
in different places, or in combination with the receiver creator.

Kustomize could be used though in combination with modules. For example
kustomize could be used to deploy templates in a configmap, like in the
following example:
```yaml
nameSuffix: ""
resources:
- service-account.yml
- cluster-role.yml
- role-binding.yml
- otel-collector.yml
configMapGenerator:
- name: templates
  files:
  - templates/nginx.yml
```

And then the template can be used in the collector configuration, like in this
example where the module is used with the receiver creator:
```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
...
spec:
  volumes:
    - name: templates
      configMap:
        name: templates
    - name: data
      emptyDir:
  volumeMounts:
    - name: templates
      mountPath: /templates
      readOnly: true
    - name: data
      mountPath: /data
  config:
    extensions:
      k8s_observer:
        auth_type: serviceAccount
        observe_pods: true
      file_modules:
        path: "/templates"
      file_storage:
        directory: "/data"
    receivers:
      receiver_creator:
        watch_observers: [k8s_observer]
        receivers:
          module/nginx:
            rule: type == "port" && pod.name matches "nginx"
            config:
              name: "nginx"
              parameters:
                endpoint: 'http://`host`:`port`/nginx-status'
                access_paths:
                - '/var/log/pods/`pod.namespace`_`pod.name`_`pod.uid`/`container_name`/*.log'
                storage_resource: file_storage
...
```

### Helm

TODO

## Open questions

### What templating language to use?

On the described options we are not detailing the templating language to use.
They could work with different languages. We have to make a concious decision on
what templating language to use. Some options that have appeared in the
different discussions and POCs are:
* Go templates are used in different POCs, and are a natural option being a Go
  project. It introduces though a new configuration language in the ecosystem,
  and the template is not valid YAML itself.
* Use a confmap resolver with "sandboxed" providers. It has the advantage of
  avoiding the inclusion of other templating languages, but it is not an actual
  templating language. It can be used to replace variables, but it doesn't have
  conditional logic or loops. We would need to confirm if this language is
  enough.
* Receiver creator uses expvar for variable expansion. This could be another
  option, already used in the ecosystem, but also limited for conditional logic and
  loops.
* Supporting multiple templating languages.

### Definition of versioning, dependencies and other metadata?

Ideally, it should be possible to include some metadata in the templates. At
least we have to decide if templates should be versioned and how, and if their
component dependencies should be declared.

### Development environment?

How could the development process of modules be? Where does their code
reside? How are they tested?

## Future possibilities

Having support for modules would help to have in the future a marketplace-like
site where users can easily obtain well-tested configurations prepared by
expert users.
