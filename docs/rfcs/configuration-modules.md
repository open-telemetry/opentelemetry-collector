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

## Terminology

A note on terminology. We will use the term "modules" for this reusable
configuration on this RFC. This is a term used in the industry for this kind of
feature. Other name commonly used is "integrations", but this term is already
used in the OTel ecosystem. Previously considered alternatives use the term
"templates". We are not using this term here to avoid coupling this feature with
general templating.

## Explanation

From the user perspective, this feature should allow to use high-level modules
by their name and parameterize them with a set of variables. These modules will
contain receivers and processors configured for signal collection from an
specific service. Each module can contain any number of receivers, and any
number of pipelines including at least one processor. Pipelines will only define
receivers and processors, they won't define other components such as extensions
or exporters.

It should be possible to configure the source of the modules. For example they
could be included in the configuration itself, in external files, in some hosted
service or in config maps.

This feature should play well with autodiscovery features such as the receiver
creator.

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

A module can contain multiple pipelines, of multiple signal types. The
implementation should be aware of this and select the pipelines to create
depending on the type of pipeline.

## Technical options

### Option 1: Module components

New receiver and processor components are implemented. They can instantiate the
pipelines defined in the modules internally, by calling the subcomponent factories
and chaining them with the provided consumers.

This approach was originally proposed in [open-telemetry/opentelemetry-collector-contrib#26312](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/26312).

#### Internal details

The module components create their subcomponents on `Start()`, by getting the
factories from the `component.Host`. They keep record of all subcomponents so
they can be stopped on `Shutdown()`.

Modules can be used this way in any place where the components can be used, what
in principle provides a more natural user experience. With this they also have
synergies with any feature that accepts them. For example the receiver
creator could use the module receiver directly, supporting autodiscovery use
cases.

Module sources are provided via extensions, that components can use to discover
modules by their name.

As all pieces are implemented as independent components, each of them can be
optionally used in distributions.

POC for this approach is available in [elastic/opentelemetry-collector-components#96](https://github.com/elastic/opentelemetry-collector-components/pull/96)

#### Trade-offs and mitigations

Most trade-offs of this approach also exist on the receiver creator, that
uses a similar approach to create receivers. Mitigating them for one would
mitigate them for both.

* Are factory getters always going to be available in the `component.Host`? They
  are not in the current interface.
* Subcomponents are built on `Start()`, while components are usually created when
  unmarshalling the configuration.
* Subcomponents are not available on the internal graph, so it is going to be
  difficult to access the effective configuration.
* Module receiver also instantiates processors, this cannot be represented with
  usual configuration without using connectors.

Mitigating these trade-offs can be complex. They would imply making the
factories available to the factories themselves, and/or providing some internal
API for instantiating subcomponents while updating the internal graph.

### Option 2: Module converter

Modules can be used in the configuration as any other component. A new converter
is introduced to expand these modules while loading configuration.

#### Internal details

#### Trade-offs and mitigations

### Option 3: Config processor / Recursive unmarshalling

This is a follow-up of option 2. Modules can be used in the configuration as any
other component. A new extension point is added in the collector that allows
higher-level modification of the configuration as part of the unmarshalling
process. A new config processor is added to expand modules.

#### Internal details

#### Trade-offs and mitigations

### Other options not considered on this RFC:

Template provider https://github.com/open-telemetry/opentelemetry-collector/issues/8372

Combining configuration files https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/26312#issuecomment-1702391029

#######

## Internal details

From a technical perspective, how do you propose accomplishing the proposal? In particular, please explain:

* How the change would impact and interact with existing functionality
* Likely error modes (and how to handle them)
* Corner cases (and how to handle them)

While you do not need to prescribe a particular implementation - indeed, OTEPs should be about **behaviour**, not implementation! - it may be useful to provide at least one suggestion as to how the proposal *could* be implemented. This helps reassure reviewers that implementation is at least possible, and often helps them inspire them to think more deeply about trade-offs, alternatives, etc.

## Trade-offs and mitigations

What are some (known!) drawbacks? What are some ways that they might be mitigated?

Note that mitigations do not need to be complete *solutions*, and that they do not need to be accomplished directly through your proposal. A suggested mitigation may even warrant its own OTEP!

#######

## Prior art and alternatives

What are some prior and/or alternative approaches? For instance, is there a corresponding feature in OpenTracing or OpenCensus? What are some ideas that you have rejected?

## Open questions

What are some questions that you know aren't resolved yet by the OTEP? These may be questions that could be answered through further discussion, implementation experiments, or anything else that the future may bring.

### What option should be implemented?

### What templating language to use?

### Definition of versioning, dependencies and other metadata?

## Future possibilities

What are some future changes that this proposal would enable?
