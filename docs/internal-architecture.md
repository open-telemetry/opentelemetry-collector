## Collector internal architecture
There are a few resources available to understand how the collector works:
### [Startup Diagram](#startup-diagram)
### [Architecture Docs](https://opentelemetry.io/docs/collector/architecture/)
### [Important Files](#important-files)
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
    I --> J("`**createNodes**
     Builds the node objects from pipeline configuration and adds to graph.  Also validates connectors`")
    I --> K("`**createEdges**
     Iterates through the pipelines and creates edges between components`")
    I --> L("`**buildComponents**
     Topological sort the graph, and create each component in reverse order`")
    L --> M(Receiver Factory) & N(Processor Factory) & O(Exporter Factory)
```
### Important Files
Here is a brief list of useful and/or important files that you may find valuable to glance through.
#### [collector.go](../otelcol/collector.go)
This file contains the main Collector struct and its constructor `NewCollector`.

`Collector.Run` starts the collector blocks until it shuts down.

`setupConfigurationComponents` is the "main" function responsible for startup - it orchestrates the loading of the 
configuration, the creation of the graph, and the starting of all the components.

#### [graph.go](../service/internal/graph/graph.go)
This file contains the internal graph representation of the pipelines.

`Build` is the constructor for a Graph object.  The method calls out to helpers that transform the graph from a config
to a DAG of components.  The configuration undergoes additional validation here as well, and is used to instantiate
the components of the pipeline.

`Graph.StartAll` starts every component in the pipelines.

`Graph.ShutdownAll` stops each component in the pipelines

#### [component.go](../component/component.go)
component.go outlines the abstraction of components within OTEL collector.  It provides details on the component 
lifecycle as well as defining the interface that components must fulfil.

#### Factories
Each component type contains a Factory interface along with its corresponding NewFactory function.
Implementations of new components use this NewFactory function in their implementation to register key functions with 
the collector.  For example, the collector uses this interface to give receivers a handle to a nextConsumer - 
representing where the receiver can send data to that it has received.
