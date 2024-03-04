---
name: Release
about: Create an issue for releasing new versions
title: 'Release stable vY.Y.Y and beta vX.X.X'
labels: release
assignees: ''

---

Like #9601, but for vX.X.X

**Performed by collector release manager**

- [ ] Prepare stable core release vY.Y.Y
- [ ] Tag and release stable core vY.Y.Y
- [ ] Prepare beta core release vX.X.X
- [ ] Tag and release beta core vX.X.X
- [ ] Prepare contrib release vX.X.X
- [ ] Tag and release contrib vX.X.X
- [ ] Prepare otelcol-releases vX.X.X
- [ ] Release binaries and container images vX.X.X
- [ ] Update the release schedule in docs/RELEASE.md

**Performed by operator maintainers**

- [ ] Release the operator vX.X.X

**Performed by helm chart maintainers**

- [ ] Update the opentelemetry-collector helm chart to use vX.X.X docker image
