---
weight: 30
title: Roadmap
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
- [ ] Provide [HTTP APIs](https://grafana.com/docs/tempo/latest/api_docs/) of Tempo Query-frontend.

The following functionality is planned in the future versions of VictoriaTraces after GA:

- [ ] Provide a web UI to visualize traces.
- [ ] Provide more analytical functionality based on trace data.
- [ ] Build an efficient trace data collection agent (vtagent).
    - [ ] Support tail-based sampling/downsampling.

The following features are planned for enterprise version:
- [ ] Advanced per-tenant stats.
- [ ] Automatic discovery of vtstorage nodes.

Refer to [the Roadmap of VictoriaLogs](https://docs.victoriametrics.com/victorialogs/roadmap/#) as well for information
about object storage and retention filters.
