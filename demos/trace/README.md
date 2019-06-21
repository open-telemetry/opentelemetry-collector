# OpenCensus Service Demo

*IMPORTANT:* This is a pre-released version of the OpenTelemetry Service.
For now, please use the [OpenCensus Service](https://github.com/open-telemetry/opentelemetry-service).

Typical flow of tracing data with OpenCensus Service: tracing data initially received by OC Agent
and then sent OC Collector using OC data format. The OC Collector then sends the data to the
tracing backend, in this demo Jaeger and Zipkin.

This demo uses `docker-compose` and runs against locally built docker images of OC service. In
order to build the docker images use the commands below from the root of the repo:

```shell
make docker-collector
```

To run the demo, switch to the `demos/trace` folder and run:

```shell
docker-compose up
```

Open `http://localhost:16686` to see the data on the Jaeger backend and `http://localhost:9411` to see
the data on the Zipkin backend.

To clean up any docker container from the demo run `docker-compose down` from the `demos/trace` folder.
