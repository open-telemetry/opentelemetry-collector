# Coding guidelines

We consider the OpenTelemetry Collector to be close to production quality and the quality bar
for contributions is set accordingly. Contributions must have readable code written
with maintainability in mind (if in doubt check [Effective Go](https://golang.org/doc/effective_go.html)
for coding advice). The code must adhere to the following robustness principles that
are important for software that runs autonomously and continuously without direct
interaction with a human (such as this Collector).

## Naming convention

To keep naming patterns consistent across the project, naming patterns are enforced to make intent clear by:

- Methods that return a variable that uses the zero value or values provided via the method MUST have the prefix `New`. For example:
  - `func NewKinesisExporter(kpl aws.KinesisProducerLibrary)` allocates a variable that uses
    the variables passed on creation.
  - `func NewKeyValueBuilder()` SHOULD allocate internal variables to a safe zero value.
- Methods that return a variable that uses non-zero value(s) that impacts business logic MUST use the prefix `NewDefault`. For example:
  - `func NewDefaultKinesisConfig()` would return a configuration that is the suggested default
    and can be updated without concern of causing a race condition.
- Methods that act upon an input variable MUST have a signature that reflects concisely the logic being done. For example:
  - `func FilterAttributes(attrs []Attribute, match func(attr Attribute) bool) []Attribute` MUST only filter attributes out of the passed input
    slice and return a new slice with values that `match` returns true. It may not do more work than what the method name implies, ie, it
    must not key a global history of all the slices that have been filtered.
- Methods that get the value of a field i.e. a getterMethod MUST use an uppercase first letter and NOT a `get` prefix. For example:
  - `func (p *Person) Name() string {return p.name} ` Name (with an uppercase N, exported) method is used here to get the value of the name field and not `getName`.The use of upper-case names for export provides the hook to discriminate the field from the method.
- Methods that set the value of a field i.e. a setterMethod MUST use a `set` prefix. For example:
  - `func (p *Person) SetName(newName string) {p.name = newName}` SetName method here sets the value of the name field.
- Variable assigned in a package's global scope that is preconfigured with a default set of values MUST use `Default` as the prefix. For example:
  - `var DefaultMarshallers = map[string]pdata.Marshallers{...}` is defined with an exporter package that allows for converting an encoding name,
    `zipkin`, and return the preconfigured marshaller to be used in the export process.
- Types that are specific to a signal MUST be worded with the signal used as an adjective, i.e. `SignalType`. For example:
  - `type TracesSink interface {...}`
- Types that deal with multiple signal types should use the relationship between the signals to describe the type, e.g. `SignalToSignalType` or `SignalAndSignalType`. For example:
  - `type TracesToTracesFunc func(...) ...`
- Functions dealing with specific signals or signal-specific types MUST be worded with the signal or type as a direct object, i.e. `VerbSignal`, or `VerbType` where `Type` is the full name of the type including the signal name. For example:
  - `func ConsumeTraces(...) {...}`
  - `func CreateTracesExport(...) {...}`
  - `func CreateTracesToTracesFunc(...) {...}`

### Configuration structs

When naming configuration structs, use the following guidelines:

- Separate the configuration set by end users in their YAML configuration from the configuration set by developers in the code into different structs.
- Use the `Config` suffix for configuration structs that have end user configuration (i.e. that set in their YAML configuration). For example, `configgrpc.ClientConfig` ends in `Config` since it contains end user configuration.
- Use the `Settings` suffix for configuration structs that are set by developers in the code. For example, `component.TelemetrySettings` ends in `Settings` since it is set by developers in the code.
- Avoid redundant prefixes that are already implied by the package name. For example, use`configgrpc.ClientConfig` instead of `configgrpc.GRPCClientConfig`.

## Module organization

As usual in Go projects, organize your code into packages grouping related functionality. To ensure
that we can evolve different parts of the API independently, you should also group related packages
into modules.

We use the following rules for some common situations where we split into separate modules:
1. Each top-level directory should be a separate module.
1. Each component referenceable by the OpenTelemetry Collector Builder should be in a separate
   module. For example, the OTLP receiver is in its own module, different from that of other
   receivers.
1. Consider splitting into separate modules if the API may evolve independently in separate groups
   of packages. For example, the configuration related to HTTP and gRPC evolve independently, so
   `config/configgrpc` and `config/confighttp` are separate modules.
1. For component names, add the component kind as a suffix for the module name. For example, the
   OTLP receiver is in the `receiver/otlpreceiver` module.
1. Modules that add specific functionality related to a parent folder should have a prefix in the
   name that relates to the parent module. For example, `configauth` has the `config` prefix since
   it is part of the `config` folder, and `extensionauth` has `extension` as a prefix since it is
   part of the `extension` module.
1. Testing helpers should be in a separate submodule with the suffix `test`. For example, if you
   have a module `component`, the helpers should be in `component/componenttest`. Testing helpers
   that are used across multiple modules should be in the [`internal/testutil`](https://github.com/open-telemetry/opentelemetry-collector/tree/main/internal/testutil)
   module.
1. Experimental packages that will later be added to another module should be in their own module,
   named as they will be after integration. For example, if adding a `pprofile` package to `pdata`,
   you should add a separate module `pdata/pprofile` for the experimental code.
1. Experimental code that will be added to an existing package in a stable module can be a submodule
   with the same name, but prefixed with an `x`. For example, `config/confighttp` module can have an
   experimental module named `config/confighttp/xconfighttp` that contains experimental APIs.

When adding a new module remember to update the following:
1. Add a changelog note for the new module.
1. Add the module in `versions.yaml`.
1. Use `make crosslink` to make sure the module replaces are added correctly throughout the
   codebase. You may also have to manually add some of the replaces.
1. Update the [otelcorecol
   manifest](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/otelcorecol/builder-config.yaml)
   and [builder
   tests](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/builder/internal/builder/main_test.go).
1. Open a follow up PR to update pseudo-versions in all go.mod files. See [this example
   PR](https://github.com/open-telemetry/opentelemetry-collector/pull/11668).

## Enumerations

To keep naming patterns consistent across the project, enumeration patterns are enforced to make intent clear:

- Enumerations should be defined using a type definition, such as `type Level int32`.
- Enumerations should use either `int` or `string` as the underlying type
- The enumeration name should succinctly describe its purpose
  - If the package name represents the entity described by the enumeration then the package name should be factored into the name of the enumeration.  For example, `component.Type` instead of `component.ComponentType`.
  - The name should convey a sense of limited categorization. For example, `pcommon.ValueType` is better than `pcommon.Value` and `component.Kind` is better than `component.KindType`, since `Kind` already conveys categorization.
- Constant values of an enumeration should be prefixed with the enumeration type name in the name:
  - `pcommon.ValueTypeStr` for `pcommon.ValueType`
  - `pmetric.MetricTypeGauge` for `pmetric.MetricType`


## Recommended Libraries / Defaults

In order to simplify development within the project, we have made certain library recommendations that should be followed.

| Scenario 	 | Recommended                   	                | Rationale                                                                                                                  |
|------------|------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| Hashing  	 | ["hashing/fnv"](https://pkg.go.dev/hash/fnv) 	 | The project adopted this as the default hashing method due to the efficiency and is reasonable for non-cryptographic use 	 |
| Testing  	 | Use `t.Parallel()` where possible            	 | Enabling more tests to be run in parallel will speed up the feedback process when working on the project.                 	 |


Within the project, there are some packages that have yet to follow the recommendations and are being addressed. However, any new code should adhere to the recommendations.

## Default Configuration

To guarantee backward-compatible behavior, all configuration packages should supply a `NewDefault[config name]` functions that create a default version of the config. The package does not need to guarantee that `NewDefault[config name]` returns a usable configuration—only that default values will be set. For example, if the configuration requires that a field, such as `Endpoint` be set, but there is no valid default value, then `NewDefault[config name]` may set that value to `""` with the expectation that the user will set a valid value.

Users should always initialize the config struct with this function and overwrite anything as needed.

## Startup Error Handling

Verify configuration during startup and fail fast if the configuration is invalid.
This will bring the attention of a human to the problem as it is more typical for humans
to notice problems when the process is starting as opposed to problems that may arise
sometime (potentially long time) after process startup. Monitoring systems are likely
to automatically flag processes that exit with failure during startup, making it
easier to notice the problem. The Collector should print a reasonable log message to
explain the problem and exit with a non-zero code. It is acceptable to crash the process
during startup if there is no good way to exit cleanly but do your best to log and
exit cleanly with a process exit code.

## Propagate Errors to the Caller

Do not crash or exit outside the `main()` function, e.g. via `log.Fatal` or `os.Exit`,
even during startup. Instead, return detailed errors to be handled appropriately
by the caller. The code in packages other than `main` may be imported and used by
third-party applications, and they should have full control over error handling
and process termination.

## Do not Crash after Startup

Do not crash or exit the Collector process after the startup sequence is finished.
A running Collector typically contains data that is received but not yet exported further
(e.g. data that is stored in the queues and other processors). Crashing or exiting the Collector
process will result in losing this data since typically the receiver has
already acknowledged the receipt for this data and the senders of the data will
not send that data again.

## Bad Input Handling

Do not crash on bad input in receivers or elsewhere in the pipeline.
[Crash-only software](https://en.wikipedia.org/wiki/Crash-only_software)
is valid in certain cases; however, this is not a correct approach for Collector (except
during startup, see above). The reason is that many senders from which the Collector
receives data have built-in automatic retries of the _same_ data if no
acknowledgment is received from the Collector. If you crash on bad input
chances are high that after the Collector is restarted it will see the same
data in the input and will crash again. This will likely result in an infinite
crashing loop if you have automatic retries in place.

Typically bad input when detected in a receiver should be reported back to the
sender. If it is elsewhere in the pipeline it may be too late to send a response
to the sender (particularly in processors which are not synchronously processing
data). In either case, it is recommended to keep a metric that counts bad input data.

## Error Handling and Retries

Be rigorous in error handling. Don't ignore errors. Think carefully about each
error and decide if it is a fatal problem or a transient problem that may go away
when retried. Fatal errors should be logged or recorded in an internal metric to
provide visibility to users of the Collector. For transient errors come up with a
retrying strategy and implement it. Typically you will
want to implement retries with some sort of exponential back-off strategy. For
connection or sending retries use jitter for back-off intervals to avoid overwhelming
your destination when the network is restored or the destination is recovered.
[Exponential Backoff](https://github.com/cenkalti/backoff) is a good library that
provides all this functionality.

## Logging

Log your component startup and shutdown, including successful outcomes (but don't
overdo it, and keep the number of success messages to a minimum).
This can help to understand the context of failures if they occur elsewhere after
your code is successfully executed.

Use logging carefully for events that can happen frequently to avoid flooding
the logs. Avoid outputting logs per a received or processed data item since this can
amount to a very large number of log entries (Collector is designed to process
many thousands of spans and metrics per second). For such high-frequency events
instead of logging consider adding an internal metric and incrementing it when
the event happens.

Make log messages human readable and also include data that is needed for easier
understanding of what happened and in what context.

## Executing External Processes

The components should avoid executing arbitrary external processes with arbitrary command
line arguments based on user input, including input received from the network or input
read from the configuration file. Failure to follow this rule can result in arbitrary
remote code execution, compelled by malicious actors that can craft the input.

The following limitations are recommended:
- If an external process needs to be executed limit and hard-code the location where
  the executable file may be located, instead of allowing the input to dictate the
  full path to the executable.
- If possible limit the name of the executable file to be pulled from a hard-coded
  list defined at compile time.
- If command line arguments need to be passed to the process do not take the arguments
  from the user input directly. Instead, compose the command line arguments indirectly,
  if necessary, deriving the value from the user input. Limit as much as possible the
  size of the possible space of values for command line arguments.

## Observability

Out of the box, your users should be able to observe the state of your
component. See [observability.md](observability.md) for more details.

When using the regular helpers, you should have some metrics added around key
events automatically. For instance, exporters should have
`otelcol_exporter_sent_spans` tracked without your exporter doing anything.

Custom metrics can be defined as part of the `metadata.yaml` for your component.
The authoritative source of information for this is [the
schema](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/mdatagen/metadata-schema.yaml),
but here are a few examples for reference, adapted from the tail sampling
processor:

```yaml
telemetry:
  metrics:
    # example of a histogram
    processor.tailsampling.samplingdecision.latency:
      description: Latency (in microseconds) of a given sampling policy.
      unit: µs # from https://ucum.org/ucum
      enabled: true
      histogram:
        value_type: int
        # bucket boundaries can be overridden
        bucket_boundaries: [1, 2, 5, 10, 25, 50, 75, 100, 150, 200, 300, 400, 500, 750, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000]

    # example of a counter
    processor.tailsampling.policyevaluation.errors:
      description: Count of sampling policy evaluation errors.
      unit: "{errors}"
      enabled: true
      sum:
        value_type: int
        monotonic: true

    # example of a gauge
    processor.tailsampling.tracesonmemory:
      description: Tracks the number of traces current on memory.
      unit: "{traces}"
      enabled: true
      gauge:
        value_type: int
```

Running `go generate ./...` at the root of your component should generate the
following files:

- `documentation.md`, with the metrics and their descriptions
- `internal/metadata/generated_telemetry.go`, with code that defines the metric
  using the OTel API
- `internal/metadata/generated_telemetry_test.go`, with sanity tests for the
  generated code

On your component's code, you can use the metric by initializing the telemetry
builder and storing it on a component's field:

```go
type tailSamplingSpanProcessor struct {
    ctx context.Context

    telemetry *metadata.TelemetryBuilder
}

func newTracesProcessor(ctx context.Context, settings component.TelemetrySettings, nextConsumer consumer.Traces, cfg Config, opts ...Option) (processor.Traces, error) {
    telemetry, err := metadata.NewTelemetryBuilder(settings)
    if err != nil {
        return nil, err
    }

    tsp := &tailSamplingSpanProcessor{
        ctx:            ctx,
        telemetry:      telemetry,
  }
}
```

To record the measurement, you can then call the metric stored in the telemetry
builder:

```go
tsp.telemetry.ProcessorTailsamplingSamplingdecisionLatency.Record(ctx, ...)
```

## Resource Usage

Limit usage of CPU, RAM, and other resources that the code can use. Do not write code
that consumes resources in an uncontrolled manner. For example, if you have a queue
that can contain unprocessed messages always limit the size of the queue unless you
have other ways to guarantee that the queue will be consumed faster than items are
added to it.

Performance test the code for both normal use-cases under acceptable load and also for
abnormal use-cases when the load exceeds acceptable limits many times over. Ensure that
your code performs predictably under abnormal use. For example, if the code
needs to process received data and cannot keep up with the receiving rate it is
not acceptable to keep allocating more memory for received data until the Collector
runs out of memory. Instead have protections for these situations, e.g. when hitting
resource limits drop the data and record the fact that it was dropped in a metric
that is exposed to users.

## Graceful Shutdown

Collector does not yet support graceful shutdown but we plan to add it. All components
must be ready to shutdown gracefully via `Shutdown()` function that all component
interfaces require. If components contain any temporary data they need to process
and export it out of the Collector before shutdown is completed. The shutdown process
will have a maximum allowed duration so put a limit on how long your shutdown
operation can take.

## Unit Tests

Cover important functionality with unit tests. We require that contributions
do not decrease the overall code coverage of the codebase - this is aligned with our
goal to increase coverage over time. Keep track of execution time for your unit
tests and try to keep them as short as possible.

## Semantic Conventions compatibility

When adding new metrics, attributes or entity attributes to a Collector's component (receiver, processor etc), the
[Semantic Conventions](https://github.com/open-telemetry/semantic-conventions) project should be checked first
to see if those are already defined as Semantic Conventions.
It's also important to check for any open issues that may already propose these or similar Semantic Conventions.
If no such Semantic Conventions are defined in the Semantic Conventions project, the component’s code owners
should consider initiating that process first
(refer to Semantic Conventions'
[contribution guidelines](https://github.com/open-telemetry/semantic-conventions/blob/main/CONTRIBUTING.md) 
for specific details).
The implementation of the component can still be submitted as a draft PR to demonstrate how the proposed
Semantic Conventions would be used while working in parallel to contribute the relevant updates to
the Semantic Conventions project.
The components's code owners can review the Semantic Conventions PR in collaboration with any existing domain-specific
SemConv approvers.
At their discretion, the code owners may choose to block the component’s implementation PR until the related
Semantic Conventions changes are completed.

## Telemetry Stability Levels

### Metrics

Metrics emitted by Collector scrapers/receivers (e.g. `system.cpu.time`) follow the same stability levels
as the Collector's internal metrics (e.g. `otelcol_process_cpu_seconds`), as documented in
[Internal Telemetry Stability](https://opentelemetry.io/docs/collector/internal-telemetry/#metrics).

In particular, for beta and stable levels the following guidelines apply:

#### Beta stability level

It is highly encouraged that metrics in beta stage
are also defined as Semantic Conventions based on the [Semantic Conventions compatibility](#semantic-conventions-compatibility),
ensuring cross-project consistency.

#### Stable stability level

Before promoting a metric to stable, it should be discussed whether it needs to
be defined as a Semantic Convention, following the [Semantic Conventions compatibility](#semantic-conventions-compatibility).
Promoting a metric to stable without it being a Semantic Convention involves
the risk of potential divergence within OpenTelemetry's projects.
For example, a stable metric in the Collector might be introduced in a slightly
different way in another OpenTelemetry project in the future, or it might be proposed
as a Semantic Convention in the future.
In case of such divergence, a stable Collector metric won't be allowed to
change, and if wider alignment is needed, the metric should be deprecated and
removed in order to come into alignment with the Semantic Conventions.
Consequently, the Collector's maintainers and components' code owners should
acknowledge that risk before marking a metric as stable without it being a
stable Semantic Convention and should provide justification for the decision. In
any case, [Semantic Conventions' guidelines](https://opentelemetry.io/docs/specs/semconv/how-to-write-conventions/)
should be advised when metrics are defined within the Collector directly.

### Testing Library Recommendations

To keep testing practices consistent across the project, it is advised to use these libraries under
these circumstances:

- For assertions to validate expectations, use `"github.com/stretchr/testify/assert"`
- For assertions that are required to continue the test, use `"github.com/stretchr/testify/require"`
- For mocking external resources, use `"github.com/stretchr/testify/mock"`
- For validating HTTP traffic interactions, `"net/http/httptest"`

## Integration Testing

Integration testing is encouraged throughout the project, container images can be used in order to facilitate
a local version. In their absence, it is strongly advised to mock the integration.

## Using CGO

Using CGO is prohibited due to the lack of portability and complexity
that comes with managing external libraries with different operating systems and configurations.
However, if the package MUST use CGO, this should be explicitly called out within the readme
with clear instructions on how to install the required libraries.
Furthermore, if your package requires CGO, it MUST be able to compile and operate in a no-op mode
or report a warning back to the collector with a clear error saying CGO is required to work.

## Breaking changes

Whenever possible, we adhere to [semver](https://semver.org/) as our minimum standards. Even before v1, we strive not to break compatibility
without a good reason. Hence, when a change is known to cause a breaking change, we intend to follow these principles:

- Breaking changes MUST have migration guidelines that clearly explain how to adapt to them.
- Users SHOULD be able to adopt the breaking change at their own pace, independent of other Collector updates.
- Users SHOULD be proactively notified about the breaking change before a migration is required.
- Users SHOULD be able to easily tell whether they have completed the migration for a breaking change.

Not all changes have the same effects on users, so some of the steps may be unnecessary for some changes.

### API breaking changes

We strive to perform API breaking changes in two stages, deprecating it first (`vM.N`) and breaking it in a subsequent
version (`vM.N+1`).

- when we need to remove something, we MUST mark a feature as deprecated in one version and MAY remove it in a
  subsequent one
- when renaming or refactoring types, functions, or attributes, we MUST create the new name and MUST deprecate the old
  one in one version (step 1), and MAY remove it in a subsequent version (step 2). For simple renames, the old name
  SHALL call the new one.
- when a feature is being replaced in favor of an existing one, we MUST mark a feature as deprecated in one version, and
  MAY remove it in a subsequent one.

Deprecation notice SHOULD contain a version starting from which the deprecation takes effect for tracking purposes. For
example, if `GetFoo` function is going to be deprecated in `v0.45.0` version, it gets the following godoc line:

```golang
package test

// Deprecated: [v0.45.0] Use MustDoFoo instead.
func DoFoo() {}
```

#### Example #1 - Renaming a function

1. Current version `v0.N` has `func GetFoo() Bar`
1. We now decided that `GetBar` is a better name. As such, on `v0.N+1` we add a new `func GetBar() Bar` function,
   changing the existing `func GetFoo() Bar` to be an alias of the new function. Additionally, a log entry with a
   warning is added to the old function, along with an entry to the changelog.
1. On `v0.N+2`, we MAY remove `func GetFoo() Bar`.

#### Example #2 - Changing the return values of a function

1. Current version `v0.N` has `func GetFoo() Foo`
1. We now need to also return an error. We do it by creating a new function that will be equivalent to the existing one
   so that current users can easily migrate to that: `func MustGetFoo() Foo`, which panics on errors. We release this in
   `v0.N+1`, deprecating the existing `func GetFoo() Foo` with it, adding an entry to the changelog and perhaps a log
   entry with a warning.
1. On `v0.N+2`, we change `func GetFoo() Foo` to `func GetFoo() (Foo, error)`.

#### Example #3 - Changing the arguments of a function

1. Current version `v0.N` has `func GetFoo() Foo`
2. We now decide to do something that might be blocking as part of `func GetFoo() Foo`, so, we start accepting a
   context: `func GetFooWithContext(context.Context) Foo`. We release this in `v0.N+1`, deprecating the existing `func
   GetFoo() Foo` with it, adding an entry to the changelog and perhaps a log entry with a warning. The existing `func
   GetFoo() Foo` is changed to call `func GetFooWithContext(context.Background()) Foo`.
3. On `v0.N+2`, we change `func GetFoo() Foo` to `func GetFoo(context.Context) Foo` if desired or remove it entirely if
   needed.

#### Exceptions

For changes to modules that do not have a version of `v1` or higher, we may skip the deprecation process described above
for the following situations. Note that these changes should still be recorded as breaking changes in the changelog.

* **Variadic arguments.** Functions that are not already variadic may have a variadic parameter added as a method of
  supporting optional parameters, particularly through the functional options pattern. If a variadic parameter is
  added to a function with no change in functionality when no variadic arguments are passed, the deprecation process
  may be skipped. Calls to updated functions without the new argument will continue to work before, but users who depend
  on the exact function signature as a type, for example as an argument to another function, will experience a
  breaking change. For this reason, the deprecation process should only be skipped when it is not expected that
  the function is commonly passed as a value.

### End-user impacting changes

For end user breaking changes, we follow the [feature gate](https://github.com/open-telemetry/opentelemetry-collector/tree/main/featuregate#feature-lifecycle)
approach. This is a well-known approach in other projects such as Kubernetes. A feature gate has
three stages: alpha, beta and stable. The intent of these stages is to decouple other software
changes from the breaking change; some users may adopt the change early, while other users may delay
its adoption.

#### Feature gate IDs

Feature gate IDs should be namespaced using dots to denote the hierarchy. The namespace should be as
specific as possible; in particular, for feature gates specific to a certain component the ID should
have the following structure: `<component kind>.<component type>.<base ID>`. The "base ID" should be
written with a verb that describes what happens when the feature gate is enabled. For example, if
you want to add a feature gate for the OTLP receiver that changes the default endpoint to bind to an
unspecified host, you could name your feature gate `receiver.otlp.UseUnspecifiedHostAsDefaultHost`.

#### Lifecycle of a breaking change

##### Alpha stage

At the alpha stage, the change is opt-in. At this stage we want to notify users that a change is
coming, so they can start preparing for it and we have some early adopters that provide us with
feedback. Consider the following items before the initial release of an alpha feature gate:

* If **docs and examples** can be updated in a way that prevents the breaking change from affecting
  users, this is the time to update them!
* Provide users with tools to understand the breaking change
  * (Optional) Create or update a **Github issue** to document what the change is about, who it
    affects and what its effects are
  * (Optional) Consider adding **telemetry** that allows users to track their migration. For
    example, you can add a counter for the times that you see a payload that would be affected by
    the breaking change.
* Notify users about the upcoming change
  * Add a **changelog entry** that describes the feature gate. It should include its name, when you
    may want to use it, and what its effects are. The changelog entry can be given the `enhancement`
    classification at this stage.
  * (Optional but strongly recommended) Log a **warning** if the user is using the software in a way
    that would be affected by the breaking change. Point the user to the feature gate and any
    official docs.
* (Optional) Try to **test this in a realistic setting.** If this solves an issue, ask the poster to
  try to use it and check that everything works.

##### Beta stage

At the beta stage, the change is opt-out. At this stage we want to notify users that the change is
happening, and help them understand how to revert back to the previous behavior temporarily if they
need to do so. You may directly start from this stage for breaking changes that are less impactful
or for changes that should not have a functional impact such as performance changes. Consider the
following items before moving from alpha to beta:

* Schedule the **docs and examples** update to align with the breaking change release if you
  couldn’t do it before
* Provide users with tools to understand the breaking change
  * Update the **Github issue** with the new default behavior (or create one if starting from here)
  * Update the feature gate to add the ‘to version’ to the feature gate
* Notify users about the change
  * Add a second **changelog entry** that describes the change one more time and is marked as
    ‘breaking’.
  * If applicable, add an **error message** that tells you this is the result of a breaking change
    that can be temporarily reverted disabling the feature gate and points to any issue or docs
    about it.

##### Stable stage

At the stable stage, the change cannot be reverted. In some cases, you may directly start here and
just do the change, in which case you do not need a feature gate, but you should still follow the
checklist below (notify, update docs and examples). Consider the following items before moving from
beta to stable:

* Remove the dead code
* Provide users with tools to understand the breaking change
  * Update the **documentation** **and examples** to remove any references to the feature gate and
    the previous behavior. Close the **Github issue** if you opened one before.
* Notify users about the change
  * Add one last **changelog entry** so users know the range where the feature gate was in beta
  * Amend the **error message** to remove any references to the feature gate.

## Specification Tracking

The [OpenTelemetry Specification](https://github.com/open-telemetry/opentelemetry-specification) can be a rapidly
moving target at times.  While it may seem efficient to get an early start on implementing new features or
functionality under development in the specification, this can also lead to significant churn and a risk that
changes in the specification can result in breaking changes to the implementation.  For this reason, it is the
policy of the Collector SIG to not implement, or accept implementations of, new or changed specification language
prior to inclusion in a stable release of the specification.
