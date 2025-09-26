---
build:
  list: never
  publishResources: false
  render: never
sitemap:
  disable: true
---
The following `tip` changes can be tested by building VictoriaTraces components from the latest commits according to the following docs:

* [How to build single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/#how-to-build-from-sources)

## tip

* SECURITY: upgrade Go builder from Go1.25.0 to Go1.25.1. See [the list of issues addressed in Go1.25.1](https://github.com/golang/go/issues?q=milestone%3AGo1.25.1%20label%3ACherryPickApproved).

## [v0.3.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.3.0)

Released at 2025-09-19

* FEATURE: improve the scalability of data ingestion on systems with big number of CPU cores. Previously only up to 40 CPU cores were used during logs' ingestion into VictoriaLogs on AMD64 and ARM64 architectures, while the remaining CPU cores were idle. Remove the scalability bottleneck by switching from [musl-based](https://wiki.musl-libc.org/) to [glibc-based](https://en.wikipedia.org/wiki/Glibc) cross-compiler. This improved the data ingestion speed on a host with hundreds of CPU cores by more than 4x. See [#517](https://github.com/VictoriaMetrics/VictoriaLogs/issues/517#issuecomment-3167039079).
* FEATURE: upgrade Go builder from Go1.24.6 to Go1.25.0. See [Go1.25.0 release notes](https://go.dev/doc/go1.25).
* FEATURE: [logstorage](https://docs.victoriametrics.com/victorialogs/): Upgrade VictoriaLogs dependency from [v1.27.0 to v1.33.1](https://github.com/VictoriaMetrics/VictoriaLogs/compare/v1.27.0...v1.33.1).
* FEATURE: [docker compose](https://github.com/VictoriaMetrics/VictoriaTraces/tree/master/deployment/docker): add cluster docker compose environment.
* FEATURE: [dashboards](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/dashboards): update dashboard for VictoriaTraces single-node and cluster to provide more charts.
* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): support [JSON protobuf encoding](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding) in the OpenTelemetry protocol (OTLP) for data ingestion. See [this issue](https://github.com/VictoriaMetrics/VictoriaTraces/issues/41) for details. Thanks to @JayiceZ for the [pull request](https://github.com/VictoriaMetrics/VictoriaTraces/pull/51).

* BUGFIX: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): Rename various [HTTP headers](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#http-headers) prefix from `VL-` to `VT-`. These headers help with debugging and customizing stream fields. Thank @JayiceZ for [the pull request](https://github.com/VictoriaMetrics/VictoriaTraces/pull/56). 
* BUGFIX: all components: properly expose metadata for summaries and histograms in VictoriaMetrics components with enabled `-metrics.exposeMetadata` cmd-line flag. See [metrics#98](https://github.com/VictoriaMetrics/metrics/issues/98) for details.

## [v0.2.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.2.0)

Released at 2025-09-01

* SECURITY: upgrade Go builder from Go1.24.5 to Go1.24.6. See [the list of issues addressed in Go1.24.6](https://github.com/golang/go/issues?q=milestone%3AGo1.24.6+label%3ACherryPickApproved).
* SECURITY: upgrade base docker image (Alpine) from 3.22.0 to 3.22.1. See [Alpine 3.22.1 release notes](https://www.alpinelinux.org/posts/Alpine-3.19.8-3.20.7-3.21.4-3.22.1-released.html).

* FEATURE: [logstorage](https://docs.victoriametrics.com/victorialogs/): Upgrade VictoriaLogs dependency from [v1.25.1 to v1.27.0](https://github.com/VictoriaMetrics/VictoriaLogs/compare/v1.25.1...v1.27.0).
* FEATURE: [dashboards](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/dashboards): add dashboard for VictoriaTraces single-node and cluster. 

## [v0.1.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.1.0)

Released at 2025-07-28

Initial release

## Previous releases

See [releases page](https://github.com/VictoriaMetrics/VictoriaMetrics/releases).
