# OpenTelemetry Collector Release Procedure

Collector build and testing is currently fully automated. However there are still certain operations that need to be performed manually in order to make a release.

We release both core and contrib collectors with the same versions where the contrib release uses the core release as a dependency. We’ve divided this process into two sections. A release engineer must first release the Core collector and then the Contrib collector.

Important: Note that you’ll need to be able to sign git commits/tags in order to be able to release a collector version. Follow [this guide](https://docs.github.com/en/github/authenticating-to-github/signing-commits) to setup it up.

Note: You’ll need to be an approver for both the repos in order to be able to make the release. This is required as you’ll need to push tags and commits directly to the upstream repo.

## Releasing OpenTelemetry Core

1. Update Contrib to use the latest in development version of Core. Run `make update-otel` in Contrib root directory and if it results in any changes, submit a PR. Get the PR approved and merged. This is to ensure that the latest core does not break contrib in any way. We’ll update it once more to the final release number later. Make sure contrib builds and end-to-end tests pass successfully after being merged and -dev docker images are published.

1. Determine the version number that will be assigned to the release. Collector uses semver, with the exception that while we are still in Beta stage breaking changes are allowed without incrementing major version number. For breaking changes we increment the minor version number and set the patch number to 0.

1. Prepare Core for release. Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top.

    <!-- markdown-link-check-disable-line --> Use commit history feature to get the list of commits since the last release to help understand what should be in the release notes, e.g.: https://github.com/open-telemetry/opentelemetry-collector-contrib/compare/${last_release}...main. Submit a PR with the changes and get the PR approved and merged.

1. Make sure the current main branch build successfully passes (Core and Contrib). For Contrib also check that the spawn-stability-tests-job triggered by the main build-publish job also passes. Check that the corresponding "-dev" images exist in Dockerhub (Core and Contrib).

1. Create a branch named release/<release-series> (e.g. release/v0.4.x) in Core from the changelog update commit and push to origin (not your fork). Wait for the release branch builds to pass successfully.

1. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.4.0`). Push them to origin (not your fork) with `git push --tags origin` (assuming origin refers to upstream open-telemetry project). Wait for the new tag build to pass successfully. This build will push new docker images to https://hub.docker.com/repository/docker/otel/opentelemetry-collector, create a Github release for the tag and push all the build artifacts to the Github release.

1. Edit the newly auto-created Github release and copy release notes from the CHANGELOG.md file to the release. This step should be automated. CI can pull the release notes from the change log and use it as the body when creating the new release.

## Releasing OpenTelemetry Contrib

1. Prepare Contrib for release. Update CHANGELOG.md file and rename the Unreleased section to the new release name. Add a new unreleased section at top. Refer to Core release notes (assuming the previous release of Core and Contrib was also performed simultaneously), and in addition to that list changes that happened in the Contrib repo.

1. Update the Core dependency to the Core version we just released with `make update-otel` command, e.g, `make update-otel OTEL_VERSION=v0.4.0`. Create a PR with both the changes, get it approved and merged.

1. Create a branch named release/<release-series> (e.g. release/v0.4.x) in Core from the changelog update commit and push to origin (not your fork). Wait for the release branch builds to pass successfully.

1. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.4.0`). Push them to origin (not your fork) with `git push --tags origin` (assuming origin refers to upstream open-telemetry project). Wait for the new tag build to pass successfully. This build will push new docker images to https://hub.docker.com/repository/docker/otel/opentelemetry-collector-contrib, create a Github release for the tag and push all the build artifacts to the Github release.

1. Edit the newly auto-created Github release and copy release notes from the CHANGELOG.md file to the release. This step should be automated. CI can pull the release notes from the change log and use it as the body when creating the new release.
