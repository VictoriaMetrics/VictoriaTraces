## vtgen

`vtgen` is a trace data generator. It reads the OTLP request body in `testdata/testdata.bin`, modifies the `trace_id`, `start_time` and `end_time`, and sends them to OTLP trace endpoint (`/v1/traces`).

### Command-line flags
```
-rate
    Spans for each address per second.
    The actual rate will be affected by each address, as the requests are sent synchronously to different addresses in a for-loop, 
    to ensure that each address receive the same pressure. (default: 10000)
-addrs
	OTLP trace export endpoints, split by `,`.
-authorizations
    Basic auth value, split by `,`. If -authorizations is not empty, it must contains same items as -addrs.
```