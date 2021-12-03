# OpenTelemetry Collector Release Procedure

Collector build and testing is currently fully automated. However there are still certain operations that need to be performed manually in order to make a release.

We release both core and contrib collectors with the same versions where the contrib release uses the core release as a dependency. We’ve divided this process into four sections. A release engineer must release:
1. The [Core](#releasing-opentelemetry-collector) collector, including the collector builder CLI tool.
1. The [Contrib](#releasing-opentelemetry-collector-contrib) collector.
1. The [artifacts](#producing-the-artifacts)

**Important Note:** You’ll need to be able to sign git commits/tags in order to be able to release a collector version. Follow [this guide](https://docs.github.com/en/github/authenticating-to-github/signing-commits) to setup it up.

**Important Note:** You’ll need to be an approver for both the repos in order to be able to make the release. This is required as you’ll need to push tags and commits directly to the upstream repo.

## Releasing opentelemetry-collector

1. Update Contrib to use the latest in development version of Core. Run `make update-otel` in Contrib root directory and if it results in any changes, submit a PR. Get the PR approved and merged. This is to ensure that the latest core does not break contrib in any way. We’ll update it once more to the final release number later. Make sure contrib builds and end-to-end tests pass successfully after being merged and -dev docker images are published.

1. Determine the version number that will be assigned to the release. Collector uses semver, with the exception that while we are still in Beta stage breaking changes are allowed without incrementing major version number. For breaking changes we increment the minor version number and set the patch number to 0.

1. Prepare Core for release.
    * Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top. <!-- markdown-link-check-disable-line --> Use commit history feature to get the list of commits since the last release to help understand what should be in the release notes, e.g.: https://github.com/open-telemetry/opentelemetry-collector/compare/${last_release}...main. Submit a PR with the changes and get the PR approved and merged.
    * Use multimod to update the version of the collector package
      * Update [versions.yaml](https://github.com/open-telemetry/opentelemetry-collector/blob/main/versions.yaml)
      * Run `make multimod-prerelease`
    * Update the collector version in the collector builder to the new release in `./cmd/builder/internal/builder/config.go`.
    * Update the collector version in the manifest used by `make run` to the new release in `./internal/buildscripts/builder-config.yaml`.

1. Make sure the current main branch build successfully passes (Core and Contrib). For Contrib also check that the spawn-stability-tests-job triggered by the main build-publish job also passes. Check that the corresponding "-dev" images exist in Dockerhub (Core and Contrib).

1. Create a branch named release/<release-series> (e.g. release/v0.4.x) in Core from the changelog update commit and push to origin (not your fork). Wait for the release branch builds to pass successfully.

1. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.40.0`). Push them upstream (not your fork) with `make push-tag TAG=v0.40.0`. NOTE: push-tag assumes your git remote is called `upstream`. Wait for the new tag build to pass successfully.

1. The release script for the collector builder should create a new GitHub release for the builder. This is a separate release from the core, but we might join them in the future if it makes sense.

1. Create a new `v0.40.0` release and use the contents from the CHANGELOG.md as the release's description.

## Releasing opentelemetry-collector-contrib

1. Prepare Contrib for release.
   * Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top. Refer to Core release notes (assuming the previous release of Core and Contrib was also performed simultaneously), and in addition to that list changes that happened in the Contrib repo.
   * Use multimod to update the version of the collector package
      * Update [versions.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/versions.yaml)
      * Run `make multimod-prerelease`

1. Update the Core dependency to the Core version we just released with `make update-otel` command, e.g, `make update-otel OTEL_VERSION=v0.4.0`. Create a PR with both the changes, get it approved and merged.

1. Create a branch named release/<release-series> (e.g. release/v0.4.x) in Contrib from the changelog update commit and push it upstream (not your fork). Wait for the release branch builds to pass successfully.

1. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.40.0`). Push them upstream (not your fork) with `make push-tag TAG=v0.40.0`. NOTE: push-tag assumes your git remote is called `upstream`. Wait for the new tag build to pass successfully.

1. Create a new Github release from the new version tag and copy release notes from the CHANGELOG.md file to the release. This step should be automated. CI can pull the release notes from the change log and use it as the body when creating the new release.

## Producing the artifacts

The last step of the release process creates artifacts for the new version of the collector and publishes images to Dockerhub. The steps in this portion of the release are done in the [opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repo.

1. Update `./distribution/otelcol/manifest.yaml` to include the new release version.

1. Update the builder version in `OTELCOL_BUILDER_VERSION` to the new release in the `Makefile`. While this might not be strictly necessary for every release, this is a good practice.

1. Create a pull request with the change and ensure the build completes successfully. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/pull/17/files).

1. Ensure the "Release" action passes, this will push new docker images to https://hub.docker.com/repository/docker/otel/opentelemetry-collector, create a Github release for the tag and push all the build artifacts to the Github release. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/actions/runs/1346637081).
