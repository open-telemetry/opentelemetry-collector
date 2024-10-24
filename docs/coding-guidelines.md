# Coding guidelines

We consider the OpenTelemetry Collector to be close to production quality and the quality bar
for contributions is set accordingly. Contributions must have readable code written
with maintainability in mind (if in doubt check [Effective Go](https://golang.org/doc/effective_go.html)
for coding advice). The code must adhere to the following robustness principles that
are important for software that runs autonomously and continuously without direct
interaction with a human (such as this Collector).

### Naming convention

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

### Enumerations

To keep naming patterns consistent across the project, enumeration patterns are enforced to make intent clear:

- Enumerations should be defined using a type definition, such as `type Level int32`.
- Enumerations should use either `int` or `string` as the underlying type
- The enumeration name should succinctly describe its purpose
  - If the package name represents the entity described by the enumeration then the package name should be factored into the name of the enumeration.  For example, `component.Type` instead of `component.ComponentType`.
  - The name should convey a sense of limited categorization. For example, `pcommon.ValueType` is better than `pcommon.Value` and `component.Kind` is better than `component.KindType`, since `Kind` already conveys categorization.
- Constant values of an enumeration should be prefixed with the enumeration type name in the name:
  - `pcommon.ValueTypeStr` for `pcommon.ValueType`
  - `pmetric.MetricTypeGauge` for `pmetric.MetricType`


### Recommended Libraries / Defaults

In order to simplify development within the project, we have made certain library recommendations that should be followed.

| Scenario 	 | Recommended                   	                | Rationale                                                                                                                  |
|------------|------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| Hashing  	 | ["hashing/fnv"](https://pkg.go.dev/hash/fnv) 	 | The project adopted this as the default hashing method due to the efficiency and is reasonable for non-cryptographic use 	 |
| Testing  	 | Use `t.Parallel()` where possible            	 | Enabling more tests to be run in parallel will speed up the feedback process when working on the project.                 	 |


Within the project, there are some packages that have yet to follow the recommendations and are being addressed. However, any new code should adhere to the recommendations.

### Default Configuration

To guarantee backward-compatible behavior, all configuration packages should supply a `NewDefault[config name]` functions that create a default version of the config. The package does not need to guarantee that `NewDefault[config name]` returns a usable configuration—only that default values will be set. For example, if the configuration requires that a field, such as `Endpoint` be set, but there is no valid default value, then `NewDefault[config name]` may set that value to `""` with the expectation that the user will set a valid value.

Users should always initialize the config struct with this function and overwrite anything as needed.

### Startup Error Handling

Verify configuration during startup and fail fast if the configuration is invalid.
This will bring the attention of a human to the problem as it is more typical for humans
to notice problems when the process is starting as opposed to problems that may arise
sometime (potentially long time) after process startup. Monitoring systems are likely
to automatically flag processes that exit with failure during startup, making it
easier to notice the problem. The Collector should print a reasonable log message to
explain the problem and exit with a non-zero code. It is acceptable to crash the process
during startup if there is no good way to exit cleanly but do your best to log and
exit cleanly with a process exit code.

### Propagate Errors to the Caller

Do not crash or exit outside the `main()` function, e.g. via `log.Fatal` or `os.Exit`,
even during startup. Instead, return detailed errors to be handled appropriately
by the caller. The code in packages other than `main` may be imported and used by
third-party applications, and they should have full control over error handling
and process termination.

### Do not Crash after Startup

Do not crash or exit the Collector process after the startup sequence is finished.
A running Collector typically contains data that is received but not yet exported further
(e.g. data that is stored in the queues and other processors). Crashing or exiting the Collector
process will result in losing this data since typically the receiver has
already acknowledged the receipt for this data and the senders of the data will
not send that data again.

### Bad Input Handling

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

### Error Handling and Retries

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

### Logging

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

### Executing External Processes

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

### Observability

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

### Resource Usage

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

### Graceful Shutdown

Collector does not yet support graceful shutdown but we plan to add it. All components
must be ready to shutdown gracefully via `Shutdown()` function that all component
interfaces require. If components contain any temporary data they need to process
and export it out of the Collector before shutdown is completed. The shutdown process
will have a maximum allowed duration so put a limit on how long your shutdown
operation can take.

### Unit Tests

Cover important functionality with unit tests. We require that contributions
do not decrease the overall code coverage of the codebase - this is aligned with our
goal to increase coverage over time. Keep track of execution time for your unit
tests and try to keep them as short as possible.

#### Testing Library Recommendations

To keep testing practices consistent across the project, it is advised to use these libraries under
these circumstances:

- For assertions to validate expectations, use `"github.com/stretchr/testify/assert"`
- For assertions that are required to continue the test, use `"github.com/stretchr/testify/require"`
- For mocking external resources, use `"github.com/stretchr/testify/mock"`
- For validating HTTP traffic interactions, `"net/http/httptest"`

### Integration Testing

Integration testing is encouraged throughout the project, container images can be used in order to facilitate
a local version. In their absence, it is strongly advised to mock the integration.

### Using CGO

Using CGO is prohibited due to the lack of portability and complexity
that comes with managing external libraries with different operating systems and configurations.
However, if the package MUST use CGO, this should be explicitly called out within the readme
with clear instructions on how to install the required libraries.
Furthermore, if your package requires CGO, it MUST be able to compile and operate in a no-op mode
or report a warning back to the collector with a clear error saying CGO is required to work.

### Breaking changes

Whenever possible, we adhere to [semver](https://semver.org/) as our minimum standards. Even before v1, we strive not to break compatibility
without a good reason. Hence, when a change is known to cause a breaking change, it MUST be clearly marked in the
changelog and SHOULD include a line instructing users how to move forward.

We also strive to perform breaking changes in two stages, deprecating it first (`vM.N`) and breaking it in a subsequent
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

#### End-user impacting changes

When deprecating a feature affecting end-users, consider first deprecating the feature in one version, then hiding it
behind a [feature
gate](https://github.com/open-telemetry/opentelemetry-collector/blob/6b5a3d08a96bfb41a5e121b34f592a1d5c6e0435/service/featuregate/)
in a subsequent version, and eventually removing it after yet another version. This is how it would look like, considering
that each of the following steps is done in a separate version:

1. Mark the feature as deprecated, add a short-lived feature gate with the feature enabled by default
1. Change the feature gate to disable the feature by default, deprecating the gate at the same time
1. Remove the feature and the gate

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

#### Configuration changes

##### Alpha components

Configuration for alpha components can be changed with minimal notice. Documenting them as part of the changelog is
sufficient. We still recommend giving users one or two minor versions' notice before breaking the configuration, such as
when removing or renaming a configuration option. Providing a migration path in the component's repository is NOT
required for alpha components, although it is still recommended.

- when adding a new configuration option, components MAY mark the new option as required and are not required to provide
  a reasonable default.
- when renaming a configuration option, components MAY treat the old name as an alias to the new one and log a WARN
  level message in case the old option is being used.
- when removing a configuration option, components MAY keep the old option for a few minor releases and log a WARN level
  message instructing users to remove the option.

##### Beta components

One of the requirements for a component to be marked as beta is to have its configuration options stabilized. Therefore,
backward incompatible changes should be rare events for beta components. Users of those components are not expecting to
have their Collector instances failing at startup because of a configuration change. When doing backward incompatible
changes, component owners should add the migration path to a place within the component's repository, linked from the
component's main README. This is to ensure that people using older instructions can understand how to migrate to the
latest version of the component.

When adding a new required option:
- the option MUST come with a sensible default value

When renaming or removing a configuration option:
- the option MUST be deprecated in one version
- a WARN level message should be logged, with a link to a place within the component's repository where the change is
  documented and a migration path is provided
- the option MUST be kept for at least N+1 version and MAY be hidden behind a feature gate in N+2
- the option and the WARN level message MUST NOT be removed earlier than N+2 or 6 months, whichever comes later

Additionally, when removing an option:
- the option MAY be made non-operational already by the same version where it is deprecated

##### Stable components

Stable components MUST be compatible between minor versions unless critical security issues are found. In that case, the
component owner MUST provide a migration path and a reasonable time frame for users to upgrade. The same rules from beta
components apply to stable when it comes to configuration changes.

### Specification Tracking

The [OpenTelemetry Specification](https://github.com/open-telemetry/opentelemetry-specification) can be a rapidly
moving target at times.  While it may seem efficient to get an early start on implementing new features or
functionality under development in the specification, this can also lead to significant churn and a risk that
changes in the specification can result in breaking changes to the implementation.  For this reason, it is the
policy of the Collector SIG to not implement, or accept implementations of, new or changed specification language
prior to inclusion in a stable release of the specification.
