---
weight: 4
title: Visualization in Grafana
disableToc: true
menu:
  docs:
    parent: "victoriatraces-querying"
    weight: 4
tags:
  - traces
aliases:
  - /victoriatraces/querying/grafana.html
---

## Jaeger Datasource

[Grafana Jaeger Datasource](https://grafana.com/docs/grafana/latest/datasources/jaeger/) allows you to query and visualize VictoriaTraces data in Grafana.

![Visualization with Grafana](grafana-jaeger.webp)

Simply click "Add new data source" on Grafana, and then fill your VictoriaTraces URL to "Connection.URL".

The URL format for VictoriaTraces single-node is:

```
http://<victoria-traces>:10428/select/jaeger
```

Finally, click "Save & Test" at the bottom to complete the process.

## Grafana Tempo Datasource

> Grafana Tempo datasource support is **experimental**. It's implemented as a complement to the Jaeger datasource, to allow using the [Grafana Traces Drilldown](https://grafana.com/docs/grafana-cloud/visualizations/simplified-exploration/traces/).
> It may not support some of the syntax in TraceQL or panels in drilldown.

[Grafana Tempo Datasource](https://grafana.com/docs/grafana/latest/datasources/tempo/) lets you query VictoriaTraces with
[TraceQL](https://grafana.com/docs/tempo/latest/traceql/) and use the [Grafana Traces Drilldown](https://grafana.com/docs/grafana-cloud/visualizations/simplified-exploration/traces/)
plugin for trace exploration. It is backed by the [Tempo HTTP API](https://docs.victoriametrics.com/victoriatraces/querying/#tempo-http-api)
that VictoriaTraces implements; see those docs for the list of supported endpoints and the versions they were introduced in.

Click "Add new data source" on Grafana, and then fill your VictoriaTraces URL to "Connection.URL".

The URL format for VictoriaTraces single-node is:

```
http://<victoria-traces>:10428/select/tempo
```

For [VictoriaTraces cluster](https://docs.victoriametrics.com/victoriatraces/cluster/), point the URL at vtselect instead:

```
http://<vtselect>:10428/select/tempo
```

Finally, click "Save & Test" at the bottom to complete the process.

### Grafana Traces Drilldown

The [Grafana Traces Drilldown](https://grafana.com/docs/grafana-cloud/visualizations/simplified-exploration/traces/) plugin
works on top of the Tempo datasource configured above. Once the datasource is added, open **Explore → Drilldown → Traces** (or
the Traces Drilldown app) and select the VictoriaTraces Tempo datasource.

Drilldown relies on the [TraceQL metrics](https://docs.victoriametrics.com/victoriatraces/querying/#traceql-metrics) endpoint to
render its rate, error, and duration panels, and on the [trace search](https://docs.victoriametrics.com/victoriatraces/querying/#searching-traces)
and [auto-completion](https://docs.victoriametrics.com/victoriatraces/querying/#auto-completion-of-tags-and-values) endpoints for
filtering and exploration. Support is **experimental**, so some panels or TraceQL features may not behave exactly as they do with Grafana Tempo.
