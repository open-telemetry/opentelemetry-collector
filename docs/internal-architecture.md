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
