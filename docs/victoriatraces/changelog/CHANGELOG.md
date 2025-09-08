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

* FEATURE: [Single-node VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) and vtinsert in [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/): support json encoding in otel ingestion. See [#44](https://github.com/VictoriaMetrics/VictoriaTraces/pull/444) for details.

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
