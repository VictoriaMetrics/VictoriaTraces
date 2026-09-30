---
weight: 30
title: Roadmap
description: "Upcoming features toward GA release of VictoriaTraces."
menu:
  docs:
    identifier: vt-roadmap
    parent: victoriatraces
    weight: 30
    title: Roadmap
tags:
  - traces
aliases:
- /victoriatraces/roadmap.html
---

The following items need to be completed before general availability (GA) version:
- [ ] Finalize the data structure and commit to backward compatibility.

The following functionality is planned in the future versions of VictoriaTraces after GA:
- [ ] Provide more analytical functionality based on trace data.
- [ ] Support tail-based sampling/downsampling.

The following features are planned for enterprise version:
- [ ] Advanced per-tenant stats.
- [ ] Automatic discovery of vtstorage nodes.

Refer to [the Roadmap of VictoriaLogs](https://docs.victoriametrics.com/victorialogs/roadmap/#) as well for information
about object storage and retention filters.

The following features are implemented as minimal viable versions but may be further enhanced in the future. We will tracking with this list until GA:
- [x] Provide [HTTP APIs](https://grafana.com/docs/tempo/latest/api_docs/) of Tempo Query-frontend.
- [x] Provide a web UI to visualize traces.
- [x] Build an efficient trace data collection agent (vtagent).