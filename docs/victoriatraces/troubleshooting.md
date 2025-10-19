---
weight: 20
title: Troubleshooting
menu:
  docs:
    identifier: vt-troubleshooting
    parent: victoriatraces
    weight: 20
    title: Troubleshooting
tags:
  - traces
aliases:
- /victoriatraces/troubleshooting.html
---

## Troubleshooting

This document contains troubleshooting guides for most common issues when working with VictoriaTraces:

- [Data ingestion](#data-ingestion)
- [Querying](#querying)

### Data ingestion

The following command can be used for verifying whether the data is successfully ingested into VictoriaTraces:

```sh
curl http://<victoria-traces>:10428/select/logsql/query -d 'query=*' | head
```

This command selects all the data ingested into VictoriaTraces via [HTTP query API](https://docs.victoriametrics.com/victoriatraces/querying/#http-api)
using [any value filter](https://docs.victoriametrics.com/victorialogs/logsql/#any-value-filter), while `head` cancels query execution after reading the first 10 trace spans.
See [these docs](https://docs.victoriametrics.com/victoriatraces/querying/#command-line) for more details on how `head` integrates with VictoriaTraces.

The response by default contains all the [trace span fields](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model).
See [how to query specific fields](https://docs.victoriametrics.com/victorialogs/logsql/#querying-specific-fields).

VictoriaTraces provides the following command-line flags, which can help debugging data ingestion issues:

- `-logNewStreams` - if this flag is passed to VictoriaTraces, then it traces all the newly
  registered [streams](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#stream-fields).
  This may help debugging [high cardinality issues](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#high-cardinality).
- `-logIngestedRows` - if this flag is passed to VictoriaTraces, then it traces all the ingested
  [trace span entries](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model).
  See also `debug` [parameter](#http-parameters).

VictoriaTraces exposes various metrics, which may help debugging data ingestion issues:

- `vt_rows_ingested_total` - the number of ingested [trace span entries](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model)
  since the last VictoriaTraces restart. If this number increases over time, then trace spans are successfully ingested into VictoriaTraces.
  The ingested trace spans can be inspected in the following ways:
  - By passing `debug=1` parameter to every request to [data ingestion APIs](#http-apis). The ingested spans aren't stored in VictoriaTraces
      in this case. Instead, they are logged, so they can be investigated later.
      The `vt_rows_dropped_total` metric is incremented for each logged row.
  - By passing `-logIngestedRows` command-line flag to VictoriaTraces. In this case it traces all the ingested data, so it can be investigated later.
- `vt_streams_created_total` - the number of created [trace streams](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#stream-fields)
  since the last VictoriaTraces restart. If this metric grows rapidly during extended periods of time, then this may lead
  to [high cardinality issues](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#high-cardinality).
  The newly created trace streams can be inspected in traces by passing `-logNewStreams` command-line flag to VictoriaTraces.

### Querying
