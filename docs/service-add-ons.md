# OpenTelemetry Service: Add-On Components

Besides the pipeline elements (receivers, processors, and exporters) the OTelSvc
uses various ad-hoc “add-on” components (e.g.: healthcheck, z-pages, etc). 
However, the interface and configuration for such components was not specified 
and the service code directly handles them individually in an ad-hoc fashion. 
This document proposes a design to make these “add-ons” another extension point 
of OTelSvc, in a similar fashion to pipeline elements.

## Design Goals

*   Allow same functionality provided by the existing ad-hoc components;
*   Make configuration of add-ons components consistent with new model used for 
pipeline elements;
*   Make add-ons extensible in the same way as pipeline elements;


## Configuration and Interface

The configuration will follow the same pattern used for pipelines: a base 
configuration type and the creation of factories to instantiate the AddOn objects.

In order to support generic add-on components an interface needs to be defined 
so the service can interactly uniformly with them. This interface needs to cover 
at minimum Start and Shutdown. 

In addition to this base interface there will be support to notify add-ons when 
pipelines are “ready” and when they are about to be stopped, i.e.: “not ready” 
to receive data. These are a necessary addition to allow implementing add-ons 
that indicate to LBs and external systems if the service instance is ready or 
not to receive data 
(e.g.: a [k8s readiness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-readiness-probes)). 
These state changes will be under the control of the service application hosting 
the add-ons.

There are more complex scenarios in which there can be notifications of state 
changes from the add-ons to their host. These more complex cases are not going 
to be covered in the first implementation for generic add-on components, but 
this design won’t prevent such extensions in the future[^1].


## Service State and Add-Ons

The diagram below shows the basic state transitions of the OpenTelemetry Service 
and how it will interact with the service add-ons.


## Configuration

The config package will be extended to load the add-ons components when the 
configuration is loaded. The settings for add-on components will live in the 
same configuration file as the pipeline elements. Below is an example of how 
these sections would look like in the configuration file:

```yaml

# Example of the component add-ons provided in OTelSvc core. This list below
# includes all, currently, configurable options and their respective default
# value.
add-ons:
  health-check:
    endpoint**: "0.0.0.0:13133"
  pprof:
    endpoint**: "0.0.0.0:1777"
    block-profile-fraction: 0
    mutex-profile-fraction: 0
  zpages:
   endpoint: "0.0.0.0:55679"

# The service lists components not directly related to data pipelines, but used
# by the service.
service:
  # add-ons lists the add-on components added to the service. They are started
  # in the order presented below and stopped in the reverse order.
  add-ons: [health-check, pprof, zpages]
```

The current command-line related flags will be removed as the components are 
moved to the new model. Command-line flags will be limited to settings directly 
affecting the service as a whole, e.g.: logging.

The configuration base type won’t share any common fields. Existing ad-hoc 
components use endpoint but it is possible to have some other add-ons that do 
not use that configuration at all.

The factory will follow the pattern established for pipeline configuration:

```go
// Factory is a factory interface for add-ons to the service.
type Factory interface {
    // Type gets the type of the add-on component created by this factory.
    Type() string 

    // CreateDefaultConfig creates the default configuration for the add-on.
    CreateDefaultConfig() configmodels.AddOn 

    // CustomUnmarshaler returns a custom unmarshaler for the configuration or nil if
    // there is no need for custom unmarshaling. This is typically used if viper.Unmarshal()
    // is not sufficient to unmarshal correctly.
    CustomUnmarshaler(v *viper.Viper, viperKey string, intoCfg interface{}) CustomUnmarshaler 

    // CreateAddOn creates a service add-on based on the given config.
    CreateAddOn(logger *zap.Logger, cfg configmodels.AddOn) (Component, error)
}
```


## Add-On Component Interface

The interface defined below is the minimum required to keep same behavior for 
ad-hoc components currently in use on the service:

```go
// Component is the interface for objects hosted by the OpenTelemetry Service that
// doesn't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Component interface {
    // Start the Component object hosted by the given host. At this point in the
    // process life-cycle the receivers are not started and the host did not
    // receive any data yet.
    Start(host Host) error 

    // Shutdown the Component instance. This happens after the pipelines were
    // shutdown.
    Shutdown() error
}

// PipelineWatcher is an extra interface for Components hosted by the OpenTelemetry
// Service that is to be implemented by Components interested in changes to pipeline
// states. Typically this will be used by Components that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
    // Ready notifies the Component that all pipelines were built and the
    // receivers were started, i.e.: the service is ready to receive data
    // (notice that it may already have received data when this method is called).
    Ready() error 

    // NotReady notifies the Component that all receivers are about to be stopped,
    // i.e.: pipeline receivers will not accept new data.
    // This is sent before receivers are stopped, so the Component can take any
    // appropriate action before that happens.
    NotReady() error
}

// Host represents the entity where the add-on is being hosted. It is used to
// allow communication between the add-on and its host.
type Host interface {
    // ReportFatalError is used to report to the host that the add-on encountered
    // a fatal error (i.e.: an error that the instance can't recover from) after
    // its start function had already returned.
    ReportFatalError(err error)
}
```

## Ad-Hoc Components to be Ported to Add-On Components

The following ad-hoc components need to be moved to the new format:

*   Health check (will be a wrapper around github.com/jaegertracing/jaeger/pkg/healthcheck)
*   PProf (from internal/pprofserver/)
*   ZPages (from zpages/)

## Notes

[^1]:
     This can be done by adding specific interfaces to AddOn types that support 
     those and having the service checking which of the AddOn instances support 
     each interface.
