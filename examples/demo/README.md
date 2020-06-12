# OpenTelemetry Collector Demo

*IMPORTANT:* This is a pre-released version of the OpenTelemetry Collector.

This demo presents the typical flow of observability data with multiple
OpenTelemetry Collectors deployed:

- Applications send data directly to a Collector configured to use fewer
 resources, aka the _agent_;
- The agent then forwards the data to Collector(s) that receive data from
 multiple agents. Collectors on this layer typically are allowed to use more
 resources and queue more data;
- The Collector then sends the data to the appropriate backend, in this demo
 Jaeger, Zipkin, and Prometheus;

This demo uses `docker-compose` and by default runs against the 
`otel/opentelemetry-collector-dev:latest` image. To run the demo, switch
to the `examples/demo` folder and run:

```shell
docker-compose up -d
```

The demo exposes the following backends:

- Jaeger at http://0.0.0.0:16686
- Zipkin at http://0.0.0.0:9411
- Prometheus at http://0.0.0.0:9090 

Notes:

- It may take some time for the application metrics to appear on the Prometheus
 dashboard;

To clean up any docker container from the demo run `docker-compose down` from 
the `examples/demo` folder.

### Using a Locally Built Image
Developers interested in running a local build of the Collector need to build a
docker image using the command below:

```shell
make docker-otelcol
```

And set an environment variable `OTELCOL_IMG` to `otelcol:latest` before 
launching the command `docker-compose up -d`.


