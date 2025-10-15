---
weight: 1
title: Quick start
menu:
  docs:
    identifier: vt-quick-start
    parent: victoriatraces
    weight: 1
    title: Quick start
tags:
  - traces
aliases:
- /victoriatraces/quick-start.html
---

## Quick Start

### How to install

VictoriaTraces is available in the following distributions:

- Single-server-VictoriaTraces - all-in-one binary that is easy to run and maintain.

VictoriaMetrics is available as:

- docker images at [Docker Hub](https://hub.docker.com/r/victoriametrics/victoria-traces) and [Quay](https://quay.io/repository/victoriametrics/victoria-traces).
- [Binary releases](https://github.com/VictoriaMetrics/VictoriaTraces/releases/)
- [Source code](https://github.com/VictoriaMetrics/VictoriaTraces). See [How to build from sources](https://docs.victoriametrics.com/victoriatraces/#how-to-build-from-sources)

#### Starting VictoriaTraces Single Node via Docker

Run the newest available [VictoriaTraces release](https://docs.victoriametrics.com/victoriatraces/changelog/) from [Docker Hub](https://hub.docker.com/r/victoriametrics/victoria-traces) or [Quay](https://quay.io/repository/victoriametrics/victoria-traces):

```shell
docker run --rm -it -p 10428:10428 -v ./victoria-traces-data:/victoria-traces-data \
  docker.io/victoriametrics/victoria-traces:latest
```

This command will make VictoriaTraces run in the foreground, and store the ingested data to the `victoria-traces-data` directory. You should see the following logs:

```
2025-08-08T07:33:13.532Z	info	VictoriaTraces/app/victoria-traces/main.go:44	starting VictoriaTraces at "[:10428]"...
2025-08-08T07:33:13.532Z	info	VictoriaTraces/app/vtstorage/main.go:111	opening storage at -storageDataPath=victoria-traces-data
...
2025-08-08T07:33:13.542Z	info	VictoriaMetrics@v0.0.0-20250714222639-15242a70a79f/lib/httpserver/httpserver.go:145	started server at http://0.0.0.0:10428/
...
```

After VictoriaTraces is running, verify VMUI is working by going to `http://<victoria-traces>:10428/vmui`.

See how to [write](#write-data) or [read](#read-data) from VictoriaTraces.

#### Starting VictoriaTraces Single Node from a Binary

- Download the correct binary for your OS and architecture from [GitHub](https://github.com/VictoriaMetrics/VictoriaTraces/releases/). Here's an example for `Linux/amd64`:

```sh
curl -L -O https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.4.0/victoria-traces-linux-amd64-v0.4.0.tar.gz
```

- Extract the archive by running:

```sh
tar -xvf victoria-traces-linux-amd64-v0.4.0.tar.gz
```

- Go to the binary's folder and start VictoriaTraces:

```sh
./victoria-traces-prod
```

This command will make VictoriaTraces run in the foreground, and store the ingested data to the `victoria-traces-data` directory by default.

After VictoriaTraces is running, verify VMUI is working by going to `http://<victoria-traces>:10428/vmui`.

See how to [write](#write-data) or [read](#read-data) from VictoriaTraces.

### Write data

VictoriaTraces can accept trace spans via [the OpenTelemetry protocol (OTLP)](https://opentelemetry.io/docs/specs/otlp/). It provides the following API:

- `/insert/opentelemetry/v1/traces`

To test the data ingestion, run the following command:

```shell
echo '{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"frontend-web"}},{"key":"telemetry.sdk.language","value":{"stringValue":"webjs"}},{"key":"telemetry.sdk.name","value":{"stringValue":"opentelemetry"}},{"key":"telemetry.sdk.version","value":{"stringValue":"1.30.1"}},{"key":"process.runtime.name","value":{"stringValue":"browser"}},{"key":"process.runtime.description","value":{"stringValue":"Web Browser"}},{"key":"process.runtime.version","value":{"stringValue":"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}]},"scopeSpans":[{"scope":{"name":"@opentelemetry/instrumentation-document-load","version":"0.44.1"},"spans":[{"traceId":"1af5dd013a30efe7f2970032ab81958b","spanId":"229d083a6c480511","parentSpanId":"","name":"documentLoad","kind":1,"startTimeUnixNano":"ingestTimePlaceHolder","endTimeUnixNano":"ingestTimePlaceHolder","attributes":[{"key":"session.id","value":{"stringValue":"96e702c3-6f05-4f54-b2b3-2fad2b7b7995"}},{"key":"http.url","value":{"stringValue":"http://frontend-proxy:8080/cart"}},{"key":"http.user_agent","value":{"stringValue":"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}],"events":[{"timeUnixNano":"1757320936519100098","name":"fetchStart"}],"status":{}}]}]}]}' | 
sed "s/ingestTimePlaceHolder/$(date +%s000000000)/g" | 
curl -X POST -H 'Content-Type: application/json' --data-binary @- http://<victoria-traces>:10428/insert/opentelemetry/v1/traces
```

This command will send an HTTP request to VictoriaTraces and ingest one example span.

Alternatively, the following example application (HotROD) can be used:

```
docker run \
  -p8080-8083:8080-8083 \
  --rm \
  --env OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://<victoria-traces>:10428/insert/opentelemetry/v1/traces \
  jaegertracing/example-hotrod:latest \
  all
```

> Please make sure the host address in environment variable `http://<victoria-traces>` is accessible from the HotROD container.
> If you're running VictoriaTraces locally (via docker or binary), the simplest way would be to fill the host IP of your machine,
> such as `http://192.168.0.100`, which you can get from the `ifconfig`/`ipconfig` output.

Simply open `http://127.0.0.1:8080/`, click any button to generate traces.

After that, you can check the data in VMUI at `http://<victoria-traces>:10428/vmui`.

See more details about how to send data to VictoriaTraces from **an instrumented application** or **an OpenTelemetry collector** [in this doc](https://docs.victoriametrics.com/victoriatraces/data-ingestion/opentelemetry/).

### Read data

[VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) has built-in VMUI for browsing data by span at `http://<victoria-traces>:10428/vmui`.

[VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) also provides [Jaeger Query Service JSON APIs](https://www.jaegertracing.io/docs/2.6/apis/#internal-http-json).
It allows users to visualize trace data on Grafana, by simply adding a [Jaeger datasource](https://grafana.com/docs/grafana/latest/datasources/jaeger/) with VictoriaTraces URL:

```
http://<victoria-traces>:10428/select/jaeger
```

See more details about the HTTP APIs and params VictoriaTraces supports and how to query data from them [in this doc](https://docs.victoriametrics.com/victoriatraces/querying/).

### Alerting

see [these docs](https://docs.victoriametrics.com/victoriatraces/vmalert/).

## Monitoring

VictoriaTraces exposes internal metrics in Prometheus exposition format at `http://<victoria-traces>:10428/metrics` page.
It is recommended to set up monitoring of these metrics via VictoriaMetrics
(see [these docs](https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#how-to-scrape-prometheus-exporters-such-as-node-exporter)),
vmagent (see [these docs](https://docs.victoriametrics.com/victoriametrics/vmagent/#how-to-collect-metrics-in-prometheus-format)) or via Prometheus.

We recommend installing Grafana dashboard for [VictoriaTraces single-node](https://grafana.com/grafana/dashboards/24136) or [cluster](https://grafana.com/grafana/dashboards/24134).

We recommend setting up [alerts](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/rules/alerts-vtraces.yml)
via [vmalert](https://docs.victoriametrics.com/victoriametrics/vmalert/) or via Prometheus.

VictoriaTraces emits its own logs to stdout. It is recommended to investigate these logs during troubleshooting.
