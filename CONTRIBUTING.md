# Contributing Guide

We'd love your help! Please join our weekly [SIG
meeting](https://github.com/open-telemetry/community#special-interest-groups).

## How to structure PRs to get expedient reviews?

We recommend that any PR (unless it is trivial) to be smaller than 500 lines
(excluding go mod/sum changes) in order to help reviewers to do a thorough and
reasonably fast reviews.

### When adding a new component

Consider submitting different PRs for (more details about adding new components
[here](#adding-new-components)) :

* First PR should include the overall structure of the new component:
  * Readme, configuration, and factory implementation usually using the helper
    factory structs.
  * This PR is usually trivial to review, so the size limit does not apply to
    it.
* Second PR should include the concrete implementation of the component. If the
  size of this PR is larger than the recommended size consider splitting it in
  multiple PRs.
* Last PR should enable the new component and add it to the `otelcontribcol`
  binary by updating the `components.go` file. The component must be enabled
  only after sufficient testing, and there is enough confidence in the
  stability and quality of the component.
* Once a new component has been added to the executable, please add the component 
  to the [OpenTelemetry.io registry](https://github.com/open-telemetry/opentelemetry.io#adding-a-project-to-the-opentelemetry-registry).

### Refactoring Work

Any refactoring work must be split in its own PR that does not include any
behavior changes. It is important to do this to avoid hidden changes in large
and trivial refactoring PRs.

## Report a bug or requesting feature

Reporting bugs is an important contribution. Please make sure to include:

* Expected and actual behavior
* OpenTelemetry version you are running
* If possible, steps to reproduce

## How to contribute

### Before you start

Please read project contribution
[guide](https://github.com/open-telemetry/community/blob/main/CONTRIBUTING.md)
for general practices for OpenTelemetry project.

Select a good issue from the links below (ordered by difficulty/complexity):

* [Good First Issue](https://github.com/open-telemetry/opentelemetry-collector/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
* [Up for Grabs](https://github.com/open-telemetry/opentelemetry-collector/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3Aup-for-grabs+)
* [Help Wanted](https://github.com/open-telemetry/opentelemetry-collector/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)

Comment on the issue that you want to work on so we can assign it to you and
clarify anything related to it.

If you would like to work on something that is not listed as an issue
(e.g. a new feature or enhancement) please first read our [vision](docs/vision.md) and
[roadmap](docs/roadmap.md) to make sure your proposal aligns with the goals of the
Collector, then create an issue and describe your proposal. It is best to do this
in advance so that maintainers can decide if the proposal is a good fit for
this repository. This will help avoid situations when you spend significant time
on something that maintainers may decide this repo is not the right place for.

Follow the instructions below to create your PR.

### Fork

In the interest of keeping this repository clean and manageable, you should
work from a fork. To create a fork, click the 'Fork' button at the top of the
repository, then clone the fork locally using `git clone
git@github.com:USERNAME/opentelemetry-collector.git`.

You should also add this repository as an "upstream" repo to your local copy,
in order to keep it up to date. You can add this as a remote like so:

`git remote add upstream https://github.com/open-telemetry/opentelemetry-collector.git`

Verify that the upstream exists:

`git remote -v`

To update your fork, fetch the upstream repo's branches and commits, then merge
your `main` with upstream's `main`:

```
git fetch upstream
git checkout main
git merge upstream/main
```

Remember to always work in a branch of your local copy, as you might otherwise
have to contend with conflicts in `main`.

Please also see [GitHub
workflow](https://github.com/open-telemetry/community/blob/main/CONTRIBUTING.md#github-workflow)
section of general project contributing guide.

## Required Tools

Working with the project sources requires the following tools:

1. [git](https://git-scm.com/)
2. [go](https://golang.org/) (version 1.17 and up)
3. [make](https://www.gnu.org/software/make/)
4. [docker](https://www.docker.com/)

## Repository Setup

Fork the repo, checkout the upstream repo to your GOPATH by:

```
$ git clone git@github.com:open-telemetry/opentelemetry-collector.git
```

Add your fork as an origin:

```shell
$ cd opentelemetry-collector
$ git remote add fork git@github.com:YOUR_GITHUB_USERNAME/opentelemetry-collector.git
```

Run tests, fmt and lint:

```shell
$ make install-tools # Only first time.
$ make
```

*Note:* the default build target requires tools that are installed at `$(go env
GOPATH)/bin`, ensure that `$(go env GOPATH)/bin` is included in your `PATH`.

## Creating a PR

Checkout a new branch, make modifications, build locally, and push the branch to your fork
to open a new PR:

```shell
$ git checkout -b feature
# edit
$ make
$ make fmt
$ git commit
$ git push fork feature
```

## General Notes

This project uses Go 1.17.* and CircleCI.

CircleCI uses the Makefile with the `ci` target, it is recommended to
run it before submitting your PR. It runs `gofmt -s` (simplify) and `golint`.

The dependencies are managed with `go mod` if you work with the sources under your
`$GOPATH` you need to set the environment variable `GO111MODULE=on`.

## Coding Guidelines

Although OpenTelemetry project as a whole is still in Alpha stage we consider
OpenTelemetry Collector to be close to production quality and the quality bar
for contributions is set accordingly. Contributions must have readable code written
with maintainability in mind (if in doubt check [Effective Go](https://golang.org/doc/effective_go.html)
for coding advice). The code must adhere to the following robustness principles that
are important for software that runs autonomously and continuously without direct
interaction with a human (such as this Collector).

### Startup Error Handling

Verify configuration during startup and fail fast if the configuration is invalid.
This will bring the attention of a human to the problem as it is more typical for humans
to notice problems when the process is starting as opposed to problems that may arise
sometime (potentially long time) after process startup. Monitoring systems are likely
to automatically flag processes that exit with failure during startup, making it
easier to notice the problem. The Collector should print a reasonable log message to
explain the problem and exit with a non-zero code. It is acceptable to crash the process
during startup if there is no good way to exit cleanly but do your best to log and
exit cleanly with a process exit code.

### Propagate Errors to the Caller

Do not crash or exit outside the `main()` function, e.g. via `log.Fatal` or `os.Exit`,
even during startup. Instead, return detailed errors to be handled appropriately
by the caller. The code in packages other than `main` may be imported and used by
third-party applications, and they should have full control over error handling
and process termination.

### Do not Crash after Startup

Do not crash or exit the Collector process after the startup sequence is finished.
A running Collector typically contains data that is received but not yet exported further
(e.g. is stored in the queues and other processors). Crashing or exiting the Collector
process will result in losing this data since typically the receiver has
already acknowledged the receipt for this data and the senders of the data will
not send that data again.

### Bad Input Handling

Do not crash on bad input in receivers or elsewhere in the pipeline.
[Crash-only software](https://en.wikipedia.org/wiki/Crash-only_software)
is valid in certain cases; however, this is not a correct approach for Collector (except
during startup, see above). The reason is that many senders from which Collector
receives data have built-in automatic retries of the _same_ data if no
acknowledgment is received from the Collector. If you crash on bad input
chances are high that after the Collector is restarted it will see the same
data in the input and will crash again. This will likely result in infinite
crashing loop if you have automatic retries in place.

Typically bad input when detected in a receiver should be reported back to the
sender. If it is elsewhere in the pipeline it may be too late to send a response
to the sender (particularly in processors which are not synchronously processing
data). In either case it is recommended to keep a metric that counts bad input data.

### Error Handling and Retries

Be rigorous in error handling. Don't ignore errors. Think carefully about each
error and decide if it is a fatal problem or a transient problem that may go away
when retried. Fatal errors should be logged or recorded in an internal metric to
provide visibility to users of the Collector. For transient errors come up with a
retrying strategy and implement it. Typically you will
want to implement retries with some sort of exponential back-off strategy. For
connection or sending retries use jitter for back-off intervals to avoid overwhelming
your destination when network is restored or the destination is recovered.
[Exponential Backoff](https://github.com/cenkalti/backoff) is a good library that
provides all this functionality.

### Logging

Log your component startup and shutdown, including successful outcomes (but don't
overdo it, keep the number of success message to a minimum).
This can help to understand the context of failures if they occur elsewhere after
your code is successfully executed.

Use logging carefully for events that can happen frequently to avoid flooding
the logs. Avoid outputting logs per a received or processed data item since this can
amount to very large number of log entries (Collector is designed to process
many thousands of spans and metrics per second). For such high-frequency events
instead of logging consider adding an internal metric and increment it when
the event happens.

Make log message human readable and also include data that is needed for easier
understanding of what happened and in what context.

### Observability

Out of the box, your users should be able to observe the state of your component. 
The collector exposes an OpenMetrics endpoint at `http://localhost:8888/metrics`
where your data will land.

When using the regular helpers, you should have some metrics added around key 
events automatically. For instance, exporters should have `otelcol_exporter_sent_spans` 
tracked without your exporter doing anything.

### Resource Usage

Limit usage of CPU, RAM or other resources that the code can use. Do not write code
that consumes resources in an uncontrolled manner. For example if you have a queue
that can contain unprocessed messages always limit the size of the queue unless you
have other ways to guarantee that the queue will be consumed faster than items are
added to it.

Performance test the code for both normal use-cases under acceptable load and also for
abnormal use-cases when the load exceeds acceptable many times. Ensure that
your code performs predictably under abnormal use. For example if the code
needs to process received data and cannot keep up with the receiving rate it is
not acceptable to keep allocating more memory for received data until the Collector
runs out of memory. Instead have protections for these situations, e.g. when hitting
resource limits drop the data and record the fact that it was dropped in a metric
that is exposed to users.

### Graceful Shutdown

Collector does not yet support graceful shutdown but we plan to add it. All components
must be ready to shutdown gracefully via `Shutdown()` function that all component
interfaces require. If components contain any temporary data they need to process
and export it out of the Collector before shutdown is completed. The shutdown process
will have a maximum allowed duration so put a limit on how long your shutdown
operation can take.

### Unit Tests

Cover important functionality with unit tests. We require that contributions
do not decrease overall code coverage of the codebase - this is aligned with our
goal to increase coverage over time. Keep track of execution time for your unit
tests and try to keep them as short as possible.

## Release

See [release](docs/release.md) for details.

## Contributing Images
If you are adding any new images, please use [Excalidraw](https://excalidraw.com). It's a free and open source web application and doesn't require any account to get started. Once you've created the design, while exporting the image, make sure to tick **"Embed scene into exported file"** option. This allows the image to be imported in an editable format for other contributors later.

## Common Issues

Build fails due to dependency issues, e.g.

```sh
go: github.com/golangci/golangci-lint@v1.31.0 requires
	github.com/tommy-muehle/go-mnd@v1.3.1-0.20200224220436-e6f9a994e8fa: invalid pseudo-version: git fetch --unshallow -f origin in /root/go/pkg/mod/cache/vcs/053b1e985f53e43f78db2b3feaeb7e40a2ae482c92734ba3982ca463d5bf19ce: exit status 128:
	fatal: git fetch-pack: expected shallow list
 ```

`go env GOPROXY` should return `https://proxy.golang.org,direct`. If it does not, set it as an environment variable:

`export GOPROXY=https://proxy.golang.org,direct`
