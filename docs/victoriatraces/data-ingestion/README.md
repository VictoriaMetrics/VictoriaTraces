---
build:
  list: never
  publishResources: false
  render: never
sitemap:
  disable: true
---

[VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) can accept trace spans via [the OpenTelemetry protocol (OTLP)](https://opentelemetry.io/docs/specs/otlp/).

## HTTP APIs

### Opentelemetry API

VictoriaTraces provides the following API for OpenTelemetry data ingestion:

- `/insert/opentelemetry/v1/traces`

See more details in [OpenTelemetry data ingestion](https://docs.victoriametrics.com/victoriatraces/data-ingestion/opentelemetry/).

### HTTP parameters

VictoriaTraces accepts optional HTTP parameters at data ingestion HTTP API via [HTTP query string parameters](https://en.wikipedia.org/wiki/Query_string), or via [HTTP headers](https://en.wikipedia.org/wiki/List_of_HTTP_header_fields).

HTTP query string parameters have priority over HTTP Headers.

#### HTTP Query string parameters

All the [HTTP-based data ingestion protocols](#http-apis) support the following [HTTP query string](https://en.wikipedia.org/wiki/Query_string) args:

- `extra_fields` - an optional comma-separated list of [trace fields](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model),
  which must be added to all the ingested traces. The format of every `extra_fields` entry is `field_name=field_value`.
  If the trace entry contains fields from the `extra_fields`, then they are overwritten by the values specified in `extra_fields`.

- `debug` - if this arg is set to `1`, then the ingested traces aren't stored in VictoriaTraces. Instead,
  the ingested data is logged by VictoriaTraces, so it can be investigated later.

See also [HTTP headers](#http-headers).

#### HTTP headers

All the [HTTP-based data ingestion protocols](#http-apis) support the following [HTTP Headers](https://en.wikipedia.org/wiki/List_of_HTTP_header_fields)
additionally to [HTTP query args](#http-query-string-parameters):

- `AccountID` - accountID of the tenant to ingest data to. See [multitenancy docs](https://docs.victoriametrics.com/victoriatraces/#multitenancy) for details.

- `ProjectID`- projectID of the tenant to ingest data to. See [multitenancy docs](https://docs.victoriametrics.com/victoriatraces/#multitenancy) for details.

- `VT-Extra-Fields` - an optional comma-separated list of [trace fields](https://docs.victoriametrics.com/victoriatraces/keyconcepts/#data-model),
  which must be added to all the ingested traces. The format of every `extra_fields` entry is `field_name=field_value`.
  If the trace entry contains fields from the `extra_fields`, then they are overwritten by the values specified in `extra_fields`.

- `VT-Debug` - if this parameter is set to `1`, then the ingested traces aren't stored in VictoriaTraces. Instead,
  the ingested data is logged by VictoriaTraces, so it can be investigated later.

See also [HTTP Query string parameters](#http-query-string-parameters).

## gRPC Services and Methods

### OpenTelemetry Collector TraceService

VictoriaTraces implements the OpenTelemetry Collector [TraceService](https://github.com/open-telemetry/opentelemetry-proto/blob/v1.8.0/opentelemetry/proto/collector/trace/v1/trace_service.proto#L30)
to accept spans pushed by applications or collectors in [OTLP/gRPC](https://opentelemetry.io/docs/specs/otlp/#otlpgrpc).

As gRPC is running over HTTP2, it can also accept optional HTTP parameters via [HTTP headers](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#http-headers)

See more details in [OpenTelemetry data ingestion](https://docs.victoriametrics.com/victoriatraces/data-ingestion/opentelemetry/#grpc-exporter).
