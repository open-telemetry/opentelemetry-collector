# OpenCensus service

OpenCensus service is an experimental component that collects and
exports from the OpenCensus instrumented processes available from the
same host machine.

Some frameworks and ecosystems are now providing out-of-the-box
instrumentation by using OpenCensus but the user is still expected
to register an exporter in order to export data. This is a problem
during an incident. Even though our users can benefit from having
more diagnostics data coming out of services already instrumented
with OpenCensus, they have to modify their code to register an
exporter and redeploy. Asking our users recompile and redeploy is
not an ideal at an incident time.

OpenCensus service is trying to eliminate this requirement.

## Goals

* Allow enabling of configuring the exporters lazily. After deploying code,
optionally run a daemon on the host and it will read the
collected data and upload to the configured backend.
* Binaries can be instrumented without thinking about the exporting story.
Allows open source binary projects (e.g. web servers like Caddy or Istio Mixer)
to adopt OpenCensus without having to link any exporters into their binary.
* Easier to scale the exporter development. Not every language has to
implement support for each backend.
* Custom daemons containing only the required exporters compiled in can be created. 

## Future goals
* Providing cluster-wide collections and cluster-wide z-pages.
* Currently we provide no ways to push configuration changes such as the
default sampling rate. It might be a problem at troubleshooting time. 
We are planning to address this problem in the future.

## Usage

First, install opencensusd if you haven't.

```
$ go get github.com/census-instrumentation/opencensus-service/cmd/opencensusd
```

Create a config.yaml file in the current directory and modify
it with the exporter configuration. For example, following
configuration exports both to Stackdriver and Zipkin.


config.yaml:

```
stackdriver:
  project: "your-project-id"
  enableTraces: true

zipkin:
  endpoint: "http://localhost:9411/api/v2/spans"
```

Run the example application that collects traces and exports
to the daemon if it is running.

```
$ go run "$(go env GOPATH)/src/github.com/census-instrumentation/opencensus-service/example/main.go"
```

Run opencensusd:

```
$ opencensusd
```

You should be able to see the traces in Stackdriver and Zipkin.
If you stop the opencensusd, example application will stop exporting.
If you run it again, it will start exporting again.
