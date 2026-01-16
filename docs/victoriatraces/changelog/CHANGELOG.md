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

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): add duration and error metrics for service graph background tasks. Thank @chenlujjj for [the pull request #100](https://github.com/VictoriaMetrics/VictoriaTraces/pull/100).

## [v0.6.0](https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/v0.6.0)

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
