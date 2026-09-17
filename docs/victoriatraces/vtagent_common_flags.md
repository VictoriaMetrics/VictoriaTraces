---
build:
  list: never
  publishResources: false
  render: never
sitemap:
  disable: true
---
<!-- The file has to be manually updated during feature work in PR, "TAG=... make docs-update-flags" command could be used periodically to ensure the flags in sync with the given release TAG. -->
```shellhelp

vtagent collects trace spans via popular data ingestion protocols and routes it to VictoriaTraces.

See the docs at https://docs.victoriametrics.com/victoriatraces/vtagent/ .

  -blockcache.missesBeforeCaching int
     The number of cache misses before putting the block into cache. Higher values may reduce indexdb/dataBlocks cache size at the cost of higher CPU and disk read usage (default 2)
  -defaultMsgValue string
     Default value for _msg field; see https://docs.victoriametrics.com/victorialogs/keyconcepts/#message-field (default "-")
  -enableTCP6
     Whether to enable IPv6 for listening and dialing. By default, only IPv4 TCP and UDP are used
  -envflag.enable
     Whether to enable reading flags from environment variables in addition to the command line. Command line flag values have priority over values from environment vars. Flags are read only from the command line if this flag isn't set. See https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#environment-variables for more details
  -envflag.prefix string
     Prefix for environment variables if -envflag.enable is set
  -filestream.disableFadvise
     Whether to disable fadvise() syscall when reading large data files. The fadvise() syscall prevents from eviction of recently accessed data from OS page cache during background merges and backups. In some rare cases it is better to disable the syscall if it uses too much CPU
  -flagsAuthKey value
     Auth key for /flags endpoint. It must be passed via authKey query arg. It overrides -httpAuth.*
     Flag value can be read from the given file when using -flagsAuthKey=file:///abs/path/to/file or -flagsAuthKey=file://./relative/path/to/file.
     Flag value can be read from the given http/https url when using -flagsAuthKey=http://host/path or -flagsAuthKey=https://host/path
  -fs.disableMincore
     Whether to disable the mincore() syscall for checking mmap()ed files. By default, mincore() is used to detect whether mmap()ed file pages are resident in memory. Disabling mincore() may be needed on older ZFS filesystems (below 2.1.5), since it may trigger ZFS bug. See https://github.com/VictoriaMetrics/VictoriaMetrics/issues/10327 for details.
  -fs.disableMmap
     Whether to use pread() instead of mmap() for reading data files. By default, mmap() is used for 64-bit arches and pread() is used for 32-bit arches, since they cannot read data files bigger than 2^32 bytes in memory. mmap() is usually faster for reading small data chunks than pread()
  -fs.maxConcurrency int
     The maximum number of concurrent goroutines to work with files; smaller values may help reducing Go scheduling latency on systems with small number of CPU cores; higher values may help reducing data ingestion latency on systems with high-latency storage such as NFS or Ceph (default 16x CPU cores, default capped at 256)
  -http.connTimeout duration
     Incoming connections to -httpListenAddr are closed after the configured timeout. This may help evenly spreading load among a cluster of services behind TCP-level load balancer. Zero value disables closing of incoming connections (default 2m0s)
  -http.disableCORS
     Disable CORS for all origins (*)
  -http.disableKeepAlive
     Whether to disable HTTP keep-alive for incoming connections at -httpListenAddr
  -http.disableResponseCompression
     Disable compression of HTTP responses to save CPU resources. By default, compression is enabled to save network bandwidth
  -http.header.csp string
     Value for 'Content-Security-Policy' header, recommended: "default-src 'self'"
  -http.header.disableServerHostname
     Whether to disable 'X-Server-Hostname' header in HTTP responses
  -http.header.frameOptions string
     Value for 'X-Frame-Options' header
  -http.header.hsts string
     Value for 'Strict-Transport-Security' header, recommended: 'max-age=31536000; includeSubDomains'
  -http.idleConnTimeout duration
     Timeout for incoming idle http connections (default 1m0s)
  -http.maxGracefulShutdownDuration duration
     The maximum duration for a graceful shutdown of the HTTP server. During this period the server stops accepting new connections, but it will continue serving existing connections. The remaining in-flight requests are canceled before the deadline, so the shutdown can finish within this duration. A highly loaded server may require increased value for a graceful shutdown (default 7s)
  -http.pathPrefix string
     An optional prefix to add to all the paths handled by http server. For example, if '-http.pathPrefix=/foo/bar' is set, then all the http requests will be handled on '/foo/bar/*' paths. This may be useful for proxied requests. See https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus
  -http.shutdownDelay duration
     Optional delay before http server shutdown. During this delay, the server returns non-OK responses from /health page, so load balancers can route new requests to other servers
  -httpAuth.password value
     Password for HTTP server's Basic Auth. The authentication is disabled if -httpAuth.username is empty
     Flag value can be read from the given file when using -httpAuth.password=file:///abs/path/to/file or -httpAuth.password=file://./relative/path/to/file.
     Flag value can be read from the given http/https url when using -httpAuth.password=http://host/path or -httpAuth.password=https://host/path
  -httpAuth.username string
     Username for HTTP server's Basic Auth. The authentication is disabled if empty. See also -httpAuth.password
  -httpListenAddr array
     TCP address to listen for incoming http requests. Set this flag to empty value in order to disable listening on any port. This mode may be useful for running multiple vtagent instances on the same server. Note that /targets and /metrics pages aren't available if -httpListenAddr=''. See also -tls and -httpListenAddr.useProxyProtocol
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -httpListenAddr.useProxyProtocol array
     Whether to use proxy protocol for connections accepted at the corresponding -httpListenAddr . See https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt . With enabled proxy protocol http server cannot serve regular /metrics endpoint. Use -pushmetrics.url for metrics pushing
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to false.
  -insert.disable
     Whether to disable both /insert/* and /internal/insert HTTP endpoints and OpenTelemetry gRPC ingestion endpoint. Useful for dedicated vtselect nodes. See also -internalinsert.disable. See https://docs.victoriametrics.com/victoriatraces/cluster/#security
  -insert.indexFlushInterval duration
     Amount of time after which the index of a trace is flushed. VictoriaTraces creates an index for each trace ID based on its start and end times.Each trace ID must wait in the queue for -insert.indexFlushInterval, continuously updating its start and end times before being flushed into the index. (default 20s)
  -insert.maxFieldsPerLine int
     The maximum number of log fields per line, which can be read by /insert/* handlers; see https://docs.victoriametrics.com/victorialogs/faq/#how-many-fields-a-single-log-entry-may-contain (default 1000)
  -insert.maxQueueDuration duration
     The maximum duration to wait in the queue when -maxConcurrentInserts concurrent insert requests are executed (default 1m0s)
  -internStringCacheExpireDuration duration
     The expiry duration for caches for interned strings. See https://en.wikipedia.org/wiki/String_interning . See also -internStringMaxLen and -internStringDisableCache (default 6m0s)
  -internStringDisableCache
     Whether to disable caches for interned strings. This may reduce memory usage at the cost of higher CPU usage. See https://en.wikipedia.org/wiki/String_interning . See also -internStringCacheExpireDuration and -internStringMaxLen
  -internStringMaxLen int
     The maximum length for strings to intern. A lower limit may save memory at the cost of higher CPU usage. See https://en.wikipedia.org/wiki/String_interning . See also -internStringDisableCache and -internStringCacheExpireDuration (default 500)
  -internalinsert.disable
     Whether to disable /internal/insert HTTP endpoint. See also -insert.disable. See https://docs.victoriametrics.com/victoriatraces/cluster/#security
  -internalinsert.maxRequestSize size
     The maximum size in bytes of a single request, which can be accepted at /internal/insert HTTP endpoint
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB (default 67108864)
  -loggerDisableTimestamps
     Whether to disable writing timestamps in logs
  -loggerErrorsPerSecondLimit int
     Per-second limit on the number of ERROR messages. If more than the given number of errors are emitted per second, the remaining errors are suppressed. Zero values disable the rate limit
  -loggerFormat string
     Format for logs. Possible values: default, json (default "default")
  -loggerJSONFields string
     Allows renaming fields in JSON formatted logs. Example: "ts:timestamp,msg:message" renames "ts" to "timestamp" and "msg" to "message". Supported fields: ts, level, caller, msg
  -loggerLevel string
     Minimum level of errors to log. Possible values: INFO, WARN, ERROR, FATAL, PANIC (default "INFO")
  -loggerMaxArgLen int
     The maximum length of a single logged argument. Longer arguments are replaced with 'arg_start..arg_end', where 'arg_start' and 'arg_end' is prefix and suffix of the arg with the length not exceeding -loggerMaxArgLen / 2 (default 5000)
  -loggerOutput string
     Output for the logs. Supported values: stderr, stdout (default "stderr")
  -loggerTimezone string
     Timezone to use for timestamps in logs. Timezone must be a valid IANA Time Zone. For example: America/New_York, Europe/Berlin, Etc/GMT+3 or Local (default "UTC")
  -loggerWarnsPerSecondLimit int
     Per-second limit on the number of WARN messages. If more than the given number of warns are emitted per second, then the remaining warns are suppressed. Zero values disable the rate limit
  -maxConcurrentInserts int
     The maximum number of concurrent insert requests. Set higher value when clients send data over slow networks. Default value depends on the number of available CPU cores. It should work fine in most cases since it minimizes resource usage. See also -insert.maxQueueDuration (default 2x CPU cores)
  -memory.allowedBytes size
     Allowed size of system memory VictoriaMetrics caches may occupy. This option overrides -memory.allowedPercent if set to a non-zero value. Too low a value may increase the cache miss rate usually resulting in higher CPU and disk IO usage. Too high a value may evict too much data from the OS page cache resulting in higher disk IO usage. The process may behave unexpectedly if this flag is set too small (e.g., 1 byte).
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB (default 0)
  -memory.allowedPercent float
     Allowed percent of system memory VictoriaMetrics caches may occupy. See also -memory.allowedBytes. Too low a value may increase cache miss rate usually resulting in higher CPU and disk IO usage. Too high a value may evict too much data from the OS page cache which will result in higher disk IO usage (default 60)
  -metrics.exposeMetadata
     Whether to expose TYPE and HELP metadata at the /metrics page, which is exposed at -httpListenAddr . The metadata may be needed when the /metrics page is consumed by systems, which require this information. For example, Managed Prometheus in Google Cloud - https://cloud.google.com/stackdriver/docs/managed-prometheus/troubleshooting#missing-metric-type
  -metricsAuthKey value
     Auth key for /metrics endpoint. It must be passed via authKey query arg. It overrides -httpAuth.*
     Flag value can be read from the given file when using -metricsAuthKey=file:///abs/path/to/file or -metricsAuthKey=file://./relative/path/to/file.
     Flag value can be read from the given http/https url when using -metricsAuthKey=http://host/path or -metricsAuthKey=https://host/path
  -nativeinsert.maxRequestSize size
     The maximum size in bytes of a single request, which can be accepted at /insert/native and /insert/multitenant/native HTTP endpoints
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB (default 67108864)
  -opentelemetry.traces.maxRequestSize size
     The maximum size in bytes of a single OpenTelemetry trace export request.
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB (default 67108864)
  -otlpGRPC.tls
     Enable TLS for incoming gRPC request at the given -otlpGRPCListenAddr. It's set to true by default, and -otlpGRPC.tlsCertFile and -otlpGRPC.tlsKeyFile must be set. It could be configured to false to allow insecure connection. (default true)
  -otlpGRPC.tlsCertFile string
     Path to file with TLS certificate for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. Prefer ECDSA certs instead of RSA certs as RSA certs are slower. The provided certificate file is automatically re-read every second, so it can be dynamically updated.
  -otlpGRPC.tlsCipherSuites array
     Optional TLS cipher suites for incoming requests over HTTPS if -otlpGRPC.tls is not set to false. See the list of supported cipher suites at https://pkg.go.dev/crypto/tls#pkg-constants
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -otlpGRPC.tlsKeyFile string
     Path to file with TLS key for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. The provided key file is automatically re-read every second, so it can be dynamically updated.
  -otlpGRPC.tlsMinVersion string
     Optional minimum TLS version to use for the corresponding -otlpGRPCListenAddr if -otlpGRPC.tls is not set to false. Supported values: TLS10, TLS11, TLS12, TLS13.
  -otlpGRPCListenAddr string
     TCP address for accepting OTLP gRPC requests. Defaults to empty, which means it is disabled. The recommended port is ":4317".
  -pprofAuthKey value
     Auth key for /debug/pprof/* endpoints. It must be passed via authKey query arg. It overrides -httpAuth.*
     Flag value can be read from the given file when using -pprofAuthKey=file:///abs/path/to/file or -pprofAuthKey=file://./relative/path/to/file.
     Flag value can be read from the given http/https url when using -pprofAuthKey=http://host/path or -pprofAuthKey=https://host/path
  -pushmetrics.disableCompression
     Whether to disable request body compression when pushing metrics to every -pushmetrics.url
  -pushmetrics.extraLabel array
     Optional labels to add to metrics pushed to every -pushmetrics.url . For example, -pushmetrics.extraLabel='instance="foo"' adds instance="foo" label to all the metrics pushed to every -pushmetrics.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -pushmetrics.header array
     Optional HTTP request header to send to every -pushmetrics.url . For example, -pushmetrics.header='Authorization: Basic foobar' adds 'Authorization: Basic foobar' header to every request to every -pushmetrics.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -pushmetrics.interval duration
     Interval for pushing metrics to every -pushmetrics.url (default 10s)
  -pushmetrics.url array
     Optional URL to push metrics exposed at /metrics page. See https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#push-metrics . By default, metrics exposed at /metrics page aren't pushed to any remote storage
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.basicAuth.password array
     Optional basic auth password to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.basicAuth.passwordFile array
     Optional path to basic auth password to use for the corresponding -remoteWrite.url. The file is re-read every second
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.basicAuth.username array
     Optional basic auth username to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.basicAuth.usernameFile array
     Optional path to basic auth username to use for the corresponding -remoteWrite.url. The file is re-read every second
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.bearerToken array
     Optional bearer auth token to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.bearerTokenFile array
     Optional path to bearer token file to use for the corresponding -remoteWrite.url. The token is re-read from the file every second
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.flushInterval duration
     Interval for flushing the data to remote storage. This option takes effect only when less than 2MB of data per second are pushed to -remoteWrite.url (default 1s)
  -remoteWrite.format array
     The data format to use for sending data to the corresponding -remoteWrite.url. Available formats: native, jsonline. Default is native. See https://docs.victoriametrics.com/victoriatraces/vtagent/#remote-write-format
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.headers array
     Optional HTTP headers to send with each request to the corresponding -remoteWrite.url. For example, -remoteWrite.headers='My-Auth:foobar' would send 'My-Auth: foobar' HTTP header with every request to the corresponding -remoteWrite.url. Multiple headers must be delimited by '^^': -remoteWrite.headers='header1:value1^^header2:value2'
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.maxBlockSize size
     The maximum block size to send to remote storage. Bigger blocks may improve performance at the cost of the increased memory usage.
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB (default 8388608)
  -remoteWrite.maxDiskUsagePerURL array
     The maximum file-based buffer size in bytes at -remoteWrite.tmpDataPath for each -remoteWrite.url. When buffer size reaches the configured maximum, then old data is dropped when adding new data to the buffer. Buffered data is stored in ~500MB chunks. It is recommended to set the value for this flag to a multiple of the block size 500MB. Disk usage is unlimited if the value is set to 0
     Supports the following optional suffixes for size values: KB, MB, GB, TB, KiB, MiB, GiB, TiB. (default 0)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.oauth2.clientID array
     Optional OAuth2 clientID to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.oauth2.clientSecret array
     Optional OAuth2 clientSecret to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.oauth2.clientSecretFile array
     Optional OAuth2 clientSecretFile to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.oauth2.endpointParams array
     Optional OAuth2 endpoint parameters to use for the corresponding -remoteWrite.url . The endpoint parameters must be set in JSON format: {"param1":"value1",...,"paramN":"valueN"}
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.oauth2.scopes array
     Optional OAuth2 scopes to use for the corresponding -remoteWrite.url. Scopes must be delimited by ';'
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.oauth2.tokenUrl array
     Optional OAuth2 tokenURL to use for the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.proxyURL array
     Optional proxy URL for writing data to the corresponding -remoteWrite.url. Supported proxies: http, https, socks5. Example: -remoteWrite.proxyURL=socks5://proxy:1234
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.queues int
     The number of concurrent queues to each -remoteWrite.url. Set more queues if default number of queues isn't enough for sending high volume of collected data to remote storage. Default value depends on the number of available CPU cores. It should work fine in most cases since it minimizes resource usage (default 2x CPU cores)
  -remoteWrite.rateLimit array
     Optional rate limit in bytes per second for data sent to the corresponding -remoteWrite.url. By default, the rate limit is disabled. It can be useful for limiting load on remote storage when big amounts of buffered data  (default 0)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.retryMaxTime array
     The max time spent on retry attempts to send a block of data to the corresponding -remoteWrite.url. Change this value if it is expected for -remoteWrite.url to be unreachable for more than -remoteWrite.retryMaxTime. See also -remoteWrite.retryMinInterval (default 1m0s)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.retryMinInterval array
     The minimum delay between retry attempts to send a block of data to the corresponding -remoteWrite.url. Every next retry attempt will double the delay to prevent hammering of remote database. See also -remoteWrite.retryMaxTime (default 1s)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.sendTimeout array
     Timeout for sending a single block of data to the corresponding -remoteWrite.url (default 1m0s)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.showURL
     Whether to show -remoteWrite.url in the exported metrics. It is hidden by default, since it can contain sensitive info such as auth key
  -remoteWrite.tlsCAFile array
     Optional path to TLS CA file to use for verifying connections to the corresponding -remoteWrite.url. By default, system CA is used
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.tlsCertFile array
     Optional path to client-side TLS certificate file to use when connecting to the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.tlsHandshakeTimeout array
     The timeout for establishing tls connections to the corresponding -remoteWrite.url (default 20s)
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to default value.
  -remoteWrite.tlsInsecureSkipVerify array
     Whether to skip tls verification when connecting to the corresponding -remoteWrite.url
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to false.
  -remoteWrite.tlsKeyFile array
     Optional path to client-side TLS certificate key to use when connecting to the corresponding -remoteWrite.url
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.tlsServerName array
     Optional TLS server name to use for connections to the corresponding -remoteWrite.url. By default, the server name from -remoteWrite.url is used
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -remoteWrite.tmpDataPath string
     Path to directory for storing pending data, which isn't sent to the configured -remoteWrite.url. If this flag isn't set, then pending data is stored in the vtagent-remotewrite-data subdirectory under the -tmpDataPath directory; see also -remoteWrite.maxDiskUsagePerURL
  -remoteWrite.url array
     Remote storage URL to write data to. Example url: http://<victoriatraces-host>:10428/insert/native. Pass multiple -remoteWrite.url options in order to replicate the collected data to multiple remote storage systems. See also -remoteWrite.maxDiskUsagePerURL and -remoteWrite.format
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -secret.flags array
     Comma-separated list of flag names with secret values. Values for these flags are hidden in logs and on /metrics page
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -tls array
     Whether to enable TLS for incoming HTTP requests at the given -httpListenAddr (aka https). -tlsCertFile and -tlsKeyFile must be set if -tls is set. See also -mtls
     Supports array of values separated by comma or specified via multiple flags.
     Empty values are set to false.
  -tlsCertFile array
     Path to file with TLS certificate for the corresponding -httpListenAddr if -tls is set. Prefer ECDSA certs instead of RSA certs as RSA certs are slower. The provided certificate file is automatically re-read every second, so it can be dynamically updated. See also -tlsAutocertHosts
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -tlsCipherSuites array
     Optional list of TLS cipher suites for incoming requests over HTTPS if -tls is set. See the list of supported cipher suites at https://pkg.go.dev/crypto/tls#pkg-constants
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -tlsKeyFile array
     Path to file with TLS key for the corresponding -httpListenAddr if -tls is set. The provided key file is automatically re-read every second, so it can be dynamically updated. See also -tlsAutocertHosts
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -tlsMinVersion array
     Optional minimum TLS version to use for the corresponding -httpListenAddr if -tls is set. Supported values: TLS10, TLS11, TLS12, TLS13
     Supports an array of values separated by comma or specified via multiple flags.
     Each array item can contain comma inside single-quoted or double-quoted string, {}, [] and () braces.
  -tmpDataPath string
     Base directory for storing vtagent data. Used as default for -remoteWrite.tmpDataPath unless that flag is set explicitly
  -version
     Show VictoriaMetrics version
```
