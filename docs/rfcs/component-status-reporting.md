# Component Status Reporting

## Overview

Since the OpenTelemetry Collector is made up of pipelines with components, it needs a way for the components within those pipelines to emit information about their health. This information allows the collector service, or other interested software or people, to make decisions about how to proceed when something goes wrong. This document describes:

1. The historical state of how components reported health
2. The current state of how components report health
3. The goals component health reporting should achieve
4. Existing deviations from those goals
5. Desired behavior for 1.0

For the sake of simplicity, when `Host` is referenced in this document you can think of the Collector service, which manages all the collector pipelines and the components within.

## Out Of Scope

How to get from the current to desired behavior is also considered out of scope and will be discussed on individual PRs. It will likely involve one or multiple feature gates, warnings and transition periods.

## The Collector’s Historical method of reporting component health
Until recently, the Collector relied on four ways to report health.

1. The `error` returned by the Component’s Start method. During startup, if any component decided to return an error, the Collector would stop gracefully.
2. The `Host.ReportFatalError` method.  This method let components tell the Host that something bad happened and the collector needed to shut down.  While this method could be used anywhere in the component, it was primarily used with a Component’s Start method to report errors in async work, such as starting a server.
    ```golang
    if errHTTP := fmr.server.Serve(listener); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
       host.ReportFatalError(errHTTP)
    }
    ```
3. The error returned by `Shutdown`. This error was indicative that the collector did not cleanly shut down, but did not prevent the shutdown process from moving forward.

4. Panicking. During runtime, if the collector experienced an unhandled error, it crashes.

These are all the way the components in a collector could report that they were unhealthy.

There are several major gaps in the Collector’s historic reporting of component health. First, many components return recoverable errors from Start, causing the collector to shutdown, while it could recover if the collector was allowed to run. Second, when a component experienced a transient error, such as an endpoint suddenly not working, the component would simply log the error and return it up the pipeline. There was no mechanism for the component to tell the Host or anything else that something was going wrong. Last, when a component experienced an issue it would never be able to recover from, such as receiving a 404 response from an endpoint, the component would log the error and return it up the pipeline.  This situation was handled in the same way as the transient error, which means the component could not tell the Host or anything else that something was wrong, but worse is that the issue would never get better.

## Current State of Component Health Reporting

See [Component Status Reporting](../component-status.md)

## The Goals the Component Health Reporting Should Achieve

The following are the goals, as of June 2024 and with Collector 1.0 looming, for a component health reporting system.

1. Runtime component health reporting, which cannot be automatically reported, must be opt-in for components.  Collector components must not be required to use the component health reporting system. This keeps component as compatible as possible with the Collector’s framework as we approach 1.0
    - The consumers of the health reporting system must be able to identify which components are and are not using the health reporting system.
2. Component health reporting must be opt-in for collector users.  While the underlying components are always allowed to report their health via the system, the Collector Host or any other listener may only take action when the user has configured the collector accordingly.
   - As one example of compliance, the current health reporting system is dependent on the user configuring an extension that can watch for status updates.
3. Component health must be representable as a finite state machine with clear transitions between states.
4. The Collector Host may report statuses Starting, Ok,  Stopping and PermanentError on behalf of components as documented in the automation section.
    - Additional status may be reported in the future
5. Component health reporting must only be a mechanism for reporting health - it should have no mechanisms for taking actions on the health it reports.  How consumers of the health reporting system respond to component updates is not a concern of the health reporting system.

## Existing deviations from those goals

### Fatal Error Reporting

Before the current implementation of component status reporting, a component could stop the collector by using `Host.ReportFatalError`. Now, a component MUST use component status reporting and emit a `FatalError`. This fact is in conflict with Goal 1, which states component health reporting must be opt-in for components.

A couple solutions:
1. Accept this reality as an exception to Goal 1.
2. Add back `Host.ReportFatalError`.
3. Remove the ability for components to stop the collector be removing `FatalError`.

### No way to identify components that are not reporting status
Goal 1 states that consumers of component status reporting must be able to identify components in use that have not opted in to component status reporting. Our current implementation does not have this feature.

### Should component health reporting be an opt-in for `component.Host` implementations?

The current implementation of component status reporting does not add anything to `component.Host` to force a `component.Host` implementation, such as `service.Service`, to be compatible with component status reporting.  Instead, it adds `ReportStatus func(*StatusEvent)` to `component.TelemetrySettings` and things that instantiate components, such as `serivce.Service`, should, but are not required, to pass in a value for `ReportStatus`.

As a result, `component.Host` implementation is not required to engage with the component status reporting system.  This could lead to situations where a user adds a status watcher extension that can do nothing because the `component.Host` is not reporting component status updates.

Is this acceptable? Should we:
1. Require the `component.Host` implementations be compatible with the component status reporting framework?
2. Add some sort of configuration/build flag then enforces the `component.Host` implementation be compatible (or not) with component status reporting?
3. Accept this edge case.

### Component TelemetrySettings Requirements

The current implementation of component status reporting added a new field to `component.TelemetrySettings`, `ReportStatus`.  This field is technically optional, but would be marked as stable with component 1.0. Are we ok with 1 of the following?

1. Including a component status reporting feature, `component.TelemetrySettings.ReportStatus`, in the 1.0 version of `component.TelemetrySettings`?
2. Marking `component.TelemetrySettings.ReportStatus` as experimentatal via godoc comments in the 1.0 version of `component.TelemetrySettings`?

Or should we refactor `component` somehow to remove `ReportStatus` from `component.TelemetrySettings`?

## Desired Behavior for 1.0

All deviations for the goals should be addressed so that component 1.0 can be released with a non-GA implementation of component status reporting.

## Reference
- Remove FatalError? Looking for opinions either way: https://github.com/open-telemetry/opentelemetry-collector/issues/9823
- In order to prioritize lifecycle events over runtime events for status reporting, allow a component to transition from PermanentError -> Stopping: https://github.com/open-telemetry/opentelemetry-collector/issues/10058
- Runtime status reporting for components in core: https://github.com/open-telemetry/opentelemetry-collector/issues/9957
- Should Start return an error: https://github.com/open-telemetry/opentelemetry-collector/issues/9324
- Should Shutdown return an error: https://github.com/open-telemetry/opentelemetry-collector/issues/9325
- Status reporting doc incoming; preview here: https://github.com/mwear/opentelemetry-collector/blob/cc870fd2a7160da298acdda447511ea9a83455e0/docs/component-status.md
- Issues
  - Closed: https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/8349
  - Open: https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/8816
- Status Reporting PRs
  - Closed
    - https://github.com/open-telemetry/opentelemetry-collector/pull/5304
    - https://github.com/open-telemetry/opentelemetry-collector/pull/6550
    - https://github.com/open-telemetry/opentelemetry-collector/pull/6560
  - Merged
    - https://github.com/open-telemetry/opentelemetry-collector/pull/8169




