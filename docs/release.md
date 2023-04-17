# OpenTelemetry Collector Release Procedure

Collector build and testing is currently fully automated. However there are still certain operations that need to be performed manually in order to make a release.

We release both core and contrib collectors with the same versions where the contrib release uses the core release as a dependency. We’ve divided this process into four sections. A release engineer must release:
1. The [Core](#releasing-opentelemetry-collector) collector, including the collector builder CLI tool.
1. The [Contrib](#releasing-opentelemetry-collector-contrib) collector.
1. The [artifacts](#producing-the-artifacts)

**Important Note:** You’ll need to be able to sign git commits/tags in order to be able to release a collector version. Follow [this guide](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) to setup it up.

**Important Note:** You’ll need to be an approver for both the repos in order to be able to make the release. This is required as you’ll need to push tags and commits directly to the following repositories:

* [open-telemetry/opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-collector)
* [open-telemetry/opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)
* [open-telemetry/opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases)

## Release manager

A release manager is the person responsible for a specific release. While the manager might request help from other folks, they are ultimately responsible for the success of a release.

In order to have more people comfortable with the release process, and in order to decrease the burden on a small number of volunteers, all core approvers are release managers from time to time, listed under the [Release Schedule](#release-schedule) section. That table is updated at every release, with the current manager adding themselves to the bottom of the table, removing themselves from the top of the table.

It is possible that a core approver isn't a contrib approver. In that case, the release manager should coordinate with a contrib approver for the steps requiring such role, like the publishing of tags.

## Releasing opentelemetry-collector

1. Determine the version number that will be assigned to the release. During the beta phase, we increment the minor version number and set the patch number to 0. In this document, we are using `v0.55.0` as the version to be released, following `v0.54.0`.

2. Run the action [Automation - Prepare Release](https://github.com/open-telemetry/opentelemetry-collector/actions/workflows/prepare-release.yml). This will create an issue to track the progress of the release and a pull request to update the changelog and version numbers in the repo.

3. Update Contrib to use the latest in development version of Core. Run `make update-otel` in Contrib root directory and if it results in any changes, submit a PR. Open this PR as draft. This is to ensure that the latest core does not break contrib in any way. We’ll update it once more to the final release number later.

4. Create a branch named `release/<release-series>` (e.g. `release/v0.45.x`) from the changelog update commit and push to `open-telemetry/opentelemetry-collector`.

5. Tag all the module groups (stable, beta) with the new release version by running the `make push-tags` command (e.g. `make push-tags MODSET=stable` and `make push-tags MODSET=beta`). Wait for the new tag build to pass successfully.

6. The release script for the collector builder should create a new GitHub release for the builder. This is a separate release from the core, but we might join them in the future if it makes sense.

7. A new `v0.55.0` release should be automatically created on Github by now. Edit it and use the contents from the CHANGELOG.md as the release's description.

## Releasing opentelemetry-collector-contrib

1. Run the action [Automation - Prepare Release](https://github.com/open-telemetry/opentelemetry-collector-contrib/actions/workflows/prepare-release.yml). This will a pull request to update the changelog and version numbers in the repo.

2. Create a branch named `release/<release-series>` (e.g. `release/v0.45.x`) in Contrib from the changelog update commit and push it to `open-telemetry/opentelemetry-collector-contrib`.

3. Tag all the module groups (`contrib-base`) with the new release version by running the `make push-tags MODSET=contrib-base` command. Wait for the new tag build to pass successfully.

4. A new `v0.55.0` release should be automatically created on Github by now. Edit it and use the contents from the CHANGELOG.md as the release's description.

## Producing the artifacts

The last step of the release process creates artifacts for the new version of the collector and publishes images to Dockerhub. The steps in this portion of the release are done in the [opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repo.

1. Update the `./distribution/**/manifest.yaml` files to include the new release version.

1. Update the builder version in `OTELCOL_BUILDER_VERSION` to the new release in the `Makefile`. While this might not be strictly necessary for every release, this is a good practice.

1. Create a pull request with the change and ensure the build completes successfully. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/pull/71).

1. Tag with the new release version by running the `make push-tags TAG=v0.55.0` command. Wait for the new tag build to pass successfully.

1. Ensure the "Release" action passes, this will

    1. push new container images to https://hub.docker.com/repository/docker/otel/opentelemetry-collector

    1. create a Github release for the tag and push all the build artifacts to the Github release. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/actions/runs/1346637081).

## Troubleshooting

1. `unknown revision internal/coreinternal/v0.55.0` -- This is typically an indication that there's a dependency on a new module. You can fix it by adding a new `replaces` entry to the `go.mod` for the affected module.
2. `commitChangesToNewBranch failed: invalid merge` -- This is a [known issue](https://github.com/open-telemetry/opentelemetry-go-build-tools/issues/47) with our release tooling. The current workaround is to clone a fresh copy of the repository and try again. Note that you may need to set up a `fork` remote pointing to your own fork for the release tooling to work properly.
3. `could not run Go Mod Tidy: go mod tidy failed` when running `multimod` -- This is a [known issue](https://github.com/open-telemetry/opentelemetry-go-build-tools/issues/46) with our release tooling. The current workaround is to run `make gotidy` manually after the multimod tool fails and commit the result.

## Bugfix releases

### Bugfix release criteria

Both `opentelemetry-collector` and `opentelemetry-collector-contrib` have very short 2 week release cycles. Because of this, we put a high bar when considering making a patch release, to avoid wasting engineering time unnecessarily.

When considering making a bugfix release on the `v0.N.x` release cycle, the bug in question needs to fulfill the following criteria:

1. The bug has no workaround or the workaround is significantly harder to put in place than updating the version. Examples of simple workarounds are:
    - Reverting a feature gate.
    - Changing the configuration to an easy to find value.
2. The bug happens in common setups. To gauge this, maintainers can consider the following:
    - The bug is not specific to an uncommon platform
    - The bug happens with the default configuration or with a commonly used one (e.g. has been reported by multiple people)
3. The bug is sufficiently severe. For example (non-exhaustive list):
    - The bug makes the Collector crash reliably
    - The bug makes the Collector fail to start under an accepted configuration
    - The bug produces significant data loss
    - The bug makes the Collector negatively affect its environment (e.g. significantly affects its host machine)

We aim to provide a release that fixes security-related issues in at most 30 days since they are publicly announced; with the current release schedule this means security issues will typically not warrant a bugfix release. An exception is critical vulnerabilities (CVSSv3 score >= 9.0), which will warrant a release within five business days.

The OpenTelemetry Collector maintainers will ultimately have the responsibility to assess if a given bug or security issue fulfills all the necessary criteria and may grant exceptions in a case-by-case basis.

### Bugfix release procedure

The following documents the procedure to release a bugfix

1. Create a pull request against the `release/<release-series>` (e.g. `release/v0.45.x`) branch to apply the fix.
2. Create a pull request to update version number against the `release/<release-series>` branch.
3. Once those changes have been merged, create a pull request to the `main` branch from the `release/<release-series>` branch.
4. Enable the **Merge pull request** setting in the repository's **Settings** tab.
5. Tag all the modules with the new release version by running the `make add-tag` command (e.g. `make add-tag TAG=v0.55.0`). Push them to `open-telemetry/opentelemetry-collector` with `make push-tag TAG=v0.55.0`. Wait for the new tag build to pass successfully.
6. **IMPORTANT**: The pull request to bring the changes from the release branch *MUST* be merged using the **Merge pull request** method, and *NOT* squashed using “**Squash and merge**”. This is important as it allows us to ensure the commit SHA from the release branch is also on the main branch. **Not following this step will cause much go dependency sadness.**
7. Once the branch has been merged, it will be auto-deleted. Restore the release branch via GitHub.
8. Once the patch is release, disable the **Merge pull request** setting.

## Release schedule

| Date       | Version | Release manager |
|------------|---------|-----------------|
| 2023-04-24 | v0.76.0 | @djaglowski     |
| 2023-05-08 | v0.77.0 | @dmitryax       |
| 2023-05-22 | v0.78.0 | @bogdandrutu    |
| 2023-06-05 | v0.79.0 | @codeboten      |
| 2023-06-19 | v0.80.0 | @Aneurysm9      |
| 2023-07-03 | v0.81.0 | @jpkrohling     |
| 2023-07-17 | v0.82.0 | @mx-psi         |
