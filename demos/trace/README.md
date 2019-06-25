# OpenTelemetry Service Demo

*IMPORTANT:* This is a pre-released version of the OpenTelemetry Service and does not yet
implement full functionality required to run this demo.

For now, please use the [OpenCensus Service](https://github.com/census-instrumentation/opencensus-service).

Typical flow of tracing data with OpenTelemetry Service: tracing data initially received by OpenTelemetry Agent
and then sent to OpenTelemetry Collector using OC data format. The OpenTelemetry Collector then sends the data to the
tracing backend, in this demo Jaeger and Zipkin.

This demo uses `docker-compose` and runs against locally built docker images of OpenTelemetry Service. In
order to build the docker images use the commands below from the root of the repo:

```shell
make docker-otelsvc
```

To run the demo, switch to the `demos/trace` folder and run:

```shell
docker-compose up
```

Open `http://localhost:16686` to see the data on the Jaeger backend and `http://localhost:9411` to see
the data on the Zipkin backend.

To clean up any docker container from the demo run `docker-compose down` from the `demos/trace` folder.
