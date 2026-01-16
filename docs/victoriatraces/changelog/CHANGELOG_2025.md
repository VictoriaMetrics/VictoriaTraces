---
weight: 2
title: Year 2025
search:
  weight: 0.1
menu:
  docs:
    identifier: vt-changelog-2025
    parent: vt-changelog
    weight: 2
tags:
  - metrics
aliases:
  - /victoriatraces/CHANGELOG_2025.html
  - /victoriatraces/changelog_2025
  - /victoriatraces/changelog/changelog_2025/index.html
  - /victoriatraces/changelog/changelog_2025/
---

## [v0.5.1](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.5.1)

Released at 2025-11-19

* SECURITY: upgrade Go builder from Go1.25.3 to Go1.25.4. See [the list of issues addressed in Go1.25.4](https://github.com/golang/go/issues?q=milestone%3AGo1.25.4%20label%3ACherryPickApproved).

* FEATURE: [logstorage](https://docs.victoriametrics.com/victorialogs/): upgrade VictoriaLogs dependency from [v1.36.1 to v1.38.0](https://github.com/VictoriaMetrics/VictoriaLogs/compare/v1.36.1...v1.38.0).

* BUGFIX: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): properly apply `maxDataSize` memory limits to the `snappy` and `zstd` encoded requests. It protects ingest endpoints from malicious requests.

## [v0.5.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.5.0)

Released at 2025-11-08

* SECURITY: upgrade Go builder from Go1.25.2 to Go1.25.3. See [the list of issues addressed in Go1.25.3](https://github.com/golang/go/issues?q=milestone%3AGo1.25.3%20label%3ACherryPickApproved).

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): support [OTLP/gRPC](https://opentelemetry.io/docs/specs/otlp/#otlpgrpc) data ingestion. It requires `-otlpGRPCListenAddr` flag to be set on Single-node VictoriaTraces or vtinsert. See [this doc](https://docs.victoriametrics.com/victoriatraces/data-ingestion/opentelemetry) for details. Thanks to @JayiceZ for the [pull request](https://github.com/VictoriaMetrics/VictoriaTraces/pull/59).

* BUGFIX: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtselect in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): return the correct error message and the total number when searching by trace ID yields no hits in the result. Thank @huan89983 for [the bug report](https://github.com/VictoriaMetrics/VictoriaTraces/issues/77).

## [v0.4.1](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.4.1)

Released at 2025-10-31

* FEATURE: add linux/s390x artifact to releases.

* BUGFIX: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): stop query at the earlier timestamp of the retention period when searching by a non-existed trace ID, and response earlier. See [#48](https://github.com/VictoriaMetrics/VictoriaTraces/issues/48) for details. Thank @JayiceZ for [the pull request](https://github.com/VictoriaMetrics/VictoriaTraces/pull/49).

## [v0.4.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.4.0)

Released at 2025-10-14

* SECURITY: upgrade Go builder from Go1.25.0 to Go1.25.2. See the list of issues addressed in [Go1.25.1](https://github.com/golang/go/issues?q=milestone%3AGo1.25.1%20label%3ACherryPickApproved) and [Go1.25.2](https://github.com/golang/go/issues?q=milestone%3AGo1.25.2%20label%3ACherryPickApproved).
* SECURITY: upgrade base docker image (Alpine) from 3.22.1 to 3.22.2. See [Alpine 3.22.2 release notes](https://www.alpinelinux.org/posts/Alpine-3.19.9-3.20.8-3.21.5-3.22.2-released.html).

* FEATURE: [logstorage](https://docs.victoriametrics.com/victorialogs/): upgrade VictoriaLogs dependency from [v1.33.1 to v1.36.1](https://github.com/VictoriaMetrics/VictoriaLogs/compare/v1.33.1...v1.36.1).
* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): (experimental) support Jaeger [service dependencies graph API](https://www.jaegertracing.io/docs/2.10/architecture/apis/#service-dependencies-graph). It requires `--servicegraph.enableTask=true` flag to be set on Single-node VictoriaTraces or each vtstorage instance. See [#52](https://github.com/VictoriaMetrics/VictoriaTraces/pull/52) for details.
* FEATURE: vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): distribute spans to vtstorages by trace ID instead of randomly. See [#65](https://github.com/VictoriaMetrics/VictoriaTraces/pull/65) for details.

* BUGFIX: all components: restore sorting order of summary and quantile metrics exposed by VictoriaTraces components on `/metrics` page. See [metrics#105](https://github.com/VictoriaMetrics/metrics/pull/105) for details.

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
