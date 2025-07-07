# OpenTelemetry Collector Release Procedure

Collector build and testing is currently fully automated. However there are still certain operations that need to be performed manually in order to make a release.

We release both core and contrib collectors with the same versions where the contrib release uses the core release as a dependency. We’ve divided this process into four sections. A release engineer must release:

1. The [Core](#releasing-opentelemetry-collector) collector, including the collector builder CLI tool.
2. The [Contrib](#releasing-opentelemetry-collector-contrib) collector.
3. The [artifacts](#producing-the-artifacts)

**Important Note:** You’ll need to be able to sign git commits/tags in order to be able to release a collector version. Follow [this guide](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) to set it up.

## Release manager

A release manager is the person responsible for a specific release. While the manager might request help from other folks, they are ultimately responsible for the success of a release.

In order to have more people comfortable with the release process, and in order to decrease the burden on a small number of volunteers, all core approvers are release managers from time to time, listed under the [Release Schedule](#release-schedule) section. That table is updated at every release, with the current manager adding themselves to the bottom of the table, removing themselves from the top of the table.

It is possible that a core approver isn't a contrib approver. In that case, the release manager should coordinate with a contrib approver for the steps requiring this role, such as branch creation or tag publishing.

To ensure the rest of the community is informed about the release and can properly help the release manager, the release manager should open a thread on the #otel-collector-dev CNCF Slack channel and provide updates there.
The thread should be shared with all Collector leads (core and contrib approvers and maintainers).

Before the release, make sure there are no open release blockers in [core](https://github.com/open-telemetry/opentelemetry-collector/labels/release%3Ablocker), [contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/labels/release%3Ablocker) and [releases](https://github.com/open-telemetry/opentelemetry-collector-releases/labels/release%3Ablocker) repos.

## Releasing opentelemetry-collector

1. Update Contrib to use the latest in development version of Core by running [Update contrib to the latest core source](https://github.com/open-telemetry/opentelemetry-collector-contrib/actions/workflows/update-otel.yaml). This is to ensure that the latest core does not break contrib in any way. If the job is failing for any reason, you can do it locally by running `make update-otel` in Contrib root directory and pushing a PR. If you are unable to run `make update-otel`, it is possible to skip this step and resolve conflicts with Contrib after Core is released, but this is generally inadvisable.
   - While this PR is open, all merging in Core is automatically halted via the `Merge freeze / Check` CI check.
   -  🛑 **Do not move forward until this PR is merged.**

2. Determine the version number that will be assigned to the release. Usually, we increment the minor version number and set the patch number to 0. In this document, we are using `v0.85.0` as the version to be released, following `v0.84.0`.
   Check if stable modules have any changes since the last release by running the following:
   - `make check-changes PREVIOUS_VERSION=v1.x.x MODSET=stable`.

   If there are no changes, there is no need to release new version for stable
   modules. If there are changes found but .chloggen directory doesn't have any
   corresponding entries, add missing changelog entries. If the changes are
   insignificant, consider not releasing a new version for stable modules.

3. Manually run the action [Automation - Prepare Release](https://github.com/open-telemetry/opentelemetry-collector/actions/workflows/prepare-release.yml). This action will create an issue to track the progress of the release and a pull request to update the changelog and version numbers in the repo.
   - When prompted, enter the version numbers determined in Step 2, but do not include a leading `v`.
   - If not intending to release stable modules, do not specify a version for `Release candidate version stable`.
   - While this PR is open all merging in Core is automatically halted via the `Merge freeze / Check` CI check.
   - If the PR needs updated in any way you can make the changes in a fork and PR those changes into the `prepare-release-prs/x` branch. You do not need to wait for the CI to pass in this prep-to-prep PR.
   -  🛑 **Do not move forward until this PR is merged.** 🛑

4. On your local machine, make sure you are on the `main` branch and that the PR from step 3 is incorporated **at the head of your branch** (this is required to ensure the proper commit is used for the release tags and branch creation below). Tag the module groups with the new release version by running:

   ⚠️ If you set your remote using `https` you need to include `REMOTE=https://github.com/open-telemetry/opentelemetry-collector.git` in each command. ⚠️

   - `make push-tags MODSET=beta` for the beta modules group,
   - `make push-tags MODSET=stable` for the stable modules group, only if there were changes since the last release.
   
   **Note**: Pushing the **beta** tags will automatically trigger the [Automation - Release Branch](https://github.com/open-telemetry/opentelemetry-collector/actions/workflows/release-branch.yml) GitHub Action, which will create the release branch (e.g. `release/v0.127.x`) from the commit that prepared the release. Pushing stable tags, if required, will not trigger creation of an additional release branch.

5. Wait for the "Automation - Release Branch" workflow to complete successfully. This workflow will automatically:
   - Detect the version from the pushed beta tags
   - Use the commit on which the tags were pushed as the "prepare release" commit
   - Create a new release branch (e.g. `release/v0.127.x`) from that commit
   
   If the workflow fails, you can check the [Actions tab](https://github.com/open-telemetry/opentelemetry-collector/actions) for details. The underlying script (./.github/workflows/scripts/release-branch.sh) can also be tested and run locally if needed by setting the GITHUB_REF environment variable (e.g., `GITHUB_REF=refs/tags/v0.85.0 ./.github/workflows/scripts/release-branch.sh`).
   
6. Wait for the tag-triggered build workflows to pass successfully.

7. A new `v0.85.0` source code release should be automatically created on Github by now. Its description should already contain the corresponding CHANGELOG.md and CHANGELOG-API.md contents.

## Releasing opentelemetry-collector-contrib

1. Manually run the action [Automation - Prepare Release](https://github.com/open-telemetry/opentelemetry-collector-contrib/actions/workflows/prepare-release.yml). When prompted, enter the version numbers of the current and new beta versions, but do not include a leading `v`. This action will create a pull request to update the changelog and version numbers in the repo. **While this PR is open all merging in Contrib should be halted**.
   - If the PR needs updated in any way you can make the changes in a fork and PR those changes into the `prepare-release-prs/x` branch. You do not need to wait for the CI to pass in this prep-to-prep PR.
   -  🛑 **Do not move forward until this PR is merged.** 🛑

2. Check out main and ensure it has the "Prepare release" commit in your local
   copy by pulling in the latest from
   `open-telemetry/opentelemetry-collector-contrib`. Use this commit to create a
   branch named `release/<release-series>` (e.g. `release/v0.85.x`). Push the
   new branch to `open-telemetry/opentelemetry-collector-contrib`. Assuming your
   upstream remote is named `upstream`, you can try the following commands:
   - `git checkout main && git fetch upstream && git rebase upstream/main`
   - `git switch -c release/<release series>` # append the commit hash of the PR in the last step if it is not the head of mainline
   - `git push -u upstream release/<release series>`

3. Make sure you are on `release/<release-series>`. Tag all the module groups with the new release version by running:

   ⚠️ If you set your remote using `https` you need to include `REMOTE=https://github.com/open-telemetry/opentelemetry-collector-contrib.git` in each command. ⚠️

   - `make push-tags MODSET=contrib-base`

4. Wait for the new tag build to pass successfully. A new `v0.85.0` release should be automatically created on Github by now, with the description containing the changelog for the new release.

5. Manually edit the release description, and add a section listing the unmaintained components at the top ([example](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.114.0)). The list of unmaintained components can be found by [searching for issues with the "unmaintained" label](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aissue%20state%3Aopen%20label%3Aunmaintained).

## Producing the artifacts

The last step of the release process creates artifacts for the new version of the collector and publishes images to Dockerhub. The steps in this portion of the release are done in the [opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repo.

1. Run the GitHub Action workflow "[Update Version in Distributions and Prepare PR](https://github.com/open-telemetry/opentelemetry-collector-releases/actions/workflows/update-version.yaml)" which will update the minor version automatically (e.g. v0.116.0 -> v0.117.0) or manually provide a new version if releasing a bugfix or skipping a version. Select "create pr" option.
   -  🛑 **Do not move forward until this PR is merged.** 🛑

2. Check out main and ensure it has the "Update version from ..." commit in your local
   copy by pulling in the latest from
   `open-telemetry/opentelemetry-collector-releases`. Assuming your upstream
   remote is named `upstream`, you can try running:
   - `git checkout main && git fetch upstream && git rebase upstream/main`

3. Create a tag for the new release version by running:
   
   ⚠️ If you set your remote using `https` you need to include `REMOTE=https://github.com/open-telemetry/opentelemetry-collector-releases.git` in each command. ⚠️
   
   - `make push-tags TAG=v0.85.0`

4. Wait for the new tag build to pass successfully.

5. Ensure the "Release Core", "Release Contrib", "Release k8s", and "Builder - Release" actions pass, this will

    1. push new container images to `https://hub.docker.com/repository/docker/otel/opentelemetry-collector`, `https://hub.docker.com/repository/docker/otel/opentelemetry-collector-contrib` and `https://hub.docker.com/repository/docker/otel/opentelemetry-collector-k8s`

    2. create a Github release for the tag and push all the build artifacts to the Github release. See [example](https://github.com/open-telemetry/opentelemetry-collector-releases/actions/workflows/release-core.yaml).

    3. build and release ocb binaries under a separate tagged Github release, e.g. `cmd/builder/v0.85.0`

    4. build and push ocb Docker images to `https://hub.docker.com/r/otel/opentelemetry-collector-builder` and the GitHub Container Registry within the releases repository

6. Update the release notes with the CHANGELOG.md updates.

## Post-release steps

After the release is complete, the release manager should do the following steps:

1. Create an issue or update existing issues for each problem encountered throughout the release in
the appropriate repositories and label them with the `release:retro` label. The release manager
should share the list of issues that affected the release with the Collector leads.
2. Update the [release schedule](#release-schedule) section of this document to remove the completed
releases and add new schedules to the bottom of the list.

## Troubleshooting

1. `unknown revision internal/coreinternal/v0.85.0` -- This is typically an indication that there's a dependency on a new module. You can fix it by adding a new `replaces` entry to the `go.mod` for the affected module.
2. `unable to tag modules: unable to load repo config: branch config: invalid merge` when running `make push-tags` -- This is a [known issue](https://github.com/open-telemetry/opentelemetry-go-build-tools/issues/47) with our release tooling, caused by a bug in `go-git`.

    It occurs if you have branches in your local repository whose entry in `.git/config` has a `merge` attribute not starting with `refs/heads`. This can typically happen when checking out PR branches using the Github CLI tool, for which the `merge` attribute starts with `refs/pull`.

    A possible workaround is to:
    - Comment out the problematic lines with `sed -E -i.bak 's/(merge = refs\/pull)/#\1/' .git/config`;
    - Try `make push-tags` again;
    - Restore the config with `mv .git/config.bak .git/config`.

    If that doesn't work, you can clone a fresh copy of the repository and try again. Note that you may need to set up a `fork` remote pointing to your own fork for the release tooling to work properly.
3. `could not run Go Mod Tidy: go mod tidy failed` when running `multimod` -- This is a [known issue](https://github.com/open-telemetry/opentelemetry-go-build-tools/issues/46) with our release tooling. The current workaround is to run `make gotidy` manually after the multimod tool fails and commit the result.
4. `Incorrect version "X" of "go.opentelemetry.io/collector/component" is included in "X"` in CI after `make update-otel` -- It could be because the make target was run too soon after updating Core and the goproxy hasn't updated yet.  Try running `export GOPROXY=direct` and then `make update-otel`.
5. `error: failed to push some refs to 'https://github.com/open-telemetry/opentelemetry-collector-contrib.git'` during `make push-tags` -- If you encounter this error the `make push-tags` target will terminate without pushing all the tags. Using the output of the `make push-tags` target, save all the un-pushed the tags in `tags.txt` and then use this make target to complete the push:

   ```bash
   .PHONY: temp-push-tags
   temp-push-tags:
       for tag in `cat tags.txt`; do \
           echo "pushing tag $${tag}"; \
           git push ${REMOTE} $${tag}; \
       done;
   ```

6. `unable to tag modules: git tag failed for v0.112.0: unable to create tag:
   "error: gpg failed to sign the data:`. Make sure you have GPG set up to sign
   commits. You can run `gpg --gen-key` to generate a GPG key.

7. When using a new GitHub Actions workflow in opentelemetry-collector-releases
   for the first time during a release, a workflow may fail. If it is possible
   to fix the workflow, you can update the release tag to the commit with the
   fix and re-run the release; it is safe to re-run the workflows that already
   succeeded. Publishing container images can be done multiple times, and
   publishing artifacts to GitHub will fail without any adverse effects.

## Bugfix releases

### Bugfix release criteria

Both `opentelemetry-collector` and `opentelemetry-collector-contrib` have very short 2 week release cycles. Because of this, we put a high bar when considering making a patch release, to avoid wasting engineering time unnecessarily.

When considering making a bugfix release on the `v0.N.x` release cycle, the bug in question needs to fulfill the following criteria:

1. The bug has no workaround or the workaround is significantly harder to put in place than updating the version. Examples of simple workarounds are:
    - Reverting a feature gate.
    - Changing the configuration to an easy to find value.
2. The bug happens in common setups. To gauge this, maintainers can consider the following:
    - If the bug is specific to a certain platform, and if that platform is in [Tier 1](../docs/platform-support.md#tiered-platform-support-model).
    - The bug happens with the default configuration or with one that is known to be used in production.
3. The bug is sufficiently severe. For example (non-exhaustive list):
    - The bug makes the Collector crash reliably
    - The bug makes the Collector fail to start under an accepted configuration
    - The bug produces significant data loss
    - The bug makes the Collector negatively affect its environment (e.g. significantly affects its host machine)
    - The bug makes it difficult to troubleshoot or debug Collector setups

We aim to provide a release that fixes security-related issues in at most 30 days since they are publicly announced; with the current release schedule this means security issues will typically not warrant a bugfix release. An exception is critical vulnerabilities (CVSSv3 score >= 9.0), which will warrant a release within five business days.

The OpenTelemetry Collector maintainers will ultimately have the responsibility to assess if a given bug or security issue fulfills all the necessary criteria and may grant exceptions in a case-by-case basis.
If the maintainers are unable to reach consensus within one working day, we will lean towards releasing a bugfix version.

### Bugfix release procedure

The release manager of a minor version is responsible for releasing any bugfix versions on this release series. The following documents the procedure to release a bugfix

1. Create a pull request against the `release/<release-series>` (e.g. `release/v0.90.x`) branch to apply the fix.
2. Make sure you are on `release/<release-series>`. Prepare release commits with `prepare-release` make target, e.g. `make prepare-release PREVIOUS_VERSION=0.90.0 RELEASE_CANDIDATE=0.90.1 MODSET=beta`, and create a pull request against the `release/<release-series>` branch.
3. Once those changes have been merged, create a pull request to the `main` branch from the `release/<release-series>` branch.
4. If you see merge conflicts when creating the pull request, do the following:
    1. Create a new branch from `origin:main`.
    2. Merge the `release/<release-series>` branch into the new branch.
    3. Resolve the conflicts.
    4. Create another pull request to the `main` branch from the new branch to replace the pull request from the `release/<release-series>` branch.
5. Disable the merge queue. An admin of the repo needs to be available for this.
6. Enable the **Merge pull request** setting in the repository's **Settings** tab.
7. Make sure you are on `release/<release-series>`. Push the new release version tags for a target module set by running `make push-tags MODSET=<beta|stable>`. Wait for the new tag build to pass successfully.
8. **IMPORTANT**: The pull request to bring the changes from the release branch *MUST* be merged using the **Merge pull request** method, and *NOT* squashed using “**Squash and merge**”. This is important as it allows us to ensure the commit SHA from the release branch is also on the main branch. **Not following this step will cause much go dependency sadness.**
9. If the pull request was created from the `release/<release-series>` branch, it will be auto-deleted. Restore the release branch via GitHub.
10. Once the patch is released, disable the **Merge pull request** setting and re-enable the merge queue.

## 1.0 release

Stable modules adhere to our [versioning document guarantees](../VERSIONING.md), so we need to be careful before releasing. Before adding a module to the stable module set and making a first 1.x release, please [open a new stabilization issue](https://github.com/open-telemetry/opentelemetry-collector/issues/new/choose) and follow the instructions in the issue template.

Once a module is ready to be released under the `1.x` version scheme, file a PR to move the module to the `stable` module set and remove it from the `beta` module set. Note that we do not make `v1.x.y-rc.z` style releases for new stable modules; we instead treat the last two beta minor releases as release candidates and the module moves directly from the `0.x` to the `1.x` release series.

## Release schedule

| Date       | Version  | Release manager                                      |
|------------|----------|------------------------------------------------------|
| 2025-07-14 | v0.130.0 | [@jmacd](https://github.com/jmacd)                   |
| 2025-07-28 | v0.131.0 | [@mx-psi](https://github.com/mx-psi)                 |
| 2025-08-11 | v0.132.0 | [@evan-bradley](https://github.com/evan-bradley)     |
| 2025-08-25 | v0.133.0 | [@TylerHelmuth](https://github.com/TylerHelmuth)     |
| 2025-09-08 | v0.134.0 | [@atoulme](https://github.com/atoulme)               |
| 2025-09-22 | v0.135.0 | [@songy23](https://github.com/songy23)               |
| 2025-10-06 | v0.136.0 | [@dmitryax](https://github.com/dmitryax)             |
| 2025-10-20 | v0.137.0 | [@codeboten](https://github.com/codeboten)           |
| 2025-11-03 | v0.138.0 | [@bogdandrutu](https://github.com/bogdandrutu)       |
| 2025-11-17 | v0.139.0 | [@jade-guiton-dd](https://github.com/jade-guiton-dd) |
| 2025-12-01 | v0.140.0 | [@dmathieu](https://github.com/dmathieu)             |
