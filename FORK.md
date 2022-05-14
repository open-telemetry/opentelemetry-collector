# How this fork works

This fork was created to be able to early patch open-telemetry/opentelemetry-collector versions in cases it is hard to get the change in upstream right away.

## How do we consume this library

For every opentelemetry-collector release we create a new release including our own patches. For example for version v0.49.0 we in open-telemetry/opentelemetry-collector we will crease v0.49.0+patches. This make sure we stick to a version in our downstream dependencies.

Whenever we need a new release on this repository we rebase the branch `latest+patches` version against the new release for open-telemetry/opentelemetry-collector and then get a new release. For example on version `v0.49.0` (asuming `origin` is `git@github.com:hypertrace/opentelemetry-collector.git` and `upstream` is `git@github.com:open-telemetry/opentelemetry-collector.git`):

```bash
git fetch --all
git checkout latest+patches
git pull --rebase upstream refs/tags/v0.49.0
# make lint test
git tag -a "v0.49.0+patches" -m "Release v0.49.0"
git push --tags
```

## What custom changes are in here
- In `config/configgrpc/configgrpc.go` we added the ability to add extra `ClientDialOptionHandler`. Also a unit for this in `config/configgrpc/configgrpcclientdialoptionhandler_test.go`.
- In `service/collector.go` we added a `ConfigPostProcessor` interface so that we can intercept the final config and do more changes in the factories or pipelines.