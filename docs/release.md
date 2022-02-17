# OpenTelemetry Collector Release Procedure

Collector build and testing is currently fully automated. However there are still certain operations that need to be performed manually in order to make a release.

We release both core and contrib collectors with the same versions where the contrib release uses the core release as a dependency. We’ve divided this process into four sections. A release engineer must release:
1. The [Core](#releasing-opentelemetry-collector) collector, including the collector builder CLI tool.
1. The [Contrib](#releasing-opentelemetry-collector-contrib) collector.
1. The [artifacts](#producing-the-artifacts)

**Important Note:** You’ll need to be able to sign git commits/tags in order to be able to release a collector version. Follow [this guide](https://docs.github.com/en/github/authenticating-to-github/signing-commits) to setup it up.

**Important Note:** You’ll need to be an approver for both the repos in order to be able to make the release. This is required as you’ll need to push tags and commits directly to the following upstream repositories:

* [open-telemetry/opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-collector)
* [open-telemetry/opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)
* [open-telemetry/opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases)

## Releasing opentelemetry-collector

1. Make sure the current `main` branch build successfully passes (Core and Contrib).

1. Determine the version number that will be assigned to the release. During the beta phase, we increment the minor version number and set the patch number to 0. In this document, we are using `v0.45.0` as the version to be released, following `v0.44.0`.

1. To keep track of the progress, it might be helpful to create a tracking issue similar to [#4870](https://github.com/open-telemetry/opentelemetry-collector/issues/4870). You are responsible for all of them, except the operator one. A template for the tracking issue can be found at the [Appendix A](#appendix-a), and once the issue is created, you can create the individual ones by hovering them and clicking the "Convert to issue" button on the right hand side.

1. Update Contrib to use the latest in development version of Core. Run `make update-otel` in Contrib root directory and if it results in any changes, submit a PR. Open this PR as draft. This is to ensure that the latest core does not break contrib in any way. We’ll update it once more to the final release number later.

1. Prepare Core for release.

    * Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top. Use commit history feature to get the list of commits since the last release to help understand what should be in the release notes, e.g.: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.44.0...main.

    * Use multimod to update the version of the collector package
      * Update [versions.yaml](https://github.com/open-telemetry/opentelemetry-collector/blob/main/versions.yaml) and commit it
      * Run `make multimod-prerelease` (it might fail if you didn't commit the change above)

    * Update the collector version in the collector builder to the new release in `./cmd/builder/internal/builder/config.go`.

    * Update the collector version in the manifest used by `make run` to the new release in `./cmd/otelcorecol/builder-config.yaml`.

    * Submit a PR with the changes and get the PR approved and merged.

    * Ensure the `main` branch builds successfully.

1. Create a branch named `release/<release-series>` (e.g. `release/v0.45.x`) from the changelog update commit and push to `open-telemetry/opentelemetry-collector`.

1. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.45.0`). Push them to `open-telemetry/opentelemetry-collector` with `make push-tag TAG=v0.45.0`. Wait for the new tag build to pass successfully.

1. The release script for the collector builder should create a new GitHub release for the builder. This is a separate release from the core, but we might join them in the future if it makes sense.

1. Create a new `v0.45.0` release and use the contents from the CHANGELOG.md as the release's description. At the top of the release's changelog, add a link to the releases repository where the binaries and other artifacts are landing, like:

```
### Images and binaries here: https://github.com/open-telemetry/opentelemetry-collector-releases/releases/tag/v0.45.0
```

## Releasing opentelemetry-collector-contrib

1. Prepare Contrib for release.

  * Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top. Refer to Core release notes (assuming the previous release of Core and Contrib was also performed simultaneously), and in addition to that list changes that happened in the Contrib repo.

  * Use multimod to update the version of the collector package:

      * Update [versions.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/versions.yaml) and commit it

      * Run `make multimod-prerelease`

1. Update the Core dependency to the Core version we just released with `make update-otel OTEL_VERSION=v0.45.0` command. Create a PR with both the changes, get it approved and merged.

1. Create a branch named `release/<release-series>` (e.g. `release/v0.45.x`) in Contrib from the changelog update commit and push it to `open-telemetry/opentelemetry-collector-contrib`.

1. Tag all the modules with the new release version by running the `make add-tag TAG=v0.45.0` command. Push them to `open-telemetry/opentelemetry-collector-contrib` with `make push-tag TAG=v0.45.0`. Wait for the new tag build to pass successfully.

1. Create a new `v0.45.0` release and use the contents from the CHANGELOG.md as the release's description.

## Producing the artifacts

The last step of the release process creates artifacts for the new version of the collector and publishes images to Dockerhub. The steps in this portion of the release are done in the [opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repo.

1. Update the `./distribution/**/manifest.yaml` files to include the new release version.

1. Update the builder version in `OTELCOL_BUILDER_VERSION` to the new release in the `Makefile`. While this might not be strictly necessary for every release, this is a good practice.

1. Create a pull request with the change and ensure the build completes successfully. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/pull/71).

1. Create a tag for this release and push it to `open-telemetry/opentelemetry-collector-releases`:
    * `git tag -m "Release v0.45.0" v0.45.0`

    * `git push git@github.com:open-telemetry/opentelemetry-collector-releases.git v0.45.0`

1. Ensure the "Release" action passes, this will

    1. push new container images to https://hub.docker.com/repository/docker/otel/opentelemetry-collector
    
    1. create a Github release for the tag and push all the build artifacts to the Github release. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/actions/runs/1346637081).

## Troubleshooting

1. `unknown revision internal/coreinternal/v0.45.0` -- This is typically an indication that there's a dependency on a new module. You can fix it by adding a new `replaces` entry to the `go.mod` for the affected module. 

## Appendix A

```
Like #4522, but for v0.45.0

- [ ] Prepare core release v0.45.0
- [ ] Tag and release core v0.45.0
- [ ] Prepare contrib release v0.45.0
- [ ] Tag and release contrib v0.45.0
- [ ] Prepare otelcol-releases v0.45.0
- [ ] Release binaries and container images v0.45.0
- [ ] Release the operator v0.45.0
```

## Release history and schedule

| Date       | Version | Release manager |
|------------|---------|-----------------|
| 2022-01-25 | v0.43.0 | @codeboten      |
| 2022-02-03 | v0.44.0 | @bogdandrutu    |
| 2022-02-17 | v0.45.0 | @jpkrohling     |
| 2022-03-02 | v0.46.0 | @jpkrohling     |
| 2022-03-16 | v0.47.0 | undetermined    |
| 2022-03-30 | v0.48.0 | undetermined    |
| 2022-04-13 | v0.49.0 | undetermined    |
| 2022-04-27 | v0.50.0 | undetermined    |
