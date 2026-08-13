---
title: vtagent
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

The main role of `vtagent` is to replicate collected trace spans among multiple VictoriaTraces instance, without 3rd party collectors. 

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

- `-remoteWrite.url` - the VictoriaTraces endpoint for sending the accepted logs to. It must end with `/insert/native`.
  The `-remoteWrite.url` may refer to [DNS SRV](https://en.wikipedia.org/wiki/SRV_record) address.
  See [these docs](https://docs.victoriametrics.com/victoriatraces/vtagent/#srv-urls) for details.

Example command, which starts `vtagent` for accepting trace spans over OTLP/HTTP at the port `10429`
and sends the collected trace spans to VictoriaTraces instance at `victoria-traces-host:10428`:

```sh
/path/to/vtagent-prod -remoteWrite.url=http://victoria-traces-host:10428/insert/native
```

vtagent can also accept data over OTLP/gRPC at the port specified by `-otlpGRPCListenAddr`.

Pass `-help` to `vtagent` in order to see [the full list of supported command-line flags with their descriptions](https://docs.victoriametrics.com/victoriatraces/vtagent/#advanced-usage).

## Advanced usage

`vtagent` can be fine-tuned with various command-line flags. Run `./vtagent -help` in order to see the full list of these flags with their descriptions and default values:

{{% content "vtagent_common_flags.md" %}}
