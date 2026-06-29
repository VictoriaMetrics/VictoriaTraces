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

Click "Add new data source" on Grafana, and then fill your VictoriaTraces URL to "Connection.URL".

The URL format for VictoriaTraces single-node is:

```
http://<victoria-traces>:10428/select/tempo
```

Finally, click "Save & Test" at the bottom to complete the process.