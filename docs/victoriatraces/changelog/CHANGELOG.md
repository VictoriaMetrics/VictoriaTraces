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

* SECURITY: upgrade Go builder from Go1.25.5 to Go1.26.0. See [the list of issues addressed in Go1.26.0](https://github.com/golang/go/issues?q=milestone%3AGo1.26.0+label%3ACherryPickApproved).
* SECURITY: upgrade base docker image (Alpine) from 3.22.2 to 3.23.3. See [Alpine 3.23.3 release notes](https://www.alpinelinux.org/posts/Alpine-3.20.9-3.21.6-3.22.3-3.23.3-released.html).

* BUGFIX: fix VictoriaTraces Docker OCI labels `org.opencontainers.image.source` and `org.opencontainers.image.documentation`: point them to VictoriaTraces repo/docs instead of VictoriaMetrics.

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtselect, vtstorage in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): (experimental) add support for [Tempo datasource APIs](https://grafana.com/docs/tempo/latest/api_docs/). This starts with support for the basic auto-completion `/tags`, search `/search`, and `/v2/traces/*` APIs.
  TraceQL metrics and pipelines are not yet available in this release.


## [v0.7.1](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.7.1)

Released at 2026-01-24

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtstorage in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): adjust `-insert.indexFlushInterval` from 30s to 20s to ensure a proper time gap with `-search.latencyOffset`. This is useful when ingest data is not real-time, as it helps reduce the probability that data can be searched by condition but is not present in the traceID index, resulting in failure to query by traceID.

* BUGFIX: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtselect in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): fix backward compatibility for the old index format. Previously, the old index format was not parsed correctly into the start and end timestamps.

## [v0.7.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.7.0)

Released at 2026-01-21

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): add duration and error metrics for service graph background tasks. Thank @chenlujjj for [the pull request #100](https://github.com/VictoriaMetrics/VictoriaTraces/pull/100).
* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): add accurate start time and end time to the trace ID index. This should help with the trace lookup by ID, and will free user from configuring `-search.traceMaxDurationWindow` to avoid missing the spans. See [the pull request #81](https://github.com/VictoriaMetrics/VictoriaTraces/pull/81).

## [v0.6.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.6.0)

Released at 2026-01-07

* SECURITY: upgrade Go builder from Go1.25.4 to Go1.25.5. See [the list of issues addressed in Go1.25.5](https://github.com/golang/go/issues?q=milestone%3AGo1.25.5%20label%3ACherryPickApproved).

* FEATURE: [logstorage](https://docs.victoriametrics.com/victorialogs/): upgrade VictoriaLogs dependency from [v1.38.0 to v1.43.1](https://github.com/VictoriaMetrics/VictoriaLogs/compare/v1.38.0...v1.43.1).
* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): reduce CPU usage of the OpenTelemetry protobuf data ingestion in both OTLP/HTTP and OTLP/gRPC APIs. Thanks to @vadimalekseev for [the initial idea](https://github.com/VictoriaMetrics/VictoriaLogs/pull/720).

## [v0.5.1](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.5.1)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v051)

## [v0.5.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.5.0)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v050)

## [v0.4.1](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.4.1)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v041)

## [v0.4.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.4.0)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v040)

## [v0.3.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.3.0)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v030)

## [v0.2.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.2.0)

See changes [here](https://docs.victoriametrics.com/victoriatraces/changelog/changelog_2025/#v020)

## [v0.1.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.1.0)

Released at 2025-07-28

Initial release

## Previous releases

See [releases page](https://github.com/VictoriaMetrics/VictoriaMetrics/releases).
