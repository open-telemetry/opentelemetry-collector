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
</p>

<p align="center">
  <strong>
    <a href="docs/vision.md">Vision</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/design.md">Design</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://opentelemetry.io/docs/collector/configuration/">Configuration</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/monitoring.md">Monitoring</a>
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

Official OpenTelemetry Collector distro binaries may be built with any supported Go version.

## Verifying the images signatures

> **Note**: To verify a signed artifact or blob, first [install Cosign](https://docs.sigstore.dev/system_config/installation/), then follow the instructions below.

We are signing the images `otel/opentelemetry-collector` and `otel/opentelemetry-collector-contrib` using [sigstore cosign](https://github.com/sigstore/cosign) tool and to verify the signatures you can run the following command:

```console
$ cosign verify \
  --certificate-identity=https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/release.yaml@refs/tags/<RELEASE_TAG>
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  <OTEL_COLLECTOR_IMAGE>
```

where:

- `<RELEASE_TAG>`: is the release that you want to validate
- `<OTEL_COLLECTOR_IMAGE>`: is the image that you want to check

Example:

```console
$ cosign verify --certificate-identity=https://github.com/open-telemetry/opentelemetry-collector-releases/.github/workflows/release.yaml@refs/tags/v99.99.01 --certificate-oidc-issuer=https://token.actions.githubusercontent.com ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:99.99.01

Verification for ghcr.io/cpanato/opentelemetry-collector-releases/opentelemetry-collector:99.99.01 --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The code-signing certificate was verified using trusted certificate authority certificates

[{"critical":{"identity":{"docker-reference":"ghcr.io/cpanato/opentelemetry-collector-releases/opentelemetry-collector"},"image":{"docker-manifest-digest":"sha256:94b02330e851e3abde5daba2fd8fbcfad7460304091105c3a833023d1f72ad41"},"type":"cosign container image signature"},"optional":{"1.3.6.1.4.1.57264.1.1":"https://token.actions.githubusercontent.com","1.3.6.1.4.1.57264.1.2":"push","1.3.6.1.4.1.57264.1.3":"ae8f64b2f70b623314258dfb858006c1439d7e28","1.3.6.1.4.1.57264.1.4":"Release","1.3.6.1.4.1.57264.1.5":"cpanato/opentelemetry-collector-releases","1.3.6.1.4.1.57264.1.6":"refs/tags/v99.99.01","Bundle":{"SignedEntryTimestamp":"MEUCIDnW5HXg7doXPabv9ijQm448R9RHkw3ZkwXdg7f5eEXGAiEA5brTNQgqYHeBnZmJRLBOaapD8DMIrQPJ2XZYY/dYWjg=","Payload":{"body":"eyJhcGlWZXJzaW9uIjoiMC4wLjEiLCJraW5kIjoiaGFzaGVkcmVrb3JkIiwic3BlYyI6eyJkYXRhIjp7Imhhc2giOnsiYWxnb3JpdGhtIjoic2hhMjU2IiwidmFsdWUiOiJmYmY1MTVhMTZmNzM4ZWJkOGZjMDdkYWQ5NWIyYTdkZDQ4YzRhNmRiMzYyMDE4NjhlZTI0YmIyYjgzMDBkNzVmIn19LCJzaWduYXR1cmUiOnsiY29udGVudCI6Ik1FUUNJREFWN0MzaGM0R1FIKzNtL3p3VUdkODhEUjNUODh0N0ErMkFkamhBWllKUEFpQXFEUFVXVDMvcS8zZXhBT2JoQnFGNFlaVlprZnlVZlJuQ1JRMTlyZnQ5ckE9PSIsInB1YmxpY0tleSI6eyJjb250ZW50IjoiTFMwdExTMUNSVWRKVGlCRFJWSlVTVVpKUTBGVVJTMHRMUzB0Q2sxSlNVUjVha05EUVRBclowRjNTVUpCWjBsVlFYcEZaSEJtZWtaaGRGbEZjbkZ4U1hwbU5YVm9WSEJ4VkU4MGQwTm5XVWxMYjFwSmVtb3dSVUYzVFhjS1RucEZWazFDVFVkQk1WVkZRMmhOVFdNeWJHNWpNMUoyWTIxVmRWcEhWakpOVWpSM1NFRlpSRlpSVVVSRmVGWjZZVmRrZW1SSE9YbGFVekZ3WW01U2JBcGpiVEZzV2tkc2FHUkhWWGRJYUdOT1RXcEpkMDlVVFhkTlZFMHhUV3BKTTFkb1kwNU5ha2wzVDFSTmQwMVVVWGROYWtrelYycEJRVTFHYTNkRmQxbElDa3R2V2tsNmFqQkRRVkZaU1V0dldrbDZhakJFUVZGalJGRm5RVVYwUld0aFlsWnpVRXQ1YzJSSlZYaFlibEJoZVRVME9EUmpOVGRsVnpGaUszcFdaMHNLU3psaFRITXdOSHBwY0hGb1oxbE1SSGRoVlUxQlRsUXdUaXRCTUhGSVJHUXdkRVptUlZocmQyRm5kRFoyWWs5SVRXRlBRMEZ0TkhkblowcHhUVUUwUndwQk1WVmtSSGRGUWk5M1VVVkJkMGxJWjBSQlZFSm5UbFpJVTFWRlJFUkJTMEpuWjNKQ1owVkdRbEZqUkVGNlFXUkNaMDVXU0ZFMFJVWm5VVlZ1YUhaT0NtaHhablV2V1d0NVl6QllUemRhV0NzMldFaG9iVEIzZDBoM1dVUldVakJxUWtKbmQwWnZRVlV6T1ZCd2VqRlphMFZhWWpWeFRtcHdTMFpYYVhocE5Ga0tXa1E0ZDJaQldVUldVakJTUVZGSUwwSklTWGRqU1ZwMVlVaFNNR05JVFRaTWVUbHVZVmhTYjJSWFNYVlpNamwwVERKT2QxbFhOV2hrUnpoMllqTkNiQXBpYmxKc1lrZFdkRnBZVW5sbFV6RnFZako0YzFwWFRqQmlNMGwwWTIxV2MxcFhSbnBhV0UxMlRHMWtjR1JIYURGWmFUa3pZak5LY2xwdGVIWmtNMDEyQ21OdFZuTmFWMFo2V2xNMU5WbFhNWE5SU0Vwc1dtNU5kbVJIUm01amVUa3lUMVJyZFU5VWEzVk5SRVYzVDFGWlMwdDNXVUpDUVVkRWRucEJRa0ZSVVhJS1lVaFNNR05JVFRaTWVUa3dZakowYkdKcE5XaFpNMUp3WWpJMWVreHRaSEJrUjJneFdXNVdlbHBZU21waU1qVXdXbGMxTUV4dFRuWmlWRUZUUW1kdmNncENaMFZGUVZsUEwwMUJSVU5DUVZKM1pGaE9iMDFFV1VkRGFYTkhRVkZSUW1jM09IZEJVVTFGUzBkR2JFOUhXVEpPUjBsNVdtcGpkMWxxV1hsTmVrMTRDazVFU1RGUFIxSnRXV3BuTVU5RVFYZE9iVTE0VGtSTk5WcEVaR3hOYW1kM1JsRlpTMHQzV1VKQ1FVZEVkbnBCUWtKQlVVaFZiVlp6V2xkR2VscFVRVElLUW1kdmNrSm5SVVZCV1U4dlRVRkZSa0pEYUdwalIwWjFXVmhTZGt3eU9YZGFWelV3V2xkNGJHSlhWakJqYm10MFdUSTVjMkpIVm1wa1J6bDVURmhLYkFwaVIxWm9ZekpXZWsxRFJVZERhWE5IUVZGUlFtYzNPSGRCVVZsRlJUTktiRnB1VFhaa1IwWnVZM2s1TWs5VWEzVlBWR3QxVFVSRmQyZFpjMGREYVhOSENrRlJVVUl4Ym10RFFrRkpSV1pSVWpkQlNHdEJaSGRCU1ZsS1RIZExSa3d2WVVWWVVqQlhjMjVvU25oR1duaHBjMFpxTTBSUFRrcDBOWEozYVVKcVduWUtZMmRCUVVGWlQwOXhORUUwUVVGQlJVRjNRa2xOUlZsRFNWRkVhWGs0TkVaak5GRkpkVXhRTTFkMFNtTkdhbVUzYWtoMU5HVkZXRzVVVFdsQ2RXMWpUd3BRUzFKSE0zZEphRUZRV2psdVpYSjRaWGRFZVZncldUUXlSMVl3Tm5CbWNFMUZRVnB1ZDNWcVZsSlJZa0pQYjNOMVExUk5UVUZ2UjBORGNVZFRUVFE1Q2tKQlRVUkJNbXRCVFVkWlEwMVJRMlJLWkRONGFUaFlRMmhvVjI5eVlsWlpUVVZ5YWtSR1YyYzFZV2xJZVM5d0wycG1hRFphV2t3MVMxTjVaRzFvT1drS1ozbEViM2h3U2poa2NHSm5OMGRuUTAxUlEwVlFVMFJqVEZodFVUZ3haU3MyUm5GUk5VMVBWRUptTlU1TUwxUmFhVUZVYzNKRGFFdFJTR0ZyTURKTWF3cFFNbmRaYW1SNlRIQkJNa05NUWtoMFVGUkZQUW90TFMwdExVVk9SQ0JEUlZKVVNVWkpRMEZVUlMwdExTMHRDZz09In19fX0=","integratedTime":1664545948,"logIndex":4288624,"logID":"c0d23d6ad406973f9559f3ba2d1ca01f84147d8ffc5b8445c224f98b9591801d"}},"Issuer":"https://token.actions.githubusercontent.com","Subject":"https://github.com/cpanato/opentelemetry-collector-releases/.github/workflows/release.yaml@refs/tags/v99.99.01","githubWorkflowName":"Release","githubWorkflowRef":"refs/tags/v99.99.01","githubWorkflowRepository":"cpanato/opentelemetry-collector-releases","githubWorkflowSha":"ae8f64b2f70b623314258dfb858006c1439d7e28","githubWorkflowTrigger":"push"}}]
```

> **Note**: We started signing the images with release x.y.z

## Contributing

See the [Contributing Guide](CONTRIBUTING.md) for details.

Here is a list of community roles with current and previous members:

- Triagers ([@open-telemetry/collector-triagers](https://github.com/orgs/open-telemetry/teams/collector-triagers)):

  - [Andrzej Stencel](https://github.com/astencel-sumo), Sumo Logic
  - [Antoine Toulme](https://github.com/atoulme), Splunk
  - [Evan Bradley](https://github.com/evan-bradley), Dynatrace
  - [Tyler Helmuth](https://github.com/TylerHelmuth), Honeycomb
  - [Yang Song](https://github.com/songy23), Datadog
  - Actively seeking contributors to triage issues

- Emeritus Triagers:

   - [Andrew Hsu](https://github.com/andrewhsu), Lightstep
   - [Alolita Sharma](https://github.com/alolita), Apple
   - [Punya Biswal](https://github.com/punya), Google
   - [Steve Flanders](https://github.com/flands), Splunk

- Approvers ([@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)):

   - [Anthony Mirabella](https://github.com/Aneurysm9), AWS
   - [Daniel Jaglowski](https://github.com/djaglowski), observIQ
   - [Juraci Paixão Kröhling](https://github.com/jpkrohling), Grafana Labs

- Emeritus Approvers:

   - [James Bebbington](https://github.com/james-bebbington), Google
   - [Jay Camp](https://github.com/jrcamp), Splunk
   - [Nail Islamov](https://github.com/nilebox), Google
   - [Owais Lone](https://github.com/owais), Splunk
   - [Rahul Patel](https://github.com/rghetia), Google
   - [Steven Karis](https://github.com/sjkaris), Splunk

- Maintainers ([@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)):

   - [Alex Boten](https://github.com/codeboten), ServiceNow
   - [Bogdan Drutu](https://github.com/BogdanDrutu), Snowflake
   - [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
   - [Pablo Baeyens](https://github.com/mx-psi), DataDog

- Emeritus Maintainers:

   - [Paulo Janotti](https://github.com/pjanotti), Splunk
   - [Tigran Najaryan](https://github.com/tigrannajaryan), Splunk

Learn more about roles in [Community membership](https://github.com/open-telemetry/community/blob/main/community-membership.md).
In addition to what is described at the organization-level, the SIG Collector requires all core approvers to take part in rotating
the role of the [release manager](./docs/release.md#release-manager).

Thanks to all the people who already contributed!

<a href="https://github.com/open-telemetry/opentelemetry-collector/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-collector" />
</a>
