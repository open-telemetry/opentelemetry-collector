# Scraper Controller Extensions

## Overview

This RFC proposes introducing an extension interface for scraper controllers to enable event-driven scraping patterns.
The current time-based scraper controller will be maintained with an option to disable the timer, and new configuration will allow specifying one or more scraper controller extensions that can trigger scrapes based on external events.

## Motivation

The current scraper controller implementation uses a time-based ticker to periodically invoke scrapers. While this works well for many use cases, there are scenarios where event-driven scraping would be more appropriate, such as:

1. **Webhook-driven scrapes**: On-demand profiling or detailed metric collection triggered by alerts (e.g., CPU/memory spikes)
2. **Job-queue based scraping**: Horizontal scaling where multiple collector instances consume scrape jobs from a shared queue
3. **External scheduling**: Using external schedulers (e.g. cron, Kubernetes CronJobs) to avoid mostly-idle, long-lived processes

## Current State

The scraper controller is implemented in `scraper/scraperhelper/internal/controller/controller.go`:

- Uses `ControllerConfig` with `CollectionInterval`, `InitialDelay`, and `Timeout` fields
- Creates a time-based ticker in `startScraping()` that periodically calls the scrape function
- Scrapers are created during controller initialization and started/shutdown with the controller
- The controller manages the lifecycle of multiple scrapers and coordinates their execution

## Proposed Changes

We propose to introduce a new extension interface for scraper controllers, enabling flexibility over how scrapes are triggered.
The timer-based scraping will remain the default behavior but can be disabled via configuration.

### 1. Extension Interface

Create a new extension interface (e.g. in `extension/extensionscrapercontroller`):

```golang
// ControllerExtension is an extension that provides a means of registering scrapers,
// and giving the extension control over when registered scrapers are invoked.
type ControllerExtension interface {
    extension.Extension

    // RegisterScraper registers a scraper with this controller extension.
    // The extension will invoke the provided scrape function according to its implementation.
    // Returns a registration handle that can be used to deregister the scraper.
    RegisterScraper(ctx context.Context, scraperID component.ID, scrapeFunc func(context.Context) error) (RegistrationHandle, error)
}

// RegistrationHandle provides a way to deregister a scraper from a controller extension
type RegistrationHandle interface {
    // Deregister removes the scraper from the controller extension
    Deregister(ctx context.Context) error
}
```

### 2. Configuration Changes

Modify `ControllerConfig` to support:
- Optional timer configuration (can be disabled)
- List of controller extension IDs

```golang
type ControllerConfig struct {
    // CollectionInterval sets how frequently the scraper should be called.
    // If zero or negative, the timer-based scraping is disabled.
    // At least one controller extension must be configured if timer is disabled.
    CollectionInterval time.Duration `mapstructure:"collection_interval"`

    // InitialDelay sets the initial start delay for the scraper timer.
    InitialDelay time.Duration `mapstructure:"initial_delay"`

    // Timeout is an optional value used to set scraper's context deadline.
    Timeout time.Duration `mapstructure:"timeout"`

    // Controllers specifies one or more controller extensions that
    // will control when scrapes are triggered. Extensions must implement
    // scraperhelper.ControllerExtension.
    Controllers []component.ID `mapstructure:"controllers"`
}
```

### 3. Controller Implementation Changes

Modify `controller.Controller` to:
- Support disabling the timer when `CollectionInterval <= 0`
- Register scrapers with configured controller extensions during `Start()`
- Deregister scrapers during `Shutdown()`
- Validate that at least one controller extension is configured if timer is disabled

## Use Cases

Below are use cases that motivate this change. Some of the use cases describe hypothetical
scrapers, such scraper-based profiling, that do not currently exist in the OpenTelemetry Collector.
They are included to illustrate how event-driven scraping could be beneficial.

### Use Case 1: Webhook-Driven Scraping

A collector receives webhook notifications when CPU usage exceeds a threshold and triggers an
on-demand profiling scrape of a pprof endpoint:

```yaml
extensions:
  webhook_scraper_controller:
    endpoint: http://0.0.0.0:8080
    path: /

receivers:
  # Hypothetical pprof receiver that scrapes pprof HTTP endpoints
  pprof/cpu_10s:
    endpoint: https://localhost:6060/debug/pprof?seconds=10
    controllers:
      - webhook_controller

service:
  extensions: [webhook_scraper_controller]
  pipelines:
    profiles:
      receivers: [pprof/cpu_10s]
      exporters: [debug]
```

### Use Case 2: Job Queue Based Scraping

Multiple collector instances consume scrape jobs from a shared queue:

```yaml
extensions:
  queue_scraper_controller:
    queue_type: redis
    queue_url: redis://queue:6379
    queue_name: httpcheck-jobs

receivers:
  httpcheck:
    collection_interval: 0
    targets:
      - ...
    controllers:
      - queue_controller
```

### Use Case 3: External Cron Scheduling (one-shot execution)

A Kubernetes CronJob runs the collector as a one-shot job that scrapes once and exits,
avoiding a mostly-idle long-lived process. A similar approach could be used with any
external scheduler, such as AWS EventBridge or Google Cloud Scheduler. In any case,
the collector is expected to run, perform the scrape, and then exit.

Multiple implementation options exist for this use case, and the best approach may
depend on the specific platform. For example, integrating with AWS EventBridge may
may involve running the collector as an AWS Lambda function, handling the relevant
invocation event.

For plain old cron, or Kubernetes CronJobs, two possible approaches are outlined below.

#### Option A: new "scrape" CLI command

```bash
# Kubernetes CronJob runs a new command, specifying the pipeline(s) and scraper(s)
# to run once and then exit.
otelcol scrape --config=config.yaml metrics hostmetrics
```

The collector would support a new `scrape` CLI command that:
- Loads the specified configuration
- Injects a controller extension into the specified scraper(s) that
  the command can invoke to trigger a scrape
- Triggers the scraper(s) once via the injected controller extension
- Exits after the scrape completes

#### Option B: self-contained, one-shot controller

```yaml
extensions:
  oneshot_controller:
    # Extension that triggers scrape on registration and signals shutdown after completion

receivers:
  hostmetrics:
    collection_interval: 0
    controllers:
      - oneshot_controller
```

The challenge with Option B is that it becomes difficult if not impossible to invoke multiple
scrapers, as they will start concurrently and independently. Consequently, there needs to be
some additional coordination mechanism to wait for all scrapers to complete before exiting.

## Open Questions

1. Should there be a way to pass dynamic configuration from the extension to the scraper, e.g. `targets` for httpcheck?
