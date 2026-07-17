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

There are two ways to get started with VictoriaTraces:

* [Try it locally](https://docs.victoriametrics.com/victoriatraces/quick-start/#try-it-locally) - if you just want to see how VictoriaTraces works,
  go with the single binary: download it, start it with one command, ingest a trace span
  and explore it in the built-in VMUI in a couple of minutes. No Docker, no configuration files
  and no extra components are required;
* [Install it](https://docs.victoriametrics.com/victoriatraces/quick-start/#how-to-install) - if you want to set up VictoriaTraces for real use,
  pick a deployment model (single-node or cluster) and an installation method
  (Docker, Helm charts, Kubernetes operator or binary releases).

If you'd rather not install anything at all, visit the [VictoriaTraces playground](https://play-vtraces.victoriametrics.com/)
to see how trace spans are structured and stored.

Whichever way you choose, you may also find interesting the other sections of this page,
like how to [write](https://docs.victoriametrics.com/victoriatraces/quick-start/#write-data)
and [read](https://docs.victoriametrics.com/victoriatraces/quick-start/#read-data) data,
[alerting](https://docs.victoriametrics.com/victoriatraces/quick-start/#alerting)
and [monitoring](https://docs.victoriametrics.com/victoriatraces/quick-start/#monitoring) of VictoriaTraces itself.

## Try it locally

The fastest way to try VictoriaTraces on your own machine is its binary - the only thing needed to run it.

### Step 1: Download the binary

Create a directory for this test drive, so all the files created along the way stay in one place:

```sh
mkdir vt-quick-start && cd vt-quick-start
```

Download the `victoria-traces-<os>-<arch>-<version>.tar.gz` archive for your OS and architecture
from the [releases page](https://github.com/VictoriaMetrics/VictoriaTraces/releases/latest)
and unpack it. It contains a single `victoria-traces-prod` binary.

For example, on Linux with `amd64` architecture:

```sh
curl -L -O https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.9.4/victoria-traces-linux-amd64-v0.9.4.tar.gz
tar xzf victoria-traces-linux-amd64-v0.9.4.tar.gz
```

The binary is self-contained and requires no installation - it is ready to run as is.

### Step 2: Start VictoriaTraces

Starting VictoriaTraces is as simple as executing the binary, with no arguments at all:

```sh
./victoria-traces-prod
```

VictoriaTraces prints a couple of dozen log lines on start, describing the storage, caches
and memory limits it sets up. Look for these two lines confirming that it is up and running:

```sh
2026-07-16T10:08:58.485Z    info    app/victoria-traces/main.go:45    starting VictoriaTraces at "[:10428]"...
...
2026-07-16T10:08:58.486Z    info    lib/httpserver/httpserver.go:148    started server at http://0.0.0.0:10428/
```

That's it - VictoriaTraces is running, listening on port `10428` and ready to accept trace spans.
If you list the `vt-quick-start` directory, you can see a new `victoria-traces-data` directory
created next to the binary - this is where the ingested trace spans are stored.

There are no trace spans stored yet, so let's ingest one.

### Step 3: Ingest a trace span

VictoriaTraces accepts trace spans via [the OpenTelemetry protocol (OTLP)](https://opentelemetry.io/docs/specs/otlp/).
For example, insert an example span with a plain `curl` command:

```sh
curl -X POST -H 'Content-Type: application/json' http://localhost:10428/insert/opentelemetry/v1/traces -d '
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "frontend-web"}},
        {"key": "telemetry.sdk.language", "value": {"stringValue": "webjs"}},
        {"key": "telemetry.sdk.name", "value": {"stringValue": "opentelemetry"}},
        {"key": "telemetry.sdk.version", "value": {"stringValue": "1.30.1"}},
        {"key": "process.runtime.name", "value": {"stringValue": "browser"}},
        {"key": "process.runtime.description", "value": {"stringValue": "Web Browser"}},
        {"key": "process.runtime.version", "value": {"stringValue": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}
      ]
    },
    "scopeSpans": [{
      "scope": {"name": "@opentelemetry/instrumentation-document-load", "version": "0.44.1"},
      "spans": [{
        "traceId": "1af5dd013a30efe7f2970032ab81958b",
        "spanId": "229d083a6c480511",
        "parentSpanId": "",
        "name": "documentLoad",
        "kind": 1,
        "startTimeUnixNano": "'$(date +%s000000000)'",
        "endTimeUnixNano": "'$(date +%s000000000)'",
        "attributes": [
          {"key": "session.id", "value": {"stringValue": "96e702c3-6f05-4f54-b2b3-2fad2b7b7995"}},
          {"key": "http.url", "value": {"stringValue": "http://frontend-proxy:8080/cart"}},
          {"key": "http.user_agent", "value": {"stringValue": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}
        ],
        "events": [{"timeUnixNano": "1757320936519100098", "name": "fetchStart"}],
        "status": {}
      }]
    }]
  }]
}'
```

The `$(date +%s000000000)` substitution stamps the span with the current time before sending it, so it shows up in the recent query results.

Now that a trace span is stored, it's time to look at it.

### Step 4: Explore the traces

Open [http://localhost:10428/select/vmui](http://localhost:10428/select/vmui) in your browser to access
the built-in VMUI for browsing trace data. You should see the `documentLoad` span
of the `frontend-web` service ingested at the previous step.

VictoriaTraces also provides [Jaeger Query Service JSON APIs](https://www.jaegertracing.io/docs/2.6/apis/#internal-http-json)
for programmatic access and for [Grafana integration](https://docs.victoriametrics.com/victoriatraces/quick-start/#read-data).
For example, list the services with ingested spans:

```sh
curl http://localhost:10428/select/jaeger/api/services
```

The command should return the `frontend-web` service the example span belongs to.

So far the only stored span is the hand-made one from the previous step - let's collect something more interesting.

### Step 5 (optional): Generate realistic traces

[HotROD](https://github.com/jaegertracing/jaeger/tree/main/examples/hotrod) is a sample "ride-sharing" application
instrumented with OpenTelemetry, which makes it a convenient source of realistic traces. This step requires Docker:

```sh
docker run \
  -p8080-8083:8080-8083 \
  --rm \
  --env OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://<host-ip>:10428/insert/opentelemetry/v1/traces \
  jaegertracing/example-hotrod:latest \
  all
```

Replace `<host-ip>` with the IP address of your machine (see the `ifconfig`/`ipconfig` output),
so VictoriaTraces is accessible from inside the HotROD container.

Open [http://127.0.0.1:8080/](http://127.0.0.1:8080/) and click any button to request rides and generate traces.
Then explore them in [VMUI](http://localhost:10428/select/vmui) - this time with multiple services
and multi-span traces showing how the request flows through the application.

Once you're done experimenting, tidying everything up takes a single command.

### Cleanup

Stop VictoriaTraces with `Ctrl+C`. All the ingested trace spans live in the `victoria-traces-data` directory -
delete it if you want to start from scratch. To remove everything created by this test drive,
delete the whole `vt-quick-start` directory created at [step 1](https://docs.victoriametrics.com/victoriatraces/quick-start/#step-1-download-the-binary).

This test drive only scratches the surface of what VictoriaTraces can do. Ready for a real setup?
Continue with the installation options below.

## How to install

VictoriaTraces can be deployed as:

- VictoriaTraces Single-node - all-in-one binary that is easy to run and maintain.
- [VictoriaTraces Cluster](https://docs.victoriametrics.com/victoriatraces/cluster/) - same binary, deployed as separate roles within a cluster: `vtinsert`, `vtselect`, and `vtstorage`.

VictoriaTraces is available as:

- docker images at [Docker Hub](https://hub.docker.com/r/victoriametrics/victoria-traces) and [Quay](https://quay.io/repository/victoriametrics/victoria-traces).
- [Binary releases](https://github.com/VictoriaMetrics/VictoriaTraces/releases/) - see how to start [a single node](#starting-victoriatraces-single-node-from-a-binary) or [a cluster](#starting-victoriatraces-cluster-from-binaries) from binaries
- [Helm charts](#helm-charts) and custom resources with [VictoriaMetrics Operator](#victoriaMetrics-operator)
- [Source code](https://github.com/VictoriaMetrics/VictoriaTraces). See [How to build from sources](https://docs.victoriametrics.com/victoriatraces/#how-to-build-from-sources)

### Starting VictoriaTraces Single Node via Docker

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

After VictoriaTraces is running, verify VMUI is working by going to `http://<victoria-traces>:10428/select/vmui`.

See how to [write](#write-data) or [read](#read-data) from VictoriaTraces.

### Starting VictoriaTraces Single Node from a Binary

1. Download the archive for your OS and architecture from the [releases page](https://github.com/VictoriaMetrics/VictoriaTraces/releases/latest).
For example, on Linux with `amd64` architecture:

```sh
curl -L -O https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.9.4/victoria-traces-linux-amd64-v0.9.4.tar.gz
```

2. Extract the archive to /usr/local/bin by running:

```sh
sudo tar xzf victoria-traces-linux-amd64-v0.9.4.tar.gz -C /usr/local/bin
```

3. Create a VictoriaTraces user on the system:

```sh
sudo useradd -s /usr/sbin/nologin victoriatraces
```

4. Create a folder for storing VictoriaTraces data:

```sh
sudo mkdir -p /var/lib/victoria-traces && sudo chown -R victoriatraces:victoriatraces /var/lib/victoria-traces
```

5. Create a Linux Service by running the following:

```sh
sudo bash -c 'cat <<END >/etc/systemd/system/victoriatraces.service
[Unit]
Description=VictoriaTraces service
After=network.target

[Service]
Type=simple
User=victoriatraces
Group=victoriatraces
ExecStart=/usr/local/bin/victoria-traces-prod -storageDataPath=/var/lib/victoria-traces
SyslogIdentifier=victoriatraces
Restart=always

PrivateTmp=yes
ProtectHome=yes
NoNewPrivileges=yes

ProtectSystem=full

[Install]
WantedBy=multi-user.target
END'
```

Extra [command-line flags](https://docs.victoriametrics.com/victoriatraces/#list-of-command-line-flags) can be added to the `ExecStart` line.

> Please note, `victoriatraces` service is listening on `:10428` for HTTP connections (see `-httpListenAddr` flag).

6. Start and enable the service by running the following command:

```sh
sudo systemctl daemon-reload && sudo systemctl enable --now victoriatraces.service
```

7. Check that the service started successfully:

```sh
sudo systemctl status victoriatraces.service
```

8. After VictoriaTraces is in `Running` state, verify the [Web UI](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui) is working
by going to `http://<ip_or_hostname>:10428/select/vmui`.

### Starting VictoriaTraces Cluster from Binaries

VictoriaTraces cluster consists of [3 components](https://docs.victoriametrics.com/victoriatraces/cluster/#architecture) -
`vtinsert`, `vtselect` and `vtstorage`. All of them share the same single-node VictoriaTraces executable,
and the role of every node is defined by its command-line flags.
It is recommended to run these components in the same private network (for [security reasons](https://docs.victoriametrics.com/victoriatraces/#security)),
but on separate physical nodes for the best performance.

On all nodes, you will need to do the following:

1. Download the archive for your OS and architecture from the [releases page](https://github.com/VictoriaMetrics/VictoriaTraces/releases/latest).
For example, on Linux with `amd64` architecture:

```sh
curl -L -O https://github.com/VictoriaMetrics/VictoriaTraces/releases/download/v0.9.4/victoria-traces-linux-amd64-v0.9.4.tar.gz
```

2. Extract the archive to /usr/local/bin by running:

```sh
sudo tar xzf victoria-traces-linux-amd64-v0.9.4.tar.gz -C /usr/local/bin
```

3. Create a user account for VictoriaTraces:

```sh
sudo useradd -s /usr/sbin/nologin victoriatraces
```

See recommendations for installing each type of cluster component below.

> Please note, every cluster component is listening on `:10428` for HTTP connections by default (see `-httpListenAddr` flag).

#### Installing vtstorage

1. Create a folder for storing `vtstorage` data:

```sh
sudo mkdir -p /var/lib/vtstorage && sudo chown -R victoriatraces:victoriatraces /var/lib/vtstorage
```

2. Create a Linux Service for `vtstorage` by running the following command:

```sh
sudo bash -c 'cat <<END >/etc/systemd/system/vtstorage.service
[Unit]
Description=VictoriaTraces vtstorage service
After=network.target

[Service]
Type=simple
User=victoriatraces
Group=victoriatraces
Restart=always
ExecStart=/usr/local/bin/victoria-traces-prod -storageDataPath=/var/lib/vtstorage

PrivateTmp=yes
ProtectHome=yes
NoNewPrivileges=yes
ProtectSystem=full

[Install]
WantedBy=multi-user.target
END'
```

3. Start and Enable `vtstorage`:

```sh
sudo systemctl daemon-reload && sudo systemctl enable --now vtstorage
```

4. Check that the service started successfully:

```sh
sudo systemctl status vtstorage
```

5. After `vtstorage` is in `Running` state, confirm the service is healthy by visiting `http://<ip_or_hostname>:10428/health` link.
It should return `OK`.

#### Installing vtinsert

1. Create a Linux Service for `vtinsert` by running the following command:

```sh
sudo bash -c 'cat <<END >/etc/systemd/system/vtinsert.service
[Unit]
Description=VictoriaTraces vtinsert service
After=network.target

[Service]
Type=simple
User=victoriatraces
Group=victoriatraces
Restart=always
ExecStart=/usr/local/bin/victoria-traces-prod -storageNode=<list of vtstorages> -select.disable

PrivateTmp=yes
ProtectHome=yes
NoNewPrivileges=yes
ProtectSystem=full

[Install]
WantedBy=multi-user.target
END'
```

Replace `<list of vtstorages>` with comma-separated addresses of previously configured `vtstorage` services
(e.g. `vtstorage-1:10428,vtstorage-2:10428`). See more details in the `-storageNode` flag description
in [cluster docs](https://docs.victoriametrics.com/victoriatraces/cluster/#architecture).
The `-select.disable` flag makes this node serve the [insert APIs](https://docs.victoriametrics.com/victoriatraces/data-ingestion/#http-apis) only.

2. Start and Enable `vtinsert`:

```sh
sudo systemctl daemon-reload && sudo systemctl enable --now vtinsert.service
```

3. Check that the service started successfully:

```sh
sudo systemctl status vtinsert.service
```

4. After `vtinsert` is in `Running` state, confirm the service is healthy by visiting `http://<ip_or_hostname>:10428/health` link.
It should return `OK`.

#### Installing vtselect

1. Create a Linux Service for `vtselect` by running the following command:

```sh
sudo bash -c 'cat <<END >/etc/systemd/system/vtselect.service
[Unit]
Description=VictoriaTraces vtselect service
After=network.target

[Service]
Type=simple
User=victoriatraces
Group=victoriatraces
Restart=always
ExecStart=/usr/local/bin/victoria-traces-prod -storageNode=<list of vtstorages> -insert.disable

PrivateTmp=yes
ProtectHome=yes
NoNewPrivileges=yes
ProtectSystem=full

[Install]
WantedBy=multi-user.target
END'
```

Replace `<list of vtstorages>` with comma-separated addresses of previously configured `vtstorage` services,
the same way as for `vtinsert`. The `-insert.disable` flag makes this node serve
the [select APIs](https://docs.victoriametrics.com/victoriatraces/querying/#http-api) only.

2. Start and Enable `vtselect`:

```sh
sudo systemctl daemon-reload && sudo systemctl enable --now vtselect.service
```

3. Check that the service started successfully:

```sh
sudo systemctl status vtselect.service
```

4. After `vtselect` is in `Running` state, confirm the service is healthy by visiting `http://<ip_or_hostname>:10428/select/vmui` link.
It should open the [Web UI](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui) page.

### Helm charts

You can run VictoriaTraces in a Kubernetes environment
with [VictoriaTraces single](https://docs.victoriametrics.com/helm/victoria-traces-single/)
or [cluster](https://docs.victoriametrics.com/helm/victoria-traces-cluster/) Helm charts.

### VictoriaMetrics Operator

You can also run VictoriaTraces in Kubernetes using [VictoriaMetrics Operator](https://docs.victoriametrics.com/operator/resources/).

- [`VTSingle` CRD](https://docs.victoriametrics.com/operator/resources/vtsingle/) declaratively defines a single-node VictoriaTraces deployment.
- [`VTCluster` CRD](https://docs.victoriametrics.com/operator/resources/vtcluster/) declaratively defines a VictoriaTraces cluster and lets the Operator manage `vtinsert`, `vtselect` and `vtstorage` components for you.


## Write data

VictoriaTraces can accept trace spans via [the OpenTelemetry protocol (OTLP)](https://opentelemetry.io/docs/specs/otlp/).

It provides the following HTTP API:

- `/insert/opentelemetry/v1/traces`

and the OpenTelemetry Collector gRPC [TraceService](https://github.com/open-telemetry/opentelemetry-proto/blob/v1.8.0/opentelemetry/proto/collector/trace/v1/trace_service.proto#L30).

These enable user to ingest trace spans through [OTLP/HTTP](https://opentelemetry.io/docs/specs/otlp/#otlphttp) and [OTLP/gRPC](https://opentelemetry.io/docs/specs/otlp/#otlpgrpc).

To test the data ingestion, run the following command:

```shell
curl -X POST -H 'Content-Type: application/json' http://<victoria-traces>:10428/insert/opentelemetry/v1/traces -d '
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "frontend-web"}},
        {"key": "telemetry.sdk.language", "value": {"stringValue": "webjs"}},
        {"key": "telemetry.sdk.name", "value": {"stringValue": "opentelemetry"}},
        {"key": "telemetry.sdk.version", "value": {"stringValue": "1.30.1"}},
        {"key": "process.runtime.name", "value": {"stringValue": "browser"}},
        {"key": "process.runtime.description", "value": {"stringValue": "Web Browser"}},
        {"key": "process.runtime.version", "value": {"stringValue": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}
      ]
    },
    "scopeSpans": [{
      "scope": {"name": "@opentelemetry/instrumentation-document-load", "version": "0.44.1"},
      "spans": [{
        "traceId": "1af5dd013a30efe7f2970032ab81958b",
        "spanId": "229d083a6c480511",
        "parentSpanId": "",
        "name": "documentLoad",
        "kind": 1,
        "startTimeUnixNano": "'$(date +%s000000000)'",
        "endTimeUnixNano": "'$(date +%s000000000)'",
        "attributes": [
          {"key": "session.id", "value": {"stringValue": "96e702c3-6f05-4f54-b2b3-2fad2b7b7995"}},
          {"key": "http.url", "value": {"stringValue": "http://frontend-proxy:8080/cart"}},
          {"key": "http.user_agent", "value": {"stringValue": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/136.0.0.0 Safari/537.36"}}
        ],
        "events": [{"timeUnixNano": "1757320936519100098", "name": "fetchStart"}],
        "status": {}
      }]
    }]
  }]
}'
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

After that, you can check the data in VMUI at `http://<victoria-traces>:10428/select/vmui`.

See more details about how to send data to VictoriaTraces from **an instrumented application** or **an OpenTelemetry collector** [in this doc](https://docs.victoriametrics.com/victoriatraces/data-ingestion/opentelemetry/).

## Read data

[VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) has built-in VMUI for browsing data by span at `http://<victoria-traces>:10428/select/vmui`.

[VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) also provides [Jaeger Query Service JSON APIs](https://www.jaegertracing.io/docs/2.6/apis/#internal-http-json).
It allows users to visualize trace data on Grafana, by simply adding a [Jaeger datasource](https://grafana.com/docs/grafana/latest/datasources/jaeger/) with VictoriaTraces URL:

```
http://<victoria-traces>:10428/select/jaeger
```

See more details about the HTTP APIs and params VictoriaTraces supports and how to query data from them [in this doc](https://docs.victoriametrics.com/victoriatraces/querying/).

## Alerting

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
