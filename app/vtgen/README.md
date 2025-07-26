## vtgen

`vtgen` is a trace data generator. It reads the OTLP request body in `testdata/testdata.bin`, modifies the `trace_id`, `start_time` and `end_time`, and sends them to OTLP trace endpoint (`/v1/traces`).

### Usage
`vtgen` can be used for:
1. Generating identical data for different targets, which is useful for query performance benchmarking of different storage backends against the same dataset.
2. Generating different data at the same rate (on a best-effort basis) for different targets, which is useful for data ingestion performance benchmarking of different storage backends.

You can build `vtgen` via the following commands:
```
# build via go build
make vtgen
# build via Docker
make vtgen-prod
```

These commands will generate `vtgen(-prod)` binary in `bin` folder.

`vtgen` MUST be run from the root path of `VictoriaTraces` repository, to load the test data correctly from `app/vtgen/testdata`.

To send identical data to different targets:
```
./bin/vtgen -addrs=http://example-url1:port/v1/traces,http://example-url2:port/insert/opentelemetry/v1/traces
```
The performance of different targets will affect each other, as `vtgen` generates data and makes HTTP requests to them one by one.

To send (potentially) different data to different addresses at the same rate, simply run multiple `vtgen` with different HTTP listening port:
```
./bin/vtgen -addrs=http://example-url1:port/v1/traces -httpListenAddr=0.0.0.0:8080
./bin/vtgen -addrs=http://example-url2:port/insert/opentelemetry/v1/traces -httpListenAddr=0.0.0.0:8081
```

### Metrics
`vtgen` exposes some metrics to help tracking the ingestion performance. Here's the example:
```
vt_gen_http_request_duration_seconds_bucket{path="http://example-url1:port/v1/traces",vmrange="7.743e-05...8.799e-05"} 8
vt_gen_http_request_duration_seconds_bucket{path="http://example-url1:port/v1/traces",vmrange="3.162e-03...3.594e-03"} 1
vt_gen_http_request_duration_seconds_sum{path="http://example-url1:port/v1/traces"} 0.17367758299999986
vt_gen_http_request_duration_seconds_count{path="http://example-url1:port/v1/traces"} 667
vt_gen_http_request_error_count{path="http://example-url1:port/v1/traces"} 667
```

- `vt_gen_http_request_duration_seconds_bucket/_sum/_count` is a `histogram` for each address.
- `vt_gen_http_request_error_count` is a `counter` for each address.

### Command-line flags
```
  -addrs string
        otlp trace export endpoints, split by ",".
  -authorizations string
        authorization header for each -addrs, split by ",".
  -httpListenAddr string
        http listen address for pprof and metrics. (default "0.0.0.0:8080")
  -logEvery10k int
        how many trace id should be logged for every 10000 traces by each worker. (default 2)
  -rate int
        spans per second. (default 10000)
  -worker int
        number of workers. (default 4)
```