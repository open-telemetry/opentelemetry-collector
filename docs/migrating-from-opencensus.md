## Action Plan for Bootstraping from OpenCensus

### Goals
We need to bootstrap the OpenTelemetry Collector using the existing OpenCensus Service codebase. We agreed to split the Service codebase into 2 parts: core and contrib. This bootstrapping is a good opportunity to do the splitting by only including in the OpenTelemetry Collector core the minimum number of receivers and exporters and moving the rest of functionality to a contrib package (most vendor-specific code).

The contrib package and vendor-specific receivers and exporters will continue to be available and there is no intent to retire it. The intent is to have a clear decoupling in the codebase that facilitates independent contribution of new components in the future, allows to easily create customized versions of a Service and makes it clear that core contributors will be responsible for maintenance of the core while vendor-specific components will be maintained by corresponding vendors (note: this does not exclude dual participation at all - some developers will likely work for vendors and will also be core maintainers).

# Migration Tasks

This is the action plan that also shows the progress. Tick the boxes after the task is complete.

[X] Copy all commits from https://github.com/census-instrumentation/opencensus-service to https://github.com/open-telemetry/opentelemetry-service
Make sure commit history is preserved.

[X] Remove receivers and exporters that are not part of core. We will keep the following in the core:

- Prometheus
- Jaeger (agent and collector ones)
- Zipkin
- OpenCensus, temporarily until OpenTelemetry one is available (we may want to keep OC for longer to facilitate migrations)

[ ] Cleanly decouple `core` from `cmd` in the repository. `core` will contain all business logic. `cmd` will be just a main.go that executes the business logic and compiles to `otsvc` executable.

`otsvc` will will only include receivers and exporters which we consider to be part of the core. 

The new codebase will contain improvements which are already in progress and which are aimed at making the codebase extensible and enable the splitting to core and contrib. This includes 3 initiatives:

- Decoupling of receiver and exporter implementations from the core logic.

- Introduction of receiver and exporter factories that can be individually registered to activate them.

- Implementation of the [new configuration format](https://docs.google.com/document/d/1NeheFG7DmcUYo_h2vLtNRlia9x5wOJMlV4QKEK05FhQ/edit#) that makes use of factories and allows for greater flexibility in the configuration.

The functionally of the new `otsvc` will heavily lean on existing implementation and will be mostly a superset of the current agent/collector functionality when considering core receivers and exporters only (however we will allow deviations if it saves significant implementation effort and makes the service better).

[ ] Provide guidelines and example implementations for vendors to follow when they add new receivers and exporters to the contrib package.

[ ] Create a new repository for contrib and copy all commits from https://github.com/census-instrumentation/opencensus-service to https://github.com/open-telemetry/opentelemetry-service
Make sure commit history is preserved.

[ ] Cleanup the `contrib` repo to only contain additional vendor specific receivers and exporters.

(Note: alternatively `contrib` can be a directory in the main repo - this is still open for discussion).

[ ] Provide OpenCensus-to-OpenTelemetry Collector migration guidelines for end-users who want to migrate. This will include recommendations on configuration file migration. We will also consider the possibility to support old configuration format in the new binary.

This approach allows us to have significant progress towards 2 stated goals in our [vision document](./vision.md): unify the codebase for agent and collector and make the service more extensible.
