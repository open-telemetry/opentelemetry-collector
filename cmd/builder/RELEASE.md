# Releasing the OpenTelemetry Collector Builder

This project uses [`goreleaser`](https://github.com/goreleaser/goreleaser) to manage the release of new versions.

To release a new version, simply add a tag named `vX.Y.Z`, like:

```
git tag -a v0.1.1 -m "Release v0.1.1"
git push upstream v0.1.1
```

A new GitHub workflow should be started, and at the end, a GitHub release should have been created, similar with the ["Release v0.56.0"](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/cmd%2Fbuilder%2Fv0.56.0).