---
weight: 4
title: Visualization in Jaeger UI
disableToc: true
menu:
  docs:
    parent: "victoriatraces-querying"
    weight: 4
tags:
  - traces
aliases:
  - /victoriatraces/querying/jaeger-frontend.html
---

[Jaeger UI](https://github.com/jaegertracing/jaeger-ui) is the official frontend that ships with Jaeger. It queries [Jaeger Query Service JSON APIs](https://www.jaegertracing.io/docs/2.6/apis/#internal-http-json)
and visualizes the response trace data.

## Deploy Jaeger UI

You can get Jaeger UI from [release page](https://github.com/jaegertracing/jaeger-ui/releases/tag/v1.70.0).

As it provides only assets and source code, an HTTP server is needed for serving requests.

### Nginx Example

Here's an example where we use Nginx to:

- Serve static content of Jaeger UI.
- Forward query requests to VictoriaTraces.

Assume you already have:

1. VictoriaTraces running locally and listening on port `:10428`.
2. Jaeger UI assets (`index.html` and `/static`) located under `/path/to/jaeger-ui/build/`.
3. Nginx Installed.

Create the following config file `jaeger.conf` and place it to the Nginx config folder:

```
server {
    listen       8080;
    listen  [::]:8080;
    server_name  localhost;

    location / {
        root   /path/to/jaeger-ui/build; # change this path to your asserts location.
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:10428/select/jaeger/api; # change this address to VictoriaTraces' address.
    }
}
```

Here are some common paths of Nginx config folder:

```sh
# Ubuntu & Install with apt
cd /etc/nginx/sites-available/

# MacOS & Install with homebrew
cd /opt/homebrew/etc/nginx/servers/
```

After reloading Nginx, you should be able to visit Jaeger UI on: [http://127.0.0.1:8080/](http://127.0.0.1:8080/).

## Jaeger UI versions

Jaeger UI moved to the [Jaeger HTTP API v3](https://docs.victoriametrics.com/victoriatraces/querying/#jaeger-http-api-v3)
in steps, so different versions call different endpoints:

| Jaeger UI version | Endpoints it calls |
| --- | --- |
| below v2.15 | the v1 endpoints only |
| v2.15 to v2.18 | v3 for the service and span name lists, v1 for search and for a single trace |
| v2.19 and above | also v3 for search, through `/api/v3/trace-summaries` |

VictoriaTraces serves both API versions, so every version above works. The Nginx config shown
above needs no change for v3, since it forwards the whole `/api` path.
