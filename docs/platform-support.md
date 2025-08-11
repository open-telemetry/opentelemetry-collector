# Platform Support

The OpenTelemetry Collector will be supported following a tiered platform support model to balance between the aim to support as many platforms as possible and to guarantee stability for the most important platforms. A platform is described by the pair of operating system and processor architecture family as they are defined by the Go programming language as [known operating systems and architectures for use with the GOOS and GOARCH values](https://go.dev/src/internal/syslist/syslist.go). 

For a supported platform, the OpenTelemetry Collector is supported when the [minimum requirements](https://github.com/golang/go/wiki/MinimumRequirements) of the Go release used by the collector are met for the operating system and architecture. Each supported platform requires the naming of designated owners. The platform support for the OpenTelemetry Collector is broken into three tiers with different levels of support for each tier and aligns with the current test strategy. 

For platforms not listed as supported by any of the tiers, support cannot be assumed to be provided. While the project may accept specific changes related to these platforms, there will be no official builds, support of issues and development of enhancements or bug fixes for these platforms. Future development of project for supported platforms may break the functionality of unsupported platforms.

## Current Test Strategy

The current verification process of the OpenTelemetry Collector includes unit and performance tests for core and additional end-to-end and integration tests for contrib. In the end-to-end tests, receivers, processors, and exporters etc. are tested in a testbed, while the integration tests rely on actual instances and available container images. Additional stability tests are in preparation for the future as well. All verification tests are run on the linux/amd64 as the primary platform today. In addition, unit tests are run for the _contrib_ collector on windows/amd64. The tests use as execution environments the latest Ubuntu and Windows Server versions [supported as Github runners](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-runners-and-hardware-resources). 

The cross compile supports the following targets:
- darwin/amd64 and darwin/arm64
- linux/amd64, linux/arm64, linux/386, linux/arm and linux/ppc64le
- windows/amd64, windows/386. 

Except of the mentioned tests for linux/amd64 and windows/amd64, no other platforms are tested by the CI/CD tooling. 

Container images of the _core_ and _contrib_ collector are built and published to Docker Hub and ghcr.io for the platforms specified in the [goreleaser configuration](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/bf8002ec6d2109cdb4184fc6eb6f8bda59ea96a2/.goreleaser.yaml#L137). End-to-end tests of the _contrib_ container images are run on the latest Ubuntu Linux supported by GitHub runners and for the four most recent Kubernetes versions.

## Tiered platform support model

The platform support for the OpenTelemetry Collector is broken into three tiers with different levels of support for each tier. 

### Platform Support - Summary

The following tables summarized the platform tiers of support by the verification tests performed for them and by the specification if dummy implementations are allowed for selected features, the availability of precompiled binaries incl. container images and if bugfix releases are provided for previous releases in case of critical defects. 

| Tier | Unit tests | Performance tests | End-to-end tests | Integrations tests | Dummy implementations | Precompiled binaries | Bugfix releases |
|------|------------|-------------------|------------------|--------------------|-----------------------|----------------------|-----------------|
| 1    | yes        | yes               | yes              | yes                | no                    | yes                  | yes             |
| 2    | yes        | optional          | optional         | optional           | yes                   | yes                  | no              |
| 3    | no         | no                | no               | no                 | yes                   | yes                  | no              |

### Tier 1 – Primary Support

The Tier 1 supported platforms are _guaranteed to work_. Precompiled binaries are built on the platform, fully supported for all collector add-ons (receivers, processor, exporters etc.), and continuously tested as part of the development processes to ensure any proposed change will function correctly. Build and test infrastructure is provided by the project. All tests are executed on the platform as part of automated continuous integration (CI) for each pull request and the [release cycle](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/release.md#release-schedule). Any build or test failure block the release of the collector distribution for all platforms. Defects are addressed with priority and depending on severity fixed for previous release(s) in a bug fix release.

Tier 1 platforms are currently:
| Platform    | Owner(s)                                                                                                    |
|-------------|-------------------------------------------------------------------------------------------------------------|
| linux/amd64 | [OpenTelemetry Collector approvers](https://github.com/open-telemetry/opentelemetry-collector#contributing) |

### Tier 2 – Secondary Support

Tier 2 platforms are _guaranteed to work with specified limitations_. Precompiled binaries are built and tested on the platform as part of the release cycle. Build and test infrastructure is provided by the platform maintainers. All tests are executed on the platform as far as they are applicable, and all prerequisites are fulfilled. Not executed tests and not tested collector add-ons (receivers, processors, exporters, etc.) are published on release of the collector distribution. Any build or test failure delays the release of the binaries for the respective platform but not the collector distribution for all other platforms. Defects are addressed but not with the priority as for Tier 1 and, if specific to the platform, require the support of the platform maintainers.

Tier 2 platforms are currently:
| Platform      | Owner(s)                                           |
|---------------|----------------------------------------------------|
| darwin/arm64  | [@MovieStoreGuy](https://github.com/MovieStoreGuy) |
| linux/arm64   | [@atoulme](https://github.com/atoulme)             |
| windows/amd64 | [@pjanotti](https://github.com/pjanotti)           |

### Tier 3 - Community Support

Tier 3 platforms are _guaranteed to build_. Precompiled binaries are made available as part of the release process and as result of a cross compile build on Linux amd64 but the binaries are not tested at all. Any build failure delays the release of the binaries for the respective platform but not the collector distribution for all other platforms. Defects are addressed based on community contributions. Core developers might provide guidance or code reviews, but direct fixes may be limited.

Tier 3 platforms are currently:
| Platform      | Owner(s)                                                                                                                                                       |
|---------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| darwin/amd64  | [@h0cheung](https://github.com/h0cheung)                                                                                                                       |
| linux/386     | [@andrzej-stencel](https://github.com/andrzej-stencel)                                                                                                         |
| linux/arm     | [@Wal8800](https://github.com/Wal8800), [@atoulme](https://github.com/atoulme)                                                                                 |
| linux/ppc64le | [@IBM-Currency-Helper](https://github.com/IBM-Currency-Helper), [@adilhusain-s](https://github.com/adilhusain-s), [@seth-priya](https://github.com/seth-priya) |
| linux/riscv64 | [@shanduur](https://github.com/shanduur)                                                                                                                       |
| linux/s390x   | [@bwalk-at-ibm](https://github.com/bwalk-at-ibm), [@rrschulze](https://github.com/rrschulze)                                                                   |
| windows/386   | [@pjanotti](https://github.com/pjanotti)                                                                                                                       |

The proposed additional platform aix/ppc64 ([#19195](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/19195)) will be included into Tier 3 once it's added to the OpenTelemetry Collector as platform. 
