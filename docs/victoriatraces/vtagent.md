---
title: vtagent
description: "Trace span collection agent for VictoriaTraces. Accepts OTLP spans and replicates them to multiple remote storages with on-disk buffering."
weight: 4
menu:
  docs:
    weight: 4
    parent: "victoriatraces"
    identifier: vtagent
tags:
  - traces
aliases:
  - /vtagent.html
  - /vtagent/index.html
  - /vtagent/
---

`vtagent` is an agent for accepting trace spans from OTLP exporters and remote write them to [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/).

## Features

The main role of `vtagent` is to replicate collected trace spans among multiple VictoriaTraces instances, without 3rd party collectors.

It also works smoothly in environments with unstable connections to VictoriaTraces instances. If the remote storage is unavailable, the collected trace spans
are buffered at the directory specified via `-remoteWrite.tmpDataPath` command-line flag. The buffered spans are sent to remote storage as soon as the connection
to the remote storage is repaired.

`vtagent` **will** support the following features under [the VictoriaTraces roadmap](https://docs.victoriametrics.com/victoriatraces/roadmap/) soon:
- OTLP output support to remotely write trace spans via the OpenTelemetry protocol.
- Efficient tail-based sampling with on-disk buffering of trace spans.
- Retroactive sampling.

## When to use vtagent

Use `vtagent` when:

- You need to replicate trace spans to multiple VictoriaTraces instances for high availability.
- You have unstable connectivity to VictoriaTraces and need on-disk buffering.

## Quick Start

Please download and unpack the `vtutils` archive from [releases page](https://github.com/VictoriaMetrics/VictoriaTraces/releases/latest) (
`vtagent` is also available as Docker images on [Docker Hub](https://hub.docker.com/r/victoriametrics/vtagent/tags)
and [Quay](https://quay.io/repository/victoriametrics/vtagent?tab=tags)), then pass the following command-line flags to the `vtagent-prod` binary:

- `-remoteWrite.url` - the VictoriaTraces endpoint for sending the accepted trace spans to. It must point to an OTLP/HTTP traces endpoint, such as `/insert/opentelemetry/v1/traces` of VictoriaTraces.
  `vtagent` sends trace spans in OTLP/HTTP protobuf format, so any OTLP-compatible backend can be used as well.
  The `-remoteWrite.url` may refer to [DNS SRV](https://en.wikipedia.org/wiki/SRV_record) address.
  See [these docs](https://docs.victoriametrics.com/victoriatraces/vtagent/#srv-urls) for details.

Example command, which starts `vtagent` for accepting trace spans over OTLP/HTTP at the port `10429`
and sends the collected trace spans to VictoriaTraces instance at `victoria-traces-host:10428`:

```sh
/path/to/vtagent-prod -remoteWrite.url=http://victoria-traces:10428/insert/opentelemetry/v1/traces
```

vtagent can also accept data over OTLP/gRPC at the port specified by `-otlpGRPCListenAddr`.

Pass `-help` to `vtagent` in order to see [the full list of supported command-line flags with their descriptions](https://docs.victoriametrics.com/victoriatraces/vtagent/#advanced-usage).

## Replication

`vtagent` can accept multiple `-remoteWrite.url` command-line flags. In this case it replicates every collected span to all the configured remote storages,
which is useful for running multiple VictoriaTraces instances for high availability:

```sh
/path/to/vtagent-prod \
  -remoteWrite.url=http://victoria-traces-1:10428/insert/opentelemetry/v1/traces \
  -remoteWrite.url=http://victoria-traces-2:10428/insert/opentelemetry/v1/traces
```

`vtagent` maintains an independent pipeline for every `-remoteWrite.url`, including in-memory buffers, on-disk queue and connections.
So a slow or unavailable remote storage doesn't delay data delivery to the rest of the remote storages. The spans for the unavailable remote storage
are kept in its queue and are sent as soon as it becomes available again. See [data persistence](https://docs.victoriametrics.com/victoriatraces/vtagent/#data-persistence) for details.

Command-line flags described as applying to "the corresponding -remoteWrite.url" in [the flags list](https://docs.victoriametrics.com/victoriatraces/vtagent/#advanced-usage),
such as `-remoteWrite.maxDiskUsagePerURL`, `-remoteWrite.headers`, `-remoteWrite.basicAuth.*` or `-remoteWrite.tls*`, can be set individually
for every `-remoteWrite.url`, in the same order as the `-remoteWrite.url` flags.

## Data persistence

`vtagent` buffers the accepted trace spans before sending them to remote storage, so they aren't lost when remote storage is slow or unavailable.
The data goes through the following stages for every `-remoteWrite.url`:

1. **Batching.** Accepted OTLP requests are merged in memory into bigger blocks, which reduces the number of requests to remote storage.
   A block is flushed when its size reaches `-remoteWrite.maxBlockSize` or every `-remoteWrite.flushInterval`.
2. **Queueing.** Flushed blocks are put into a queue. When remote storage cannot keep up or is unavailable, the queue is stored at the `-remoteWrite.tmpDataPath` directory.
   Its size is limited only by the free disk space by default. Use `-remoteWrite.maxDiskUsagePerURL` for limiting it; the oldest data is dropped when the limit is reached.
3. **Sending.** Blocks are sent to remote storage via `-remoteWrite.queues` concurrent connections. Failed blocks are re-sent until they are delivered.

The in-memory data is saved to `-remoteWrite.tmpDataPath` on graceful shutdown and is sent to remote storage after the restart.

Bigger `-remoteWrite.maxBlockSize` and `-remoteWrite.flushInterval` values reduce the number of requests to remote storage at the cost of higher memory usage and delivery latency.
Make sure that remote storage accepts requests of `-remoteWrite.maxBlockSize` size.

## SRV URLs

`vtagent` supports [DNS SRV](https://en.wikipedia.org/wiki/SRV_record) hostname resolution in the `-remoteWrite.url` command-line flag when the hostname starts with `srv+` prefix.
For example, the following command instructs `vtagent` to send the data to the TCP address obtained from the `victoria-traces` SRV record:

```sh
/path/to/vtagent-prod -remoteWrite.url=http://srv+victoria-traces/insert/opentelemetry/v1/traces
```

If SRV lookup returns multiple targets, `vtagent` randomly chooses a target per every new connection to the remote storage.

## Advanced usage

`vtagent` can be fine-tuned with various command-line flags. Run `./vtagent -help` in order to see the full list of these flags with their descriptions and default values:

{{% content "vtagent_common_flags.md" %}}
