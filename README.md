---

<p align="center">
  <strong>
    <a href="https://opentelemetry.io/docs/collector/getting-started/">Getting Started</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="CONTRIBUTING.md">Getting Involved</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://cloud-native.slack.com/archives/C01N6P7KR6W">Getting In Touch</a>
  </strong>
</p>

<p align="center">
  <a href="https://github.com/open-telemetry/opentelemetry-collector/actions/workflows/build-and-test.yml?query=branch%3Amain">
    <img alt="Build Status" src="https://img.shields.io/github/actions/workflow/status/open-telemetry/opentelemetry-collector/build-and-test.yml?branch=main&style=for-the-badge">
  </a>
  <a href="https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-collector">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-collector?style=for-the-badge">
  </a>
  <a href="https://codecov.io/gh/open-telemetry/opentelemetry-collector/branch/main/">
    <img alt="Codecov Status" src="https://img.shields.io/codecov/c/github/open-telemetry/opentelemetry-collector?style=for-the-badge">
  </a>
  <a href="https://github.com/open-telemetry/opentelemetry-collector/releases">
    <img alt="GitHub release (latest by date including pre-releases)" src="https://img.shields.io/github/v/release/open-telemetry/opentelemetry-collector?include_prereleases&style=for-the-badge">
  </a>
  <a href="https://www.bestpractices.dev/projects/8404"><img src="https://www.bestpractices.dev/projects/8404/badge">
  </a>
  <a href="https://bugs.chromium.org/p/oss-fuzz/issues/list?sort=-opened&can=1&q=proj:opentelemetry">
    <img alt="Fuzzing Status" src="https://oss-fuzz-build-logs.storage.googleapis.com/badges/opentelemetry.svg">
  </a>
</p>

<p align="center">
  <strong>
    <a href="docs/vision.md">Vision</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://opentelemetry.io/docs/collector/configuration/">Configuration</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://opentelemetry.io/docs/collector/internal-telemetry/#use-internal-telemetry-to-monitor-the-collector</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/security-best-practices.md">Security</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://pkg.go.dev/go.opentelemetry.io/collector">Package</a>
  </strong>
</p>

---

# <img src="https://opentelemetry.io/img/logos/opentelemetry-logo-nav.png" alt="OpenTelemetry Icon" width="45" height=""> OpenTelemetry Collector

The OpenTelemetry Collector offers a vendor-agnostic implementation on how to
receive, process and export telemetry data. In addition, it removes the need
to run, operate and maintain multiple agents/collectors in order to support
open-source telemetry data formats (e.g. Jaeger, Prometheus, etc.) to
multiple open-source or commercial back-ends.

Objectives:

- Usable: Reasonable default configuration, supports popular protocols, runs and collects out of the box.
- Performant: Highly stable and performant under varying loads and configurations.
- Observable: An exemplar of an observable service.
- Extensible: Customizable without touching the core code.
- Unified: Single codebase, deployable as an agent or collector with support for traces, metrics and logs.

## Community

The OpenTelemetry Collector SIG is present at the [#otel-collector](https://cloud-native.slack.com/archives/C01N6P7KR6W)
channel on the CNCF Slack and [meets once a week](https://github.com/open-telemetry/community#implementation-sigs) via
video calls. Everyone is invited to join those calls, which typically serves the following purposes:

- meet the humans behind the project
- get an opinion about specific proposals
- look for a sponsor for a proposed component after trying already via GitHub and Slack
- get attention to a specific pull-request that got stuck and is difficult to discuss asynchronously

Between 11 July 2024 and 09 January 2025, we'll have our video calls rotating between three time slots, in order to
allow everyone to join at least once every three meetings:

- [00:00 UTC](https://dateful.com/convert/utc?t=0000)
- [12:00 UTC](https://dateful.com/convert/utc?t=1200)
- [16:00 UTC](https://dateful.com/convert/utc?t=1600)

Contributors to the project are also welcome to have ad-hoc meetings for synchronous discussions about specific points.
Post a note in #otel-collector on Slack inviting others, specifying the topic to be discussed. Unless there are strong
reasons to keep the meeting private, please make it an open invitation for other contributors to join. Try also to
identify who would be the other contributors interested on that topic and in which timezones they are.

Remember that our source of truth is GitHub: every decision made via Slack or video calls has to be recorded in the
relevant GitHub issue. Ideally, the agenda items from the meeting notes would include a link to the issue or pull
request where a discussion is happening already. We acknowledge that not everyone can join Slack or the synchronous
calls and don't want them to feel excluded.

## Supported OTLP version

This code base is currently built against using OTLP protocol v1.3.1,
considered Stable. [See the OpenTelemetry Protocol Stability
definition
here.](https://github.com/open-telemetry/opentelemetry-proto?tab=readme-ov-file#stability-definition)

## Stability levels

The collector components and implementation are in different stages of stability, and usually split between
functionality and configuration. The status for each component is available in the README file for the component. While
we intend to provide high-quality components as part of this repository, we acknowledge that not all of them are ready
for prime time. As such, each component should list its current stability level for each telemetry signal, according to
the following definitions:

### Development

Not all pieces of the component are in place yet and it might not be available as part of any distributions yet. Bugs and performance issues should be reported, but it is likely that the component owners might not give them much attention. Your feedback is still desired, especially when it comes to the user-experience (configuration options, component observability, technical implementation details, ...). Configuration options might break often depending on how things evolve. The component should not be used in production.

### Alpha

The component is ready to be used for limited non-critical workloads and the authors of this component would welcome your feedback. Bugs and performance problems should be reported, but component owners might not work on them right away. The configuration options might change often without backwards compatibility guarantees.

### Beta

Same as Alpha, but the configuration options are deemed stable. While there might be breaking changes between releases, component owners should try to minimize them. A component at this stage is expected to have had exposure to non-critical production workloads already during its **Alpha** phase, making it suitable for broader usage.

### Stable

The component is ready for general availability. Bugs and performance problems should be reported and there's an expectation that the component owners will work on them. Breaking changes, including configuration options and the component's output are not expected to happen without prior notice, unless under special circumstances.

### Deprecated

The component is planned to be removed in a future version and no further support will be provided. Note that new issues will likely not be worked on. When a component enters "deprecated" mode, it is expected to exist for at least two minor releases. See the component's readme file for more details on when a component will cease to exist.

### Unmaintained

A component identified as unmaintained does not have an active code owner. Such component may have never been assigned a code owner or a previously active code owner has not responded to requests for feedback within 6 weeks of being contacted. Issues and pull requests for unmaintained components will be labelled as such. After 6 months of being unmaintained, these components will be removed from official distribution. Components that are unmaintained are actively seeking contributors to become code owners.

## Compatibility

When used as a library, the OpenTelemetry Collector attempts to track the currently supported versions of Go, as [defined by the Go team](https://go.dev/doc/devel/release#policy).
Removing support for an unsupported Go version is not considered a breaking change.

Support for Go versions on the OpenTelemetry Collector is updated as follows:

1. The first release after the release of a new Go minor version `N` will add build and tests steps for the new Go minor version.
2. The first release after the release of a new Go minor version `N` will remove support for Go version `N-2`.

Official OpenTelemetry Collector distro binaries will be built with a release in the latest Go minor version series.

## Verifying the images signatures

> [!NOTE]
> To verify a signed artifact or blob, first [install Cosign](https://docs.sigstore.dev/cosign/system_config/installation/), then follow the instructions below.

We are signing the images `otel/opentelemetry-collector` and `otel/opentelemetry-collector-contrib` using [sigstore cosign](https://github.com/sigstore/cosign) tool and to verify the signatures you can run the following command:

```console
$ cosign verify \
  --certificate-identity=https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/base-release.yaml@refs/tags/<RELEASE_TAG> \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  <OTEL_COLLECTOR_IMAGE>
```

where:

- `<RELEASE_TAG>`: is the release that you want to validate
- `<OTEL_COLLECTOR_IMAGE>`: is the image that you want to check

Example:

```console
$ cosign verify --certificate-identity=https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/base-release.yaml@refs/tags/v0.98.0 --certificate-oidc-issuer=https://token.actions.githubusercontent.com ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.98.0

Verification for ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.98.0 --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The code-signing certificate was verified using trusted certificate authority certificates

[{"critical":{"identity":{"docker-reference":"ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib"},"image":{"docker-manifest-digest":"sha256:5cea85bcbc734a3c0a641368e5a4ea9d31b472997e9f2feca57eeb4a147fcf1a"},"type":"cosign container image signature"},"optional":{"1.3.6.1.4.1.57264.1.1":"https://token.actions.githubusercontent.com","1.3.6.1.4.1.57264.1.2":"push","1.3.6.1.4.1.57264.1.3":"9e20bf5c142e53070ccb8320a20315fffb41469e","1.3.6.1.4.1.57264.1.4":"Release Contrib","1.3.6.1.4.1.57264.1.5":"open-telemetry/opentelemetry-collector-releases","1.3.6.1.4.1.57264.1.6":"refs/tags/v0.98.0","Bundle":{"SignedEntryTimestamp":"MEUCIQDdlmNeKXQrHnonwWiHLhLLwFDVDNoOBCn2sv85J9P8mgIgDQFssWJImo1hn38VlojvSCL7Qq5FMmtnGu0oLsNdOm8=","Payload":{"body":"eyJhcGlWZXJzaW9uIjoiMC4wLjEiLCJraW5kIjoiaGFzaGVkcmVrb3JkIiwic3BlYyI6eyJkYXRhIjp7Imhhc2giOnsiYWxnb3JpdGhtIjoic2hhMjU2IiwidmFsdWUiOiIxMzVjY2RlN2YzZTNhYjU2NmFmYzJhYWU3MDljYmJlNmFhMDZlZWMzNDA2MWNkZjMyNmRhYzM2MmY0NWM4Yjg4In19LCJzaWduYXR1cmUiOnsiY29udGVudCI6Ik1FVUNJUURFbDV6N0diMWRVYkM5KzR4c1VvbDhMcWZNV2hiTzhkdEpwdExyMXhUNWZnSWdTdEwwN1I0ZDA5R2x0ZkV0azJVbmlJSlJhQVdrVDJNWDVtRXJNSlplc2pRPSIsInB1YmxpY0tleSI6eyJjb250ZW50IjoiTFMwdExTMUNSVWRKVGlCRFJWSlVTVVpKUTBGVVJTMHRMUzB0Q2sxSlNVaG9ha05EUW5jeVowRjNTVUpCWjBsVlNETkNjRFZTYlVSU1VpOXphMWg0YVdWUFlrcFhSbmRrUjNNNGQwTm5XVWxMYjFwSmVtb3dSVUYzVFhjS1RucEZWazFDVFVkQk1WVkZRMmhOVFdNeWJHNWpNMUoyWTIxVmRWcEhWakpOVWpSM1NFRlpSRlpSVVVSRmVGWjZZVmRrZW1SSE9YbGFVekZ3WW01U2JBcGpiVEZzV2tkc2FHUkhWWGRJYUdOT1RXcFJkMDVFUlhoTlJGRjRUMFJOTlZkb1kwNU5hbEYzVGtSRmVFMUVVWGxQUkUwMVYycEJRVTFHYTNkRmQxbElDa3R2V2tsNmFqQkRRVkZaU1V0dldrbDZhakJFUVZGalJGRm5RVVZyWlRsSE1ubHNjMjkzYVZZMmRFOVZSazlRVVhNd2NXY3hTSEV5WmpsVUx6UTJZbEFLU1ZSNE0ybFRkVXBhV0hGc1dEUldWV2Q1VlZndmNVazJhblZ2WlZSVEswaG5XVUoyYjBseVNERTFUeTltZEd0VmVtRlBRMEpwZDNkbloxbHZUVUUwUndwQk1WVmtSSGRGUWk5M1VVVkJkMGxJWjBSQlZFSm5UbFpJVTFWRlJFUkJTMEpuWjNKQ1owVkdRbEZqUkVGNlFXUkNaMDVXU0ZFMFJVWm5VVlZHTkRrMUNrdDFNRWhqTm5rek1rNUNTVTFFU21ReVpuWkxNMHBCZDBoM1dVUldVakJxUWtKbmQwWnZRVlV6T1ZCd2VqRlphMFZhWWpWeFRtcHdTMFpYYVhocE5Ga0tXa1E0ZDJkWldVZEJNVlZrUlZGRlFpOTNVamhOU0hGSFpVZG9NR1JJUW5wUGFUaDJXakpzTUdGSVZtbE1iVTUyWWxNNWRtTkhWblZNV0ZKc1lrZFdkQXBhV0ZKNVpWTTVkbU5IVm5Wa1IxWnpXbGN4YkdSSVNqVk1WMDUyWWtkNGJGa3pVblpqYVRGNVdsZDRiRmxZVG14amVUaDFXakpzTUdGSVZtbE1NMlIyQ21OdGRHMWlSemt6WTNrNWFWbFlUbXhNV0Vwc1lrZFdhR015VlhWbFYwWjBZa1ZDZVZwWFducE1NMUpvV2pOTmRtUnFRWFZQVkdkMVRVUkJOVUpuYjNJS1FtZEZSVUZaVHk5TlFVVkNRa04wYjJSSVVuZGplbTkyVEROU2RtRXlWblZNYlVacVpFZHNkbUp1VFhWYU1td3dZVWhXYVdSWVRteGpiVTUyWW01U2JBcGlibEYxV1RJNWRFMUNTVWREYVhOSFFWRlJRbWMzT0hkQlVVbEZRa2hDTVdNeVozZE9aMWxMUzNkWlFrSkJSMFIyZWtGQ1FYZFJiMDlYVlhsTlIwcHRDazVYVFhoT1JFcHNUbFJOZDA1NlFtcFpNa2swVFhwSmQxbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCWkVKbmIzSkNaMFZGUVZsUEwwMUJSVVVLUWtFNVUxcFhlR3haV0U1c1NVVk9kbUp1VW5saFYwbDNVRkZaUzB0M1dVSkNRVWRFZG5wQlFrSlJVWFppTTBKc1lta3hNRnBYZUd4aVYxWXdZMjVyZGdwaU0wSnNZbTVTYkdKSFZuUmFXRko1WlZNeGFtSXllSE5hVjA0d1lqTkpkR050Vm5OYVYwWjZXbGhOZDBoM1dVdExkMWxDUWtGSFJIWjZRVUpDWjFGU0NtTnRWbTFqZVRrd1dWZGtla3d6V1hkTWFtczBUR3BCZDA5M1dVdExkMWxDUWtGSFJIWjZRVUpEUVZGMFJFTjBiMlJJVW5kamVtOTJURE5TZG1FeVZuVUtURzFHYW1SSGJIWmliazExV2pKc01HRklWbWxrV0U1c1kyMU9kbUp1VW14aWJsRjFXVEk1ZEUxSlIwbENaMjl5UW1kRlJVRlpUeTlOUVVWS1FraHZUUXBsUjJnd1pFaENlazlwT0haYU1td3dZVWhXYVV4dFRuWmlVemwyWTBkV2RVeFlVbXhpUjFaMFdsaFNlV1ZUT1haalIxWjFaRWRXYzFwWE1XeGtTRW8xQ2t4WFRuWmlSM2hzV1ROU2RtTnBNWGxhVjNoc1dWaE9iR041T0hWYU1td3dZVWhXYVV3elpIWmpiWFJ0WWtjNU0yTjVPV2xaV0U1c1RGaEtiR0pIVm1nS1l6SlZkV1ZYUm5SaVJVSjVXbGRhZWt3elVtaGFNMDEyWkdwQmRVOVVaM1ZOUkVFMFFtZHZja0puUlVWQldVOHZUVUZGUzBKRGIwMUxSR3hzVFdwQ2FRcGFhbFpxVFZSUmVWcFVWWHBOUkdOM1dUSk9hVTlFVFhsTlIwVjVUVVJOZUU1WFdtMWFiVWt3VFZSUk1rOVhWWGRJVVZsTFMzZFpRa0pCUjBSMmVrRkNDa04zVVZCRVFURnVZVmhTYjJSWFNYUmhSemw2WkVkV2EwMUdTVWREYVhOSFFWRlJRbWMzT0hkQlVYZEZVa0Y0UTJGSVVqQmpTRTAyVEhrNWJtRllVbThLWkZkSmRWa3lPWFJNTWpsM1dsYzBkR1JIVm5OYVZ6RnNaRWhLTlV3eU9YZGFWelV3V2xkNGJHSlhWakJqYm10MFdUSTVjMkpIVm1wa1J6bDVURmhLYkFwaVIxWm9ZekpXZWsxRVowZERhWE5IUVZGUlFtYzNPSGRCVVRCRlMyZDNiMDlYVlhsTlIwcHRUbGROZUU1RVNteE9WRTEzVG5wQ2Fsa3lTVFJOZWtsM0NsbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCYUVKbmIzSkNaMFZGUVZsUEwwMUJSVTlDUWsxTlJWaEtiRnB1VFhaa1IwWnVZM2s1TWsxRE5EVUtUME0wZDAxQ2EwZERhWE5IUVZGUlFtYzNPSGRCVVRoRlEzZDNTazVFUVhkTmFsVjZUbXBqTWsxRVJVZERhWE5IUVZGUlFtYzNPSGRCVWtGRlNYZDNhQXBoU0ZJd1kwaE5Oa3g1T1c1aFdGSnZaRmRKZFZreU9YUk1NamwzV2xjMGRHUkhWbk5hVnpGc1pFaEtOVTFDWjBkRGFYTkhRVkZSUW1jM09IZEJVa1ZGQ2tObmQwbE9SR3MxVDFSbmQwMUVTWGRuV1hOSFEybHpSMEZSVVVKbk56aDNRVkpKUldaUmVEZGhTRkl3WTBoTk5reDVPVzVoV0ZKdlpGZEpkVmt5T1hRS1RESTVkMXBYTkhSa1IxWnpXbGN4YkdSSVNqVk1NamwzV2xjMU1GcFhlR3hpVjFZd1kyNXJkRmt5T1hOaVIxWnFaRWM1ZVV4WVNteGlSMVpvWXpKV2VncE1lVFZ1WVZoU2IyUlhTWFprTWpsNVlUSmFjMkl6WkhwTU0wcHNZa2RXYUdNeVZYUlpNamwxWkVoS2NGbHBOVFZaVnpGelVVaEtiRnB1VFhaa1IwWnVDbU41T1RKTlF6UTFUME0wZDAxRVowZERhWE5IUVZGUlFtYzNPSGRCVWsxRlMyZDNiMDlYVlhsTlIwcHRUbGROZUU1RVNteE9WRTEzVG5wQ2Fsa3lTVFFLVFhwSmQxbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCVlVKbmIzSkNaMFZGUVZsUEwwMUJSVlZDUVZsTlFraENNV015WjNka1VWbExTM2RaUWdwQ1FVZEVkbnBCUWtaUlVtNUVSMVp2WkVoU2QyTjZiM1pNTW1Sd1pFZG9NVmxwTldwaU1qQjJZak5DYkdKcE1UQmFWM2hzWWxkV01HTnVhM1ppTTBKc0NtSnVVbXhpUjFaMFdsaFNlV1ZUTVdwaU1uaHpXbGRPTUdJelNYUmpiVlp6V2xkR2VscFlUWFpaVjA0d1lWYzVkV041T1hsa1Z6VjZUSHBuTWs1RVJYZ0tUbnBGTVU1cVkzWlpXRkl3V2xjeGQyUklUWFpOYWtGWFFtZHZja0puUlVWQldVOHZUVUZGVjBKQlowMUNia0l4V1cxNGNGbDZRMEpwWjFsTFMzZFpRZ3BDUVVoWFpWRkpSVUZuVWpoQ1NHOUJaVUZDTWtGT01EbE5SM0pIZUhoRmVWbDRhMlZJU214dVRuZExhVk5zTmpRemFubDBMelJsUzJOdlFYWkxaVFpQQ2tGQlFVSnFjM1JvUlVOUlFVRkJVVVJCUldOM1VsRkpaMWg2Y2xaME0xQjRkU3ROWVZKRkswUkdORzlGUldNMGVucHphSGR1VDJ4bGMwZGlla2xwYnpNS0wxWmpRMGxSUkZNelJ6QmlNemRhYUhRNGFITjJUSEozYkc1UFFXYzJWRXh1U1ZSS09HTjNkMVEzTW5sMVRVdFlUbFJCUzBKblozRm9hMnBQVUZGUlJBcEJkMDV1UVVSQ2EwRnFRWGxFUkZSYVFqQlRPVXBGYkZsSGJuTnZWVmhLYm04MU5Fc3ZUVUZUTlN0RFFVMU9lbWRqUWpWQ2JrRk5OMWhNUjBoV01HRnhDbVpaY21weFkyOXFia3RaUTAxSFRWRnFjalpUVGt0Q2NVaEtZVGwxTDBSTlQySlpNa0pKTVV0ME4yTnhOemhFT0VOcVMzQmFVblJoYnpadFVVMUVZMk1LUms5M2VYWnhWalJPVld0dlpsRTlQUW90TFMwdExVVk9SQ0JEUlZKVVNVWkpRMEZVUlMwdExTMHRDZz09In19fX0=","integratedTime":1712809120,"logIndex":84797936,"logID":"c0d23d6ad406973f9559f3ba2d1ca01f84147d8ffc5b8445c224f98b9591801d"}},"Issuer":"https://token.actions.githubusercontent.com","Subject":"https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/base-release.yaml@refs/tags/v0.98.0","githubWorkflowName":"Release Contrib","githubWorkflowRef":"refs/tags/v0.98.0","githubWorkflowRepository":"open-telemetry/opentelemetry-collector-releases","githubWorkflowSha":"9e20bf5c142e53070ccb8320a20315fffb41469e","githubWorkflowTrigger":"push"}},{"critical":{"identity":{"docker-reference":"ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib"},"image":{"docker-manifest-digest":"sha256:5cea85bcbc734a3c0a641368e5a4ea9d31b472997e9f2feca57eeb4a147fcf1a"},"type":"cosign container image signature"},"optional":{"1.3.6.1.4.1.57264.1.1":"https://token.actions.githubusercontent.com","1.3.6.1.4.1.57264.1.2":"push","1.3.6.1.4.1.57264.1.3":"9e20bf5c142e53070ccb8320a20315fffb41469e","1.3.6.1.4.1.57264.1.4":"Release Contrib","1.3.6.1.4.1.57264.1.5":"open-telemetry/opentelemetry-collector-releases","1.3.6.1.4.1.57264.1.6":"refs/tags/v0.98.0","Bundle":{"SignedEntryTimestamp":"MEUCIQD1ehDnPO6fzoPIpeQ3KFuYHHBiX7RcEbpo9B2r7JAlzwIgZ1bsuQz7gAXbNU1IEdsTQgfAnRk3xVXO16GnKXM2sAQ=","Payload":{"body":"eyJhcGlWZXJzaW9uIjoiMC4wLjEiLCJraW5kIjoiaGFzaGVkcmVrb3JkIiwic3BlYyI6eyJkYXRhIjp7Imhhc2giOnsiYWxnb3JpdGhtIjoic2hhMjU2IiwidmFsdWUiOiIxMzVjY2RlN2YzZTNhYjU2NmFmYzJhYWU3MDljYmJlNmFhMDZlZWMzNDA2MWNkZjMyNmRhYzM2MmY0NWM4Yjg4In19LCJzaWduYXR1cmUiOnsiY29udGVudCI6Ik1FUUNJRU92QXl0aE5RVGNvNHFMdG9GZUVOV0toNCtEK2I5SUxyYWhoa09WMmVBM0FpQjNEL2FpUGd1T05zUlB5alhaWk1hdnlCam0vMkVxNFNUMkZJWHozTnpyYWc9PSIsInB1YmxpY0tleSI6eyJjb250ZW50IjoiTFMwdExTMUNSVWRKVGlCRFJWSlVTVVpKUTBGVVJTMHRMUzB0Q2sxSlNVaHBSRU5EUW5jMlowRjNTVUpCWjBsVlZuRlRLMnd4WXpoMWVFUktOWEppZDAxMlVuaDBSR3hXVW1nMGQwTm5XVWxMYjFwSmVtb3dSVUYzVFhjS1RucEZWazFDVFVkQk1WVkZRMmhOVFdNeWJHNWpNMUoyWTIxVmRWcEhWakpOVWpSM1NFRlpSRlpSVVVSRmVGWjZZVmRrZW1SSE9YbGFVekZ3WW01U2JBcGpiVEZzV2tkc2FHUkhWWGRJYUdOT1RXcFJkMDVFUlhoTlJGRjRUMFJSZVZkb1kwNU5hbEYzVGtSRmVFMUVVWGxQUkZGNVYycEJRVTFHYTNkRmQxbElDa3R2V2tsNmFqQkRRVkZaU1V0dldrbDZhakJFUVZGalJGRm5RVVYyWlRCdGJrRkdRVzl1TVZoUGRIVlRMMXBNT0djeE5YUlJkVmxPTmtRemVUUlBWM0FLT1ZSTFMwUlVkRkJHU2xST1ZrWlJkVTlKUWs1bVJqWk1ORTlGYkd4dlZuUndaSE5uYjB0NVZGTnlPR3hTV1c1S1JIRlBRMEpwTUhkbloxbHdUVUUwUndwQk1WVmtSSGRGUWk5M1VVVkJkMGxJWjBSQlZFSm5UbFpJVTFWRlJFUkJTMEpuWjNKQ1owVkdRbEZqUkVGNlFXUkNaMDVXU0ZFMFJVWm5VVlZDSzFkSENuVmtlRE5IZUcxS1RWUkpUVVJyYW13clJtdzFXRzkzZDBoM1dVUldVakJxUWtKbmQwWnZRVlV6T1ZCd2VqRlphMFZhWWpWeFRtcHdTMFpYYVhocE5Ga0tXa1E0ZDJkWldVZEJNVlZrUlZGRlFpOTNVamhOU0hGSFpVZG9NR1JJUW5wUGFUaDJXakpzTUdGSVZtbE1iVTUyWWxNNWRtTkhWblZNV0ZKc1lrZFdkQXBhV0ZKNVpWTTVkbU5IVm5Wa1IxWnpXbGN4YkdSSVNqVk1WMDUyWWtkNGJGa3pVblpqYVRGNVdsZDRiRmxZVG14amVUaDFXakpzTUdGSVZtbE1NMlIyQ21OdGRHMWlSemt6WTNrNWFWbFlUbXhNV0Vwc1lrZFdhR015VlhWbFYwWjBZa1ZDZVZwWFducE1NMUpvV2pOTmRtUnFRWFZQVkdkMVRVUkJOVUpuYjNJS1FtZEZSVUZaVHk5TlFVVkNRa04wYjJSSVVuZGplbTkyVEROU2RtRXlWblZNYlVacVpFZHNkbUp1VFhWYU1td3dZVWhXYVdSWVRteGpiVTUyWW01U2JBcGlibEYxV1RJNWRFMUNTVWREYVhOSFFWRlJRbWMzT0hkQlVVbEZRa2hDTVdNeVozZE9aMWxMUzNkWlFrSkJSMFIyZWtGQ1FYZFJiMDlYVlhsTlIwcHRDazVYVFhoT1JFcHNUbFJOZDA1NlFtcFpNa2swVFhwSmQxbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCWkVKbmIzSkNaMFZGUVZsUEwwMUJSVVVLUWtFNVUxcFhlR3haV0U1c1NVVk9kbUp1VW5saFYwbDNVRkZaUzB0M1dVSkNRVWRFZG5wQlFrSlJVWFppTTBKc1lta3hNRnBYZUd4aVYxWXdZMjVyZGdwaU0wSnNZbTVTYkdKSFZuUmFXRko1WlZNeGFtSXllSE5hVjA0d1lqTkpkR050Vm5OYVYwWjZXbGhOZDBoM1dVdExkMWxDUWtGSFJIWjZRVUpDWjFGU0NtTnRWbTFqZVRrd1dWZGtla3d6V1hkTWFtczBUR3BCZDA5M1dVdExkMWxDUWtGSFJIWjZRVUpEUVZGMFJFTjBiMlJJVW5kamVtOTJURE5TZG1FeVZuVUtURzFHYW1SSGJIWmliazExV2pKc01HRklWbWxrV0U1c1kyMU9kbUp1VW14aWJsRjFXVEk1ZEUxSlIwbENaMjl5UW1kRlJVRlpUeTlOUVVWS1FraHZUUXBsUjJnd1pFaENlazlwT0haYU1td3dZVWhXYVV4dFRuWmlVemwyWTBkV2RVeFlVbXhpUjFaMFdsaFNlV1ZUT1haalIxWjFaRWRXYzFwWE1XeGtTRW8xQ2t4WFRuWmlSM2hzV1ROU2RtTnBNWGxhVjNoc1dWaE9iR041T0hWYU1td3dZVWhXYVV3elpIWmpiWFJ0WWtjNU0yTjVPV2xaV0U1c1RGaEtiR0pIVm1nS1l6SlZkV1ZYUm5SaVJVSjVXbGRhZWt3elVtaGFNMDEyWkdwQmRVOVVaM1ZOUkVFMFFtZHZja0puUlVWQldVOHZUVUZGUzBKRGIwMUxSR3hzVFdwQ2FRcGFhbFpxVFZSUmVWcFVWWHBOUkdOM1dUSk9hVTlFVFhsTlIwVjVUVVJOZUU1WFdtMWFiVWt3VFZSUk1rOVhWWGRJVVZsTFMzZFpRa0pCUjBSMmVrRkNDa04zVVZCRVFURnVZVmhTYjJSWFNYUmhSemw2WkVkV2EwMUdTVWREYVhOSFFWRlJRbWMzT0hkQlVYZEZVa0Y0UTJGSVVqQmpTRTAyVEhrNWJtRllVbThLWkZkSmRWa3lPWFJNTWpsM1dsYzBkR1JIVm5OYVZ6RnNaRWhLTlV3eU9YZGFWelV3V2xkNGJHSlhWakJqYm10MFdUSTVjMkpIVm1wa1J6bDVURmhLYkFwaVIxWm9ZekpXZWsxRVowZERhWE5IUVZGUlFtYzNPSGRCVVRCRlMyZDNiMDlYVlhsTlIwcHRUbGROZUU1RVNteE9WRTEzVG5wQ2Fsa3lTVFJOZWtsM0NsbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCYUVKbmIzSkNaMFZGUVZsUEwwMUJSVTlDUWsxTlJWaEtiRnB1VFhaa1IwWnVZM2s1TWsxRE5EVUtUME0wZDAxQ2EwZERhWE5IUVZGUlFtYzNPSGRCVVRoRlEzZDNTazVFUVhkTmFsVjZUbXBqTWsxRVJVZERhWE5IUVZGUlFtYzNPSGRCVWtGRlNYZDNhQXBoU0ZJd1kwaE5Oa3g1T1c1aFdGSnZaRmRKZFZreU9YUk1NamwzV2xjMGRHUkhWbk5hVnpGc1pFaEtOVTFDWjBkRGFYTkhRVkZSUW1jM09IZEJVa1ZGQ2tObmQwbE9SR3MxVDFSbmQwMUVTWGRuV1hOSFEybHpSMEZSVVVKbk56aDNRVkpKUldaUmVEZGhTRkl3WTBoTk5reDVPVzVoV0ZKdlpGZEpkVmt5T1hRS1RESTVkMXBYTkhSa1IxWnpXbGN4YkdSSVNqVk1NamwzV2xjMU1GcFhlR3hpVjFZd1kyNXJkRmt5T1hOaVIxWnFaRWM1ZVV4WVNteGlSMVpvWXpKV2VncE1lVFZ1WVZoU2IyUlhTWFprTWpsNVlUSmFjMkl6WkhwTU0wcHNZa2RXYUdNeVZYUlpNamwxWkVoS2NGbHBOVFZaVnpGelVVaEtiRnB1VFhaa1IwWnVDbU41T1RKTlF6UTFUME0wZDAxRVowZERhWE5IUVZGUlFtYzNPSGRCVWsxRlMyZDNiMDlYVlhsTlIwcHRUbGROZUU1RVNteE9WRTEzVG5wQ2Fsa3lTVFFLVFhwSmQxbFVTWGROZWtVeFdtMWFiVmxxVVhoT1JGazFXbFJCVlVKbmIzSkNaMFZGUVZsUEwwMUJSVlZDUVZsTlFraENNV015WjNka1VWbExTM2RaUWdwQ1FVZEVkbnBCUWtaUlVtNUVSMVp2WkVoU2QyTjZiM1pNTW1Sd1pFZG9NVmxwTldwaU1qQjJZak5DYkdKcE1UQmFWM2hzWWxkV01HTnVhM1ppTTBKc0NtSnVVbXhpUjFaMFdsaFNlV1ZUTVdwaU1uaHpXbGRPTUdJelNYUmpiVlp6V2xkR2VscFlUWFpaVjA0d1lWYzVkV041T1hsa1Z6VjZUSHBuTWs1RVJYZ0tUbnBGTVU1cVkzWlpXRkl3V2xjeGQyUklUWFpOYWtGWFFtZHZja0puUlVWQldVOHZUVUZGVjBKQlowMUNia0l4V1cxNGNGbDZRMEpwZDFsTFMzZFpRZ3BDUVVoWFpWRkpSVUZuVWpsQ1NITkJaVkZDTTBGT01EbE5SM0pIZUhoRmVWbDRhMlZJU214dVRuZExhVk5zTmpRemFubDBMelJsUzJOdlFYWkxaVFpQQ2tGQlFVSnFjM1JvUjJKSlFVRkJVVVJCUldkM1VtZEphRUZQZUZNM2RteDRjVzVGYTBKVVRtSlZVRUpsUkZSbk0waGtlRlkyY0cxWk9FdGliREV6TjNBS1lWUnViMEZwUlVFelMyMUxVbU5uYWxBeVQzSmxORVpyVm5vNU4xaENNWGRsUzBOeWFXazFTMWx2UTB0bVkxRktSREJSZDBObldVbExiMXBKZW1vd1JRcEJkMDFFWVVGQmQxcFJTWGhCUzNwcVpHMUZTV2gzV21Kb1lVSlNlalk1Y1N0MWVrNVZSMmxhYlRWVk4xcE5aWFJMUTFSM1VFTkljRkZQVldvdlVERkJDa2R0YWt3elJucFFObTVpYkRGblNYZFNUbXN6UkhkNWMwOUJUMHhoUVVoR09IaHhZV0ZzT0U5WGNGRmFhRGh4TTJVMVNVSmFXR0ZWVkhocFlWbGFTM29LUXpWS1RGVlNWbnBMTURsd04wVjBUd290TFMwdExVVk9SQ0JEUlZKVVNVWkpRMEZVUlMwdExTMHRDZz09In19fX0=","integratedTime":1712809122,"logIndex":84797940,"logID":"c0d23d6ad406973f9559f3ba2d1ca01f84147d8ffc5b8445c224f98b9591801d"}},"Issuer":"https://token.actions.githubusercontent.com","Subject":"https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/base-release.yaml@refs/tags/v0.98.0","githubWorkflowName":"Release Contrib","githubWorkflowRef":"refs/tags/v0.98.0","githubWorkflowRepository":"open-telemetry/opentelemetry-collector-releases","githubWorkflowSha":"9e20bf5c142e53070ccb8320a20315fffb41469e","githubWorkflowTrigger":"push"}}]
```

> [!NOTE]
> We started signing the images with release `v0.95.0`

## Contributing

See the [Contributing Guide](CONTRIBUTING.md) for details.

Here is a list of community roles with current and previous members:

- Triagers ([@open-telemetry/collector-triagers](https://github.com/orgs/open-telemetry/teams/collector-triagers)):

  - [Andrzej Stencel](https://github.com/andrzej-stencel), Elastic
  - Actively seeking contributors to triage issues

- Emeritus Triagers:

   - [Andrew Hsu](https://github.com/andrewhsu)
   - [Alolita Sharma](https://github.com/alolita)
   - [Punya Biswal](https://github.com/punya)
   - [Steve Flanders](https://github.com/flands)

- Approvers ([@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)):

   - [Antoine Toulme](https://github.com/atoulme), Splunk
   - [Daniel Jaglowski](https://github.com/djaglowski), observIQ
   - [Evan Bradley](https://github.com/evan-bradley), Dynatrace
   - [Juraci Paixão Kröhling](https://github.com/jpkrohling), Grafana Labs
   - [Tyler Helmuth](https://github.com/TylerHelmuth), Honeycomb
   - [Yang Song](https://github.com/songy23), Datadog

- Emeritus Approvers:

   - [James Bebbington](https://github.com/james-bebbington)
   - [Jay Camp](https://github.com/jrcamp)
   - [Nail Islamov](https://github.com/nilebox)
   - [Owais Lone](https://github.com/owais)
   - [Rahul Patel](https://github.com/rghetia)
   - [Steven Karis](https://github.com/sjkaris)
   - [Anthony Mirabella](https://github.com/Aneurysm9)

- Maintainers ([@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)):

   - [Alex Boten](https://github.com/codeboten), Honeycomb
   - [Bogdan Drutu](https://github.com/BogdanDrutu), Snowflake
   - [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
   - [Pablo Baeyens](https://github.com/mx-psi), DataDog

- Emeritus Maintainers:

   - [Paulo Janotti](https://github.com/pjanotti)
   - [Tigran Najaryan](https://github.com/tigrannajaryan)

Learn more about roles in [Community membership](https://github.com/open-telemetry/community/blob/main/community-membership.md).
In addition to what is described at the organization-level, the SIG Collector requires all core approvers to take part in rotating
the role of the [release manager](./docs/release.md#release-manager).

Thanks to all the people who already contributed!

<a href="https://github.com/open-telemetry/opentelemetry-collector/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-collector" />
</a>
