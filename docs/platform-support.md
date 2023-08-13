# Platform Support

The OpenTelemetry Collector will be supported following a tiered platform support model to balance between the aim to support as many platforms as possible and to guarantee stability for the most important platforms. The platform support for the OpenTelemetry Collector is broken into three tiers with different levels of support for each tier  and aligns with the current test strategy.

## Current Test Strategy

The current verification process of the OpenTelemetry Collector includes unit and performance tests for core and additional end-to-end and integration tests for contrib. In the end-to-end tests, receivers, processors, and exporters etc. are tested in a testbed, while the integration tests rely on actual instances and available container images. Additional stability tests are in preparation for the future as well. All verification tests are run on Ubuntu 22.04 Linux on amd64 as the primary platform today. In addition, unit tests are run for the _contrib_ collector on Microsoft Windows Server 2022 (amd64). The cross compile supports two MacOS/Darwin targets (amd64 and arm64), five Linux platforms (amd64, arm64, i386, arm and ppc64le) and two Windows binaries (amd64, i386) today. None of those platforms is tested today, except of Linux on amd64 as the primary platform. The OpenTelemetry Collector can be installed using apk, deb or rpm files and is available as container images (core, contrib) for deployment on Kubernetes and Docker. The container images are built using qemu and published to Docker Hub and ghcr.io for Linux on amd64, arm64, i386, arm/v7 and ppc64le. The end-to-end test for the _contrib_ container images is run on Ubuntu 22.04 Linux for the Kubernetes versions v1.23 to v1.26.

## Tiered platform support model

The OpenTelemetry Collector will be supported following a tiered platform support model to balance between the aim to support as many platforms as possible and to guarantee stability for the most important platforms. The platform support for the OpenTelemetry Collector is broken into three tiers with different levels of support for each tier. 

### Tier 1 – Primary Support

The Tier 1 supported platforms are _guaranteed to work_. Precompiled binaries are built on the platform, fully supported for all collector add-ons (receivers, processor, exporters etc.), and continuously tested as part of the development processes to ensure any proposed change will function correctly. Build and test infrastructure is provided by the project. All tests are executed on the platform as part of automated continuous integration (CI) for each pull request and the biweekly release cycle. Any build or test failure block the release of the collector distribution for all platforms. Defects are addressed with priority and depending on severity fixed for the previous release in a bug fix release.

Tier 1 platforms are currently:
- Linux amd64
- Kubernetes amd64

### Tier 2 – Secondary Support

Tier 2 platforms are _guaranteed to work with specified limitations_. Precompiled binaries are built and tested on the platform as part of the biweekly release cycle. Build and test infrastructure is provided by the platform maintainers. All tests are executed on the platform as far as they are applicable, and all prerequisites are fulfilled. Not executed tests and not tested collector add-ons (received, processors, exporters, etc.) are published on release of the collector distribution. Any build or test failure delays the release of the binaries for the respective platform but not the collector distribution for all other platforms. Defects are addressed but not with the priority as for Tier 1 and, if specific to the platform, require the support of the platform maintainers.

Tier 2 platforms are currently:
- None

### Tier 3 - Community Support

Tier 3 platforms are _guaranteed to build_. Precompiled binaries are made available as part of the release process and as result of a cross compile build on Linux amd64 but the binaries are not tested at all. Any build failure delays the release of the binaries for the respective platform but not the collector distribution for all other platforms. Defects are addressed based on community contributions. Core developers might provide guidance or code reviews, but direct fixes may be limited.

Tier 3 platforms are currently:
- MacOS/Darwin amd64 
- MacOS/Darwin arm64
- Linux arm64 
- Linux i386
- Linux arm/7 
- Linux ppc64le 
- Windows amd64
- Windows i386
- Kubernetes arm64, i386, arm/v7 and ppc64le
- Docker amd64, arm64, i386, arm/v7 and ppc64le

The proposed additional platforms Linux on s390x (#378) and AIX on ppc64 ([#19195](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/19195#issuecomment-1458560971)) will be included into Tier 3 once they're added to the OpenTelemetry Collector as platforms. 
