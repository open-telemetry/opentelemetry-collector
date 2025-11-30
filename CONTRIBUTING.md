# Contributing Guide

We'd love your help! Please join our weekly [SIG
meeting](https://github.com/open-telemetry/community#special-interest-groups).

## Target audiences

The OpenTelemetry Collector has three main target audiences:

1. *End-users*, aiming to use an OpenTelemetry Collector binary.
1. *Component developers*, consuming the Go APIs to create components compatible with the OpenTelemetry Collector Builder.
1. *Collector library users*, consuming other Go APIs exposed by the opentelemetry-collector repository, for example to
   build custom distributions or other projects building on top of the Collector Go APIs.

When the needs of these audiences conflict, end-users should be prioritized, followed by component developers, and
finally Collector library users.

### End-users

End-users are the target audience for our binary distributions, as made available via the
[opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repository, as
well as distributions created using the [OpenTelemetry Collector
Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder). To them, stability in the
behavior is important, be it runtime, configuration, or [internal
telemetry](https://opentelemetry.io/docs/collector/internal-telemetry/). They are more numerous and harder to get in
touch with, making our changes to the Collector more disruptive to them than to other audiences. As a general rule,
whenever you are developing OpenTelemetry Collector components (extensions, receivers, processors, exporters,
connectors), you should have end-users' interests in mind. Similarly, changes to code within packages like `config` will
have an impact on this audience. Make sure to cause minimal disruption when doing changes here.

### Component developers

Component developers create new extensions, receivers, processors, exporters, and connectors to be used with the
OpenTelemetry Collector. They are the primary audience for the opentelemetry-collector repository's public Go API. A
significant part of them will contribute to opentelemetry-collector-contrib. In addition to the end-user aspect
mentioned above, this audience also cares about Go API compatibility of Go modules such as the ones in the `pdata`,
`component`, `consumer`, `confmap`, `exporterhelper`, `config*` modules and others, even though such changes wouldn't cause any
impact to end-users. See the [Breaking changes](docs/coding-guidelines.md#breaking-changes) section
in the coding guidelines for more information on how to perform changes affecting this audience.

### Collector library users

A third audience uses the OpenTelemetry Collector as a library to build their own distributions or other projects based
on the Collector. This audience is the main consumer of modules such as `service` or `otelcol`. They also share the same
concerns as component developers regarding Go API compatibility and are likewise interested in behavior stability. These
are our most advanced users and are the most equipped to deal with disruptive changes.

## How to structure PRs to get expedient reviews?

We recommend that any PR (unless it is trivial) to be smaller than 500 lines
(excluding go mod/sum changes) in order to help reviewers to do a thorough and
reasonably fast reviews.

### When adding a new component

Components refer to connectors, exporters, extensions, processors, and receivers. The key criteria for implementing a component is to:

* Implement the `component.Component` interface
* Provide a configuration structure which defines the configuration of the component
* Provide the implementation that performs the component operation

For more details on components, see the [Adding New Components](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#adding-new-components) document and the tutorial [Building a Trace Receiver](https://opentelemetry.io/docs/collector/trace-receiver/) which provides a detailed example of building a component.

When adding a new component to the OpenTelemetry Collector, ensure that any configuration structs used by the component include fields with the `configopaque.String` type for sensitive data. This ensures that the data is masked when serialized to prevent accidental exposure.

When submitting a component to the community, consider breaking it down into separate PRs as follows:

* **First PR** should include the overall structure of the new component:
  * Readme, configuration, and factory implementation should usually use the helper
    factory structs.
  * This PR is usually trivial to review, so the size limit does not apply to
    it.
  * The component should use [`In Development` Stability](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development) in its README.
* **Second PR** should include the concrete implementation of the component. If the
  size of this PR is larger than the recommended size consider splitting it into
  multiple PRs.
* **Last PR** should mark the new component as `Alpha` stability and add it to the `otelcorecol`
  binary by updating the `otelcorecol/components.go` file. The component must be enabled
  only after sufficient testing and only when it meets [`Alpha` stability requirements.](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha)
* Once a new component has been added to the executable, please add the component
  to the [OpenTelemetry.io registry](https://github.com/open-telemetry/opentelemetry.io#adding-a-project-to-the-opentelemetry-registry).
* intra-repository `replace` statements in `go.mod` files can be automatically inserted by running `make crosslink`. For more information
  on the `crosslink` tool see the README [here](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/crosslink).

### Refactoring Work

Any refactoring work must be split in its own PR that does not include any
behavior changes. It is important to do this to avoid hidden changes in large
and trivial refactoring PRs.

## Report a bug or request a feature

Reporting bugs is an important contribution. Please make sure to include:

* Expected and actual behavior
* The OpenTelemetry version you are running
* If possible, steps to reproduce

### Adding Labels via Comments

In order to facilitate proper label usage and to empower Code Owners, you are able to add labels to issues via comments. To add a label through a comment, post a new comment on an issue starting with `/label`, followed by a space-separated list of your desired labels. Supported labels come from the table below, or correspond to a component defined in the [CODEOWNERS file](.github/CODEOWNERS).

The following general labels are supported:

| Label                    | Label in Comment         |
|--------------------------|--------------------------|
| `arm64`                  | `arm64`                  |
| `good first issue`       | `good-first-issue`       |
| `help wanted`            | `help-wanted`            |
| `discussion needed`      | `discussion-needed`      |
| `os:macos`               | `os:macos`               |
| `os:windows`             | `os:windows`             |
| `waiting for author`     | `waiting-for-author`     |
| `waiting-for-codeowners` | `waiting-for-codeowners` |
| `bug`                    | `bug`                    |
| `priority:p0`            | `priority:p0`            |
| `priority:p1`            | `priority:p1`            |
| `priority:p2`            | `priority:p2`            |
| `priority:p3`            | `priority:p3`            |
| `Stale`                  | `stale`                  |

To delete a label, prepend the label with `-`. Note that you must make a new comment to modify labels; you cannot edit an existing comment.

Example label comment:

```
/label help-wanted -arm64
```

## How to contribute

### Before you start

Please read the project contribution
[guide](https://github.com/open-telemetry/community/tree/main/guides/contributor)
for general practices for the OpenTelemetry project.

Select a good issue from the links below (ordered by difficulty/complexity):

* [Good First Issue](https://github.com/open-telemetry/opentelemetry-collector/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
* [Help Wanted](https://github.com/open-telemetry/opentelemetry-collector/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)

Comment on the issue that you want to work on so we can assign it to you and
clarify anything related to it.

If you would like to work on something that is not listed as an issue
(e.g. a new feature or enhancement) please first read our [vision](docs/vision.md) 
to make sure your proposal aligns with the goals of the
Collector, then create an issue and describe your proposal. It is best to do this
in advance so that maintainers can decide if the proposal is a good fit for
this repository. This will help avoid situations when you spend significant time
on something that maintainers may decide this repo is not the right place for.

If you're new to the Collector, the [internal architecture](docs/internal-architecture.md) documentation may be helpful.

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
workflow](https://github.com/open-telemetry/community/blob/main/guides/contributor/processes.md#github-workflow)
section of the general project contributing guide.

## Required Tools

Working with the project sources requires the following tools:

1. [git](https://git-scm.com/)
2. [go](https://golang.org/) (version 1.24 and up)
3. [make](https://www.gnu.org/software/make/)
4. [docker](https://www.docker.com/)

## Repository Setup

Fork the repo and checkout the upstream repo to your GOPATH by:

```
$ git clone git@github.com:open-telemetry/opentelemetry-collector.git
```

Add your fork as an origin:

```shell
$ cd opentelemetry-collector
$ git remote add fork git@github.com:YOUR_GITHUB_USERNAME/opentelemetry-collector.git
```

Run tests, fmt, and lint:

```shell
$ make
```

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

### Commit Messages

Use descriptive commit messages. Here are [some recommendations](https://cbea.ms/git-commit/)
on how to write good commit messages.
When creating PRs GitHub will automatically copy commit messages into the PR description,
so it is a useful habit to write good commit messages before the PR is created.
Also, unless you actually want to tell a story with multiple commits make sure to squash
into a single commit before creating the PR.

When maintainers merge PRs with multiple commits, they will be squashed and GitHub will
concatenate all commit messages right before you hit the "Confirm squash and merge"
button. Maintainers must make sure to edit this concatenated message to make it right before merging.
In some cases, if the commit messages are lacking the easiest approach to have at
least something useful is copy/pasting the PR description into the commit message box
before merging (but see the above paragraph about writing good commit messages in the first place).

## General Notes

This project uses Go 1.24.* and [Github Actions.](https://github.com/features/actions)

It is recommended to run `make gofmt all` before submitting your PR.

## Coding Guidelines

See the [Coding Guidelines](docs/coding-guidelines.md) document for more information.

## Changelog

### Overview

There are two Changelogs for this repository:

- `CHANGELOG.md` is intended for users of the collector and lists changes that affect the behavior of the collector.
- `CHANGELOG-API.md` is intended for developers who are importing packages from the collector codebase.

### When to add a Changelog Entry

An entry into the changelog is required for the following reasons:

- Changes made to the behaviour of the component
- Changes to the configuration
- Changes to default settings
- New components being added
- Changes to exported elements of a package

It is reasonable to omit an entry to the changelog under these circumstances:

- Updating test to remove flakiness or improve coverage
- Updates to the CI/CD process
- Updates to internal packages

If there is some uncertainty with regards to if a changelog entry is needed, the recommendation is to create
an entry to in the event that the change is important to the project consumers.

### Adding a Changelog Entry

The [CHANGELOG.md](./CHANGELOG.md) and [CHANGELOG-API.md](./CHANGELOG-API.md) files in this repo is autogenerated from `.yaml` files in the `./.chloggen` directory.

Your pull request should add a new `.yaml` file to this directory. The name of your file must be unique since the last release.

During the collector release process, all `./chloggen/*.yaml` files are transcribed into `CHANGELOG.md` and `CHANGELOG-API.md` and then deleted.

**Recommended Steps**
1. Create an entry file using `make chlog-new`. This generates a file based on your current branch (e.g. `./.chloggen/my-branch.yaml`)
2. Fill in all fields in the new file
3. Run `make chlog-validate` to ensure the new file is valid
4. Commit and push the file

Alternatively, copy `./.chloggen/TEMPLATE.yaml`, or just create your file from scratch.

## Local Testing

To manually test your changes, follow these steps to build and run the Collector
locally. Ensure that you execute these commands from the root of the repository:

1. Build the Collector:

  ```shell
  make otelcorecol
  ```

2. Run the Collector with a local configuration file:

  ```shell
  ./bin/otelcorecol_<os>_<arch> --config ./examples/local/otel-config.yaml
  ```

  The actual name of the binary will depend on your platform, adjust accordingly
  (e.g., `./bin/otelcorecol_darwin_arm64`).
  
  Replace `otel-config.yaml` with the appropriate configuration file as needed.

3. Verify that your changes are reflected in the Collector's behavior by testing
   it against the provided configuration.

## Membership, Roles, and Responsibilities

### Membership levels

See the [OpenTelemetry membership guide](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md) for information on how to become a member of the OpenTelemetry organization and the different roles available. In addition to the roles listed there we also have a Collector-specific role: code owners.

### Becoming a Code Owner

A Code Owner is responsible for one or multiple packages within the Collector. That responsibility includes maintaining the component, triaging and responding to issues, and reviewing pull requests.
Maintainers are expected to seek feedback from code owners for changes that are not trivial, but they may merge PRs without code owner approval.

Code Ownership does not have to be a full-time job. If you can find a couple hours to help out on a recurring basis, please consider pursuing Code Ownership.

#### Requirements

If you would like to help and become a Code Owner, you must meet the following requirements. These are more stringent requirements than those in the opentelemetry-collector-contrib repository due to the higher impact of changes in this repository:

1. [Be a member of the OpenTelemetry organization](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#member).
2. Be an existing [approver](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#approver) or [maintainer](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#maintainer) in at least one repository within the OpenTelemetry Github organization.
3. Have made significant contributions directly to the package you want to own.

#### How to become a Code Owner

To become a Code Owner, open a PR that adds you to the [.github/CODEOWNERS](.github/CODEOWNERS) file. Make sure to ping existing code owners for the package you want to own to get their approval.

## Release

See [release](docs/release.md) for details.

## Contributing Images
If you are adding any new images, please use [Excalidraw](https://excalidraw.com). It's a free and open-source web application and doesn't require any account to get started. Once you've created the design, while exporting the image, make sure to tick **"Embed scene into exported file"** option. This allows the image to be imported in an editable format for other contributors later.

## Common Issues

Build fails due to dependency issues, e.g.

```sh
go: github.com/golangci/golangci-lint@v1.31.0 requires
	github.com/tommy-muehle/go-mnd@v1.3.1-0.20200224220436-e6f9a994e8fa: invalid pseudo-version: git fetch --unshallow -f origin in /root/go/pkg/mod/cache/vcs/053b1e985f53e43f78db2b3feaeb7e40a2ae482c92734ba3982ca463d5bf19ce: exit status 128:
	fatal: git fetch-pack: expected shallow list
 ```

`go env GOPROXY` should return `https://proxy.golang.org,direct`. If it does not, set it as an environment variable:

`export GOPROXY=https://proxy.golang.org,direct`

### Makefile Guidelines

When adding or modifying the `Makefile`'s in this repository, consider the following design guidelines.

Make targets are organized according to whether they apply to the entire repository, or only to an individual module.
The [Makefile](./Makefile) SHOULD contain "repo-level" targets. (i.e. targets that apply to the entire repo.)
Likewise, `Makefile.Common` SHOULD contain "module-level" targets. (i.e. targets that apply to one module at a time.)
Each module should have a `Makefile` at its root that includes `Makefile.Common`.

#### Module-level targets

Module-level targets SHOULD NOT act on nested modules. For example, running `make lint` at the root of the repo will
_only_ evaluate code that is part of the `go.opentelemetry.io/collector` module. This excludes nested modules such as
`go.opentelemetry.io/collector/component`.

Each module-level target SHOULD have a corresponding repo-level target. For example, `make golint` will run `make lint`
in each module. In this way, the entire repository is covered. The root `Makefile` contains some "for each module" targets
that can wrap a module-level target into a repo-level target.

#### Repo-level targets

Whenever reasonable, targets SHOULD be implemented as module-level targets (and wrapped with a repo-level target).
However, there are many valid justifications for implementing a standalone repo-level target.

1. The target naturally applies to the repo as a whole. (e.g. Building the collector.)
2. Interaction between modules would be problematic.
3. A necessary tool does not provide a mechanism for scoping its application. (e.g. `porto` cannot be limited to a specific module.)
4. The "for each module" pattern would result in incomplete coverage of the codebase. (e.g. A target that scans all files, not just `.go` files.)

#### Default targets

The default module-level target (i.e. running `make` in the context of an individual module), should run a substantial set of module-level
targets for an individual module. Ideally, this would include *all* module-level targets, but exceptions should be made if a particular
target would result in unacceptable latency in the local development loop.

The default repo-level target (i.e. running `make` at the root of the repo) should meaningfully validate the entire repo. This should include
running the default common target for each module as well as additional repo-level targets.

## How to update the OTLP protocol version

When a new OTLP version is published, the following steps are required to update this code base:

1. Edit the top-level Makefile's `OPENTELEMETRY_PROTO_VERSION` variable
2. Run `make genproto` 
3. Inspect modifications to the generated code in `pdata/internal/data/protogen`
4. When new fields are added in the protocol, make corresponding changes in `pdata/internal/cmd/pdatagen/internal`
5. Run `make genpdata` 
6. Inspect modifications to the generated code in `pdata/*`
7. Run `make genproto-cleanup`, to remove temporary files
8. Update the supported OTLP version in [README.md](./README.md).

## Exceptions

While the rules in this and other documents in this repository are what we strive to follow, we acknowledge that it may be unfeasible to apply these rules in some situations. Exceptions to the rules
on this and other documents are acceptable if consensus can be obtained from approvers in the pull request they are proposed.
A reason for requesting the exception MUST be given in the pull request. Until unanimity is obtained, approvers and maintainers are
encouraged to discuss the issue at hand. If a consensus (unanimity) cannot be obtained, the maintainers' group is then tasked with making a
decision using its regular means (voting, TC help, etc.).
