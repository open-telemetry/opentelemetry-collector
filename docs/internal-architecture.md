## Internal architecture

This document describes the Collector internal architecture and startup flow. It can be helpful if you are starting to contribute to the Collector codebase.

For the end-user focused architecture document, please see the [opentelemetry.io's Architecture documentation](https://opentelemetry.io/docs/collector/architecture/).  While it is end user focused, it's still a good place to start if you're trying to learn about the Collector codebase.

### Startup Diagram
```mermaid
flowchart TD
    A("`**command.NewCommand**`") -->|1| B("`**updateSettingsUsingFlags**`")
    A --> |2| C("`**NewCollector**
    Creates and returns a new instance of Collector`")
    A --> |3| D("`**Collector.Run**
    Starts the collector and blocks until it shuts down`")
    D --> E("`**setupConfigurationComponents**`")
    E -->  |1| F("`**getConfMap**`")
    E ---> |2| G("`**Service.New**
     Initializes telemetry, then initializes the pipelines`")
    E --> |3| Q("`**Service.Start**
    1. Start all extensions.
    2. Notify extensions about Collector configuration
    3. Start all pipelines.
    4. Notify extensions that the pipeline is ready.
    `")
    Q --> R("`**Graph.StartAll**
    Calls Start on each component in reverse topological order`")
    G --> H("`**initExtensionsAndPipeline**
     Creates extensions and then builds the pipeline graph`")
    H --> I("`**Graph.Build**
     Converts the settings to an internal graph representation`")
    I --> |1| J("`**createNodes**
     Builds the node objects from pipeline configuration and adds to graph.  Also validates connectors`")
    I --> |2| K("`**createEdges**
     Iterates through the pipelines and creates edges between components`")
    I --> |3| L("`**buildComponents**
     Topological sort the graph, and create each component in reverse order`")
    L --> M(Receiver Factory) & N(Processor Factory) & O(Exporter Factory) & P(Connector Factory)
```
### Configuration Reload

The Collector supports reloading its configuration at runtime, triggered by a file system change event from a config provider or a `SIGHUP` signal. The reload is handled by `Collector.reloadConfiguration`, which by default performs a **full reload**: the entire `Service` (extensions, pipelines, telemetry) is shut down and rebuilt from scratch using the new configuration. This is simple and correct, but it disrupts all components — processors lose in-flight data, exporter queues are drained, and extensions are restarted — even when only a small part of the config changed.

```mermaid
flowchart TD
    A("`Config change or SIGHUP`") --> B("`**Collector.reloadConfiguration**`")
    B --> C("`Shutdown Service
    (extensions, pipelines, telemetry)`")
    C --> D("`**setupConfigurationComponents**
    Rebuild everything from new config`")
    D --> E("`Service running with new config`")
```

#### Partial Receiver Reload (Alpha)

When the `service.receiverPartialReload` feature gate is enabled (`--feature-gates=service.receiverPartialReload`), the Collector first checks whether the config change is limited to receivers. If only receiver configurations and/or pure-receiver entries in pipeline receiver lists have changed — and all other sections (processors, exporters, connectors, extensions, telemetry, pipeline structure) are identical — a partial reload is performed instead.

The partial reload path (`Collector.tryPartialReceiverReload`) delegates to `Graph.UpdateReceivers`, which runs a 9-phase algorithm:

1. **Collect** all current receiver nodes from pipelines (skipping connectors-as-receivers)
2. **Determine** the desired receiver set from the new pipeline configs
3. **Categorize** receivers into add, remove, and rebuild sets
4. **Shutdown** receivers that are being removed or rebuilt
5. **Remove** old receiver nodes and edges from the graph
6. **Create** new receiver nodes for added and rebuilt receivers
7. **Wire** edges from new receiver nodes to existing downstream capabilities nodes
8. **Build** new receiver components via their factories
9. **Start** new receivers

Processors, exporters, connectors, and extensions remain running and untouched throughout. If the config change affects anything beyond receivers, the partial reload is skipped and the standard full reload executes.

```mermaid
flowchart TD
    A("`Config change or SIGHUP`") --> B("`**Collector.reloadConfiguration**`")
    B --> C{"`Feature gate enabled
    AND receiver-only change?`"}
    C -->|Yes| D("`**tryPartialReceiverReload**
    Diff old/new config`")
    D --> E("`**Graph.UpdateReceivers**
    Shutdown old receivers,
    build and start new ones`")
    E --> F("`Service running with updated receivers
    (all other components unchanged)`")
    C -->|No| G("`Full reload
    (shutdown and rebuild everything)`")
    G --> F
```

### Where to start to read the code
Here is a brief list of useful and/or important files and interfaces that you may find valuable to glance through.
Most of these have package-level documentation and function/struct-level comments that help explain the Collector!

- [collector.go](../otelcol/collector.go)
- [graph.go](../service/internal/graph/graph.go)
- [component.go](../component/component.go)

#### Factories
Each component type contains a `Factory` interface along with its corresponding `NewFactory` function.
Implementations of new components use this `NewFactory` function in their implementation to register key functions with 
the Collector.  An example of this is in [receiver.go](../receiver/receiver.go).

For example, the Collector uses this interface to give receivers a handle to a `nextConsumer` -
which represents where the receiver will send its data next in its telemetry pipeline.
