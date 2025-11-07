---
weight: 4
title: OpenTelemetry setup
disableToc: true
menu:
  docs:
    identifier: victoriatraces-opentelemetry-setup
    parent: "victoriatraces-data-ingestion"
    weight: 4
tags:
  - traces
aliases:
  - /victoriatraces/data-ingestion/OpenTelemetry.html
---

VictoriaTraces supports both client open-telemetry [SDK](https://opentelemetry.io/docs/languages/) and [collector](https://opentelemetry.io/docs/collector/).

## Client SDK

The OpenTelemetry provides detailed document and examples for various programming languages:

- [C++](https://opentelemetry.io/docs/languages/cpp/)
- [C#/.NET](https://opentelemetry.io/docs/languages/dotnet/)
- [Erlang/Elixir](https://opentelemetry.io/docs/languages/erlang/)
- [Go](https://opentelemetry.io/docs/languages/go/)
- [Java](https://opentelemetry.io/docs/languages/java/)
- [JavaScript](https://opentelemetry.io/docs/languages/js/)
- [PHP](https://opentelemetry.io/docs/languages/php/)
- [Python](https://opentelemetry.io/docs/languages/python/)
- [Ruby](https://opentelemetry.io/docs/languages/ruby/)
- [Rust](https://opentelemetry.io/docs/languages/rust/)
- [Swift](https://opentelemetry.io/docs/languages/swift/)

You can send data to VictoriaTraces by HTTP or gRPC endpoints.

### HTTP endpoint 

To send data by HTTP endpoint, specify the `EndpointURL` for http-exporter builder to `http://<victoria-traces>:10428/insert/opentelemetry/v1/traces`.

Consider the following example for Go SDK:

```go
traceExporter, err := otlptracehttp.New(ctx,
  otlptracehttp.WithEndpointURL("http://<victoria-traces>:10428/insert/opentelemetry/v1/traces"),
)
```

### gRPC endpoint

To send the trace data to VictoriaTraces gRPC trace service, you need to first enable the OTLP gRPC server on VictoriaTraces by:
```shell
./victoria-traces -otlpGRPCListenAddr=:4317 -otlpGRPC.tlsCertFile=<cert_file> -otlpGRPC.tlsKeyFile=<key_file>
```

> You can also **disable TLS** for incoming gRPC requests by setting `-otlpGRPC.tls=false`. TLS is recommended for production use, and disabling it should only be done when you're testing or aware of the potential risks.

After that, specify the `Endpoint` for grpc-exporter builder to `<victoria-traces>:4317`, and disable TLS by `WithInsecure()` (Because VictoriaTraces gRPC endpoint doesn't support TLS yet).

Consider the following example for Go SDK:
```go
traceExporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("<victoria-traces>:4317"),
    otlptracegrpc.WithInsecure(),
)
```

VictoriaTraces supports other HTTP headers in both HTTP and gRPC endpoints - see [HTTP headers](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#http-headers).

VictoriaTraces automatically use `service.name` in **resource attributes** and `name` in **span** as [stream fields](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#stream-fields).
While the remaining data (including [resource](https://opentelemetry.io/docs/specs/otel/overview/#resources), [instrumentation scope](https://opentelemetry.io/docs/specs/otel/common/instrumentation-scope/), and fields in [span](https://opentelemetry.io/docs/specs/otel/trace/api/#span), like `trace_id`, `span_id`, span `attributes` and more) are stored as [regular fields](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model):

The ingested trace spans can be queried according to [these docs](https://docs.victoriametrics.com/victoriatraces/querying/).

## Collector configuration

VictoriaTraces supports receiving traces from the following OpenTelemetry collector:

- [OpenTelemetry](#opentelemetry)

### OpenTelemetry

#### HTTP exporter

To send the collected traces to VictoriaTraces HTTP endpoint, specify traces endpoint for [OTLP/HTTP exporter](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlphttpexporter/README.md) in configuration file:

```yaml
exporters:
  otlphttp:
    traces_endpoint: http://<victoria-traces>:10428/insert/opentelemetry/v1/traces
```

VictoriaTraces supports various HTTP headers, which can be used during data ingestion - see the list of [HTTP headers](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#http-headers).
These headers can be passed to OpenTelemetry exporter config via `headers` options. For example, the following configs add (or overwrites) `foo: bar` field to each trace span during data ingestion:

```yaml
exporters:
  otlphttp:
    traces_endpoint: http://<victoria-traces>:10428/insert/opentelemetry/v1/traces
    headers:
      VT-Extra-Fields: foo=bar
```
#### gRPC exporter

To send the collected traces to VictoriaTraces gRPC trace service, you need to first enable the OTLP gRPC server on VictoriaTraces by:
```shell
./victoria-traces -otlpGRPCListenAddr=:4317 -otlpGRPC.tlsCertFile=<cert_file> -otlpGRPC.tlsKeyFile=<key_file>
```

> You can also **disable TLS** for incoming gRPC requests by setting `-otlpGRPC.tls=false`. TLS is recommended for production use, and disabling it should only be done when you're testing or aware of the potential risks.

After that, specify endpoint for [OTLP/gRPC exporter](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/README.md):
```yaml
exporters:
  otlp/with-tls:
    endpoint: <victoria-traces>:4317
    tls:
      cert_file: file.cert
      key_file: file.key
  otlp/without-tls:
    endpoint: <victoria-traces>:4317
    tls:
      insecure: true
```

> Optionally, you can specify the `compression` type to one of the following: `gzip` (default), `snappy`, `zstd`, and `none`.

As same as HTTP endpoint, gRPC also support various HTTP headers. For example, the following configs add (or overwrites) `foo: bar` field to each trace span during data ingestion:
```yaml
exporters:
  otlp/without-tls:
    endpoint: <victoria-traces>:4317
    tls:
      insecure: true
    headers:
      VT-Extra-Fields: foo=bar
```

See also:

- [Data ingestion troubleshooting](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#troubleshooting).
- [How to query VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/querying/).
- [Docker-compose demo for HotROD application integration with VictoriaTraces](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/compose-vt-single.yml).
