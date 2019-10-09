# Contributing Guide

We'd love your help!

## Report a bug or requesting feature

Reporting bugs is an important contribution. Please make sure to include:

* Expected and actual behavior
* OpenTelemetry version you are running
* If possible, steps to reproduce

## How to contribute

### Before you start

Please read project contribution
[guide](https://github.com/open-telemetry/community/blob/master/CONTRIBUTING.md)
for general practices for OpenTelemetry project.

Select a good issue from the links below (ordered by difficulty/complexity):

* [Good First Issue](https://github.com/open-telemetry/opentelemetry-collector/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
* [Up for Grabs](https://github.com/open-telemetry/opentelemetry-collector/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3Aup-for-grabs+)
* [Help Wanted](https://github.com/open-telemetry/opentelemetry-collector/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)

Comment on the issue that you want to work on so we can assign it to you and
clarify anything related to it.

If you would like to work on something that is not listed as an issue 
(e.g. a new feature or enhancement) please create an issue and describe your 
proposal. It is best to do this in advance so that maintainers can decide if 
the proposal is a good fit for this repository. This will help avoid situations 
when you spend significant time on something that maintainers may decide this 
repo is not the right place for.

Follow the instructions below to create your PR.

### Fork

In the interest of keeping this repository clean and manageable, you should
work from a fork. To create a fork, click the 'Fork' button at the top of the
repository, then clone the fork locally using `git clone
git@github.com:USERNAME/opentelemetry-service.git`.

You should also add this repository as an "upstream" repo to your local copy,
in order to keep it up to date. You can add this as a remote like so:

`git remote add upstream https://github.com/open-telemetry/opentelemetry-collector.git

# verify that the upstream exists
git remote -v`

To update your fork, fetch the upstream repo's branches and commits, then merge your master with upstream's master:

```
git fetch upstream
git checkout master
git merge upstream/master
```

Remember to always work in a branch of your local copy, as you might otherwise
have to contend with conflicts in master.

Please also see [GitHub
workflow](https://github.com/open-telemetry/community/blob/master/CONTRIBUTING.md#github-workflow)
section of general project contributing guide.

## Required Tools

Working with the project sources requires the following tools:

1. [git](https://git-scm.com/)
2. [go](https://golang.org/) (version 1.12.5 and up)
3. [make](https://www.gnu.org/software/make/)
4. [docker](https://www.docker.com/)

## Repository Setup

Fork the repo, checkout the upstream repo to your GOPATH by:

```
$ GO111MODULE="" go get -d github.com/open-telemetry/opentelemetry-collector
```

Add your fork as an origin:

```shell
$ cd $(go env GOPATH)/src/github.com/open-telemetry/opentelemetry-collector
$ git remote add fork git@github.com:YOUR_GITHUB_USERNAME/opentelemetry-service.git
```

Run tests, fmt and lint:

```shell 
$ make install-tools # Only first time.
$ make
```

*Note:* the default build target requires tools that are installed at `$(go env GOPATH)/bin`, ensure that `$(go env GOPATH)/bin` is included in your `PATH`.

## Creating a PR

Checkout a new branch, make modifications, build locally, and push the branch to your fork
to open a new PR:

```shell
$ git checkout -b feature
# edit
$ make
$ git commit
$ git push fork feature
```

## General Notes

This project uses Go 1.12.5 and Travis for CI.

Travis CI uses the Makefile with the default target, it is recommended to
run it before submitting your PR. It runs `gofmt -s` (simplify) and `golint`.

The dependencies are managed with `go mod` if you work with the sources under your
`$GOPATH` you need to set the environment variable `GO111MODULE=on`.
