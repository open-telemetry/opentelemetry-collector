# Partial Reload for the OpenTelemetry Collector

Support partial configuration reload to rebuild only affected components, reducing event loss and downtime during configuration changes. Rather than tearing down and rebuilding the entire pipeline graph on every configuration update, the collector will diff the old and new configurations and selectively restart only the components that have changed. This will be rolled out incrementally behind a feature flag, starting with receivers and expanding to all component types.

## Motivation

Currently, when a configuration change is applied to the OpenTelemetry Collector, the entire pipeline graph is torn down and rebuilt from scratch. This means every receiver, processor, exporter, and connector is stopped and restarted, regardless of whether it was affected by the change. This results in lost events during the reload window, downtime where the collector is not accepting or processing telemetry data, and unnecessary performance overhead from rebuilding components that did not change.

A survey of components in the collector-contrib repository reveals the full scope of this problem across four categories of impact:

### Accumulated State Loss

Many components maintain in-memory state (buffers, counters, accumulators, caches) that is permanently lost on restart. This causes metric gaps, trace loss, incorrect calculations, and false alerts visible to end users.

Affected components: `tailsamplingprocessor`, `groupbytraceprocessor`, `deltatocumulativeprocessor`, `cumulativetodeltaprocessor`, `intervalprocessor`, `logdedupprocessor`, `metricstarttimeprocessor`, `statsdreceiver`, `prometheusexporter`, `signalfxexporter`

### Cursor and Checkpoint Disruption

Components that track consumption progress via file offsets, poll cursors, or partition checkpoints lose their position on restart. This causes duplicate ingestion or missed data depending on how the cursor is recovered.

Affected components: `filelogreceiver`, `azureeventhubreceiver`, `awscloudwatchreceiver`, `googlecloudspannerreceiver`, `mongodbatlasreceiver`

### Expensive Reconnection and Authentication

Components with costly startup involving connection establishment, authentication handshakes, or large state synchronization. Startup cost ranges from seconds (TLS handshakes, OAuth token exchange) to minutes (Kubernetes cluster state sync).

Affected components: `k8sclusterreceiver`, `prometheusreceiver`, `googlecloudpubsubreceiver`, `solacereceiver`, `pulsarreceiver`, `kafkametricsreceiver`, `elasticsearchexporter`, `prometheusremotewriteexporter`, `datadogexporter`

### External System Disruption

Some components have a blast radius that extends beyond the collector itself, causing disruption to other systems and instances that are entirely unrelated to the configuration change.

Affected components: `kafkareceiver` (consumer group rebalance affects all group members), `loadbalancingexporter` (routing disruption breaks downstream tail sampling)

---

By enabling partial reload, the collector can intelligently determine which components are affected by a configuration change and only restart those specific components. Pipelines and components that are unaffected by the change can continue operating without interruption, significantly reducing the impact of configuration updates.

## Explanation

When partial reload is enabled via the `service.partialReload` feature flag, the collector compares the incoming configuration with the current running configuration to determine the minimal set of components that need to be rebuilt. Unaffected components continue running without interruption.

### Component Changes

- **Receiver modified**: Only the affected receiver is rebuilt. Downstream processors and exporters continue operating and will receive data from the new receiver instance once it starts.
- **Processor modified**: The processor and all receivers upstream of it in the pipeline are rebuilt. This ensures that no in-flight data is lost during the processor restart.
- **Exporter modified**: The exporter, along with all processors and receivers upstream of it in the pipeline, are rebuilt. This provides a clean data path from ingestion to export.

### Pipeline Changes

- **Pipeline added**: Only the new pipeline is instantiated and started. Existing pipelines continue operating without interruption.
- **Pipeline removed**: Only the removed pipeline is stopped and torn down. Other pipelines remain unaffected.
- **Pipeline modified**: When a receiver, processor, or exporter is added to or removed from an existing pipeline, only that pipeline is rebuilt. Other pipelines remain unaffected.

### Connector Changes

Connectors require special handling since they bridge two pipelines:

- **Connector added**: On the exporter side (the pipeline feeding into the connector), the entire upstream pipeline is rebuilt to establish the new data flow. On the receiver side (the pipeline receiving from the connector), only the portion of the pipeline starting from the connector is affected.
- **Connector removed**: On the exporter side, the pipeline is rebuilt to remove the connector as a destination. On the receiver side, the portion of the pipeline that was receiving from the connector is torn down. If the receiving pipeline has other receivers, those portions continue operating unaffected.

The general principle is that changes are propagated "up the graph" from the point of change, rebuilding components upstream while leaving downstream components untouched. This keeps the blast radius of any configuration change as small as possible.

### Telemetry and Extension Changes

Changes to telemetry or extension configurations will always result in a full reload, as these are service-level settings that affect the entire collector.

## Internal details

### Feature Flag

The partial reload functionality will be entirely behind a feature flag named `service.partialReload`. Only when this flag is explicitly enabled will the new partial reload code path be taken. When disabled, the collector continues to use the existing full reload behavior. This ensures that no existing behavior is changed by default. Once the feature has been validated as stable, the feature flag will be removed and partial reload will become the default behavior.

### Configuration Diff

On each configuration change event, the collector performs a diff between the current and new configurations using `reflect.DeepEqual` to keep the comparison simple:

1. Compare service-level settings (telemetry, extensions). If any have changed, fall back to full reload.
2. Compare each component type (receivers, processors, exporters, connectors) to identify which specific component instances have been added, removed, or modified.
3. For each pipeline, determine if the pipeline itself has changed (added, removed, or its component references modified).
4. Based on the diff, calculate the minimal set of components to rebuild using the "up the graph" propagation rule.

### Phased Rollout

The implementation will follow a phased approach to minimize risk:

Each phase will be validated independently before moving to the next. When a phase is not yet implemented, changes to those component types will fallback to a full reload.

#### Phase 1 - Receivers

Partial reload for receivers only, including adding, removing, and modifying receiver configurations. Since receivers are the initial senders in the pipeline graph, changes to them do not require rebuilding any other components in the pipeline. Downstream processors and exporters continue operating uninterrupted.

#### Phase 2 - Processors

Partial reload support for processors, including adding, removing, and modifying processor configurations. When a processor is changed, the receivers in that pipeline will also be re-created to ensure a clean data flow from ingestion through the updated processor.

#### Phase 3 - Exporters

Partial reload support for exporters, including adding, removing, and modifying exporter configurations. When an exporter is changed, the processors and receivers in that pipeline will also be re-created to ensure a clean data path from ingestion through to the updated exporter.

#### Phase 4 - Pipeline

Adding, removing, or updating entire pipelines. When a new pipeline is added, only that pipeline is created and started without affecting existing pipelines. When a pipeline is removed, only that pipeline is stopped and torn down. When a pipeline is updated, only that pipeline is rebuilt.

#### Phase 5 - Connectors

Partial reload support for connectors, including adding, removing, and modifying their configurations. Connectors bind two pipelines together, acting as an exporter in one pipeline and a receiver in another. Because of this, changes to a connector affect both pipelines. On the exporter side, the entire upstream pipeline is rebuilt to establish or remove the data flow. On the receiver side, the portion of the pipeline starting from the connector is rebuilt.

### Error Handling

- If partial reload fails mid-way (e.g., a new receiver fails to start), the graph may be in a partially modified state. In this case, the error is propagated and the collector exits, consistent with how full reload failures are handled today.
- If the configuration diff determines that the change cannot be handled by the currently implemented phase, the collector falls back to a full reload transparently.

### Corner Cases

- **Shared components**: A receiver instance may be shared across multiple pipelines. The diff logic must account for this and only rebuild the receiver once, reconnecting it to all relevant pipelines.
- **Connector bridging**: Connectors act as both an exporter in one pipeline and a receiver in another. Changes to connectors must propagate correctly to both sides.
- **Rapid successive reloads**: If a new configuration change arrives while a partial reload is in progress, the behavior matches the existing full reload semantics (queue the change).

## Trade-offs and mitigations

**Increased code complexity**: The partial reload logic adds new code paths for diffing configurations and selectively rebuilding components. This is mitigated by:
- Gating behind a feature flag so the new paths are only exercised when opted in.
- Not modifying any existing core reload logic. The partial reload is an entirely separate code path.
- Phased rollout that allows each component type to be validated independently.

**Partial failure states**: If a component fails to start during partial reload, the pipeline graph may be in an inconsistent state. This is mitigated by treating partial reload failures the same as full reload failures (propagate the error). Future work could explore rollback semantics.

**Configuration comparison overhead**: Diffing configurations on every change adds overhead. In practice, this overhead is negligible compared to the cost of tearing down and rebuilding components, and uses `reflect.DeepEqual` on configuration structs which is fast for typical collector configurations.

## Open questions

- What metrics and observability should be added to track partial reload performance and success rates?

## Future possibilities

- **Component hot reload**: Building on partial reload, a future enhancement could allow components to implement a `Reload` interface that accepts new configuration without requiring a stop/start cycle, further reducing disruption.
