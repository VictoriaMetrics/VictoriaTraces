# Docker compose environment for VictoriaTraces

Docker compose environment for VictoriaTraces includes VictoriaTraces components and [Grafana](https://grafana.com/).

For starting the docker-compose environment ensure that you have docker installed and running, and that you have access
to the Internet.
**All commands should be executed from the root directory of [the VictoriaTraces repo](https://github.com/VictoriaMetrics/VictoriaTraces).**

* Traces:
  * [VictoriaTraces single server](#victoriaTraces-server)
  * [VictoriaTraces cluster](#victoriaTraces-cluster)
* [Common](#common-components)
  * [Grafana](#grafana)
* [Troubleshooting](#troubleshooting)

## VictoriaTraces server

To spin-up environment with [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/), run the following command:

```sh
# clone VictoriaTraces
git clone https://github.com/VictoriaMetrics/VictoriaTraces.git
cd VictoriaTraces

# start docker compose
make docker-vt-single-up
```

_See [compose-vt-single.yml](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/compose-vt-single.yml)_

VictoriaTraces will be accessible on the `--httpListenAddr=:10428` port.

In addition to VictoriaTraces server, the docker compose contains the following components:

* [HotROD](https://hub.docker.com/r/jaegertracing/example-hotrod) application to generate trace data.
* `VictoriaMetrics single-node` to collect metrics from all the components.
* [Grafana](#grafana) is configured with [VictoriaMetrics](https://github.com/VictoriaMetrics/victoriametrics-datasource) and Jaeger datasource pointing to VictoriaTraces server.
* [vmalert](#vmalert) is configured to query `VictoriaTraces single-node`, and send alerts state and recording rules results to `VictoriaMetrics single-node`.
* [alertmanager](#alertmanager) is configured to receive notifications from `vmalert`.

<img alt="VictoriaTraces single-server deployment" width="500" src="assets/vt-single-server.png">

To generate trace data, you need to access HotROD at [http://localhost:8080](http://localhost:8080), and **click any button on the page**.

To access Grafana, use link [http://localhost:3000](http://localhost:3000).

To access [VictoriaTraces UI](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui),
use link [http://localhost:10428/select/vmui](http://localhost:10428/select/vmui).

To shut down environment execute the following command:

```sh
make docker-vt-single-down
```

## VictoriaTraces cluster

To spin-up environment with [VictoriaTraces](https://docs.victoriametrics.com/victoriatraces/) **cluster**, run the following command:

```sh
# start docker compose
make docker-vt-cluster-up
```

_See [compose-vt-cluster.yml](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/compose-vt-cluster.yml)_

VictoriaTraces cluster environment consists of `vtinsert`, `vtstorage` and `vtselect` components.
`vtinsert` and `vtselect` are available through `vmauth` on port `:8427`.
For example, `HotROD` pushes trace spans via `http://vmauth:8427/insert/opentelemetry/v1/traces` path,
and Grafana queries `http://vmauth:8427/select/jaeger/` for datasource queries.

In addition to VictoriaTraces cluster, the docker compose contains the following components:

* [HotROD](https://hub.docker.com/r/jaegertracing/example-hotrod) application to generate trace data.
* `VictoriaMetrics single-node` to collect metrics from all the components.
* [Grafana](#grafana) is configured with [VictoriaMetrics](https://github.com/VictoriaMetrics/victoriametrics-datasource) and Jaeger datasource pointing to VictoriaTraces cluster (vmauth).
* [vmauth](#vmauth) balances incoming read and write requests among `vtselect`s and `vtinsert`s;
* [vmalert](#vmalert) is configured to query `vtselect` (vmauth), and send alerts state and recording rules results to `VictoriaMetrics single-node`.
* [alertmanager](#alertmanager) is configured to receive notifications from `vmalert`.

To generate trace data, you need to access HotROD at [http://localhost:8080](http://localhost:8080), and **click any button on the page**.

To access Grafana, use link [http://localhost:3000](http://localhost:3000).

To access [VictoriaTraces UI](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui),
use link [http://localhost:8427/select/vmui](http://localhost:8427/select/vmui).

To shut down environment execute the following command:

```sh
make docker-vt-cluster-down
```

# Common components

## vmauth

[vmauth](https://docs.victoriametrics.com/victoriametrics/vmauth/) acts as a [load balancer](https://docs.victoriametrics.com/victoriametrics/vmauth/#load-balancing)
to spread the load across `vtselect` and `vtinsert` nodes. [Grafana](#grafana) and [vmalert](#vmalert) use vmauth for read queries.
vmauth routes read queries to VictoriaTraces depending on requested path.
vmauth config is available here for [VictoriaTraces cluster](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/auth-vt-cluster.yml).

## vmalert

There are two vmalert containers which responsible for evaluating alerting rules on VictoriaMetrics and VictoriaTraces.

vmalert-metrics evaluates [alerting rules](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/rules) on VictoriaMetrics base on metrics.

vmalert-traces evaluates [alerting rules](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/vtraces-example-alerts) on VictoriaTraces base on trace spans.

They are connected with AlertManager for firing alerts.

Web interface link:

* vmalert-metrics: [http://localhost:8880/](http://localhost:8880/).
* vmalert-traces: [http://localhost:8881/](http://localhost:8881/).

## alertmanager

AlertManager accepts notifications from `vmalert` and fires alerts.
All notifications are blackholed according to [alertmanager.yml](https://github.com/VictoriaMetrics/VictoriaTraces/blob/master/deployment/docker/alertmanager.yml) config.

Web interface link [http://localhost:9093/](http://localhost:9093/).

## Grafana

Web interface link [http://localhost:3000](http://localhost:3000).

Default credentials:

* login: `admin`
* password: `admin`

Grafana is provisioned with default dashboards and datasources.

## Troubleshooting

This environment has the following requirements:

* installed [docker compose](https://docs.docker.com/compose/);
* access to the Internet for downloading docker images;
* **All commands should be executed from the root directory of [the VictoriaTraces repo](https://github.com/VictoriaMetrics/VictoriaTraces).**

The expected output of running a command like `make docker-vm-single-up` is the following:

```sh
 make docker-vm-single-up                                                                                                           :(
docker compose -f deployment/docker/compose-vm-single.yml up -d
[+] Running 9/9
 ✔ Network docker_default              Created                                                                                                                                                                                     0.0s
 ✔ Volume "docker_vmagentdata"         Created                                                                                                                                                                                     0.0s
 ✔ Container docker-alertmanager-1     Started                                                                                                                                                                                     0.3s
 ✔ Container docker-victoriametrics-1  Started                                                                                                                                                                                     0.3s
...
 ```

Containers are started in [--detach mode](https://docs.docker.com/reference/cli/docker/compose/up/), meaning they run in the background.
As a result, you won't see their logs or exit status directly in the terminal.

If something isn’t working as expected, try the following troubleshooting steps:

1. Run from the correct directory. Make sure you're running the command from the root of the [VictoriaTraces repository](https://github.com/VictoriaMetrics/VictoriaTraces).
2. Check container status. Run `docker ps -a` to list all containers and their status. Healthy and running containers should have `STATUS` set to `Up`.
3. View container logs. To inspect logs for a specific container, get its container ID from step p2 and run: `docker logs -f <containerID>`.
4. Read the logs carefully and follow any suggested actions.
5. Check for port conflicts. Some containers (e.g., Grafana) expose HTTP ports. If a port (like `:3000`) is already in use, the container may fail to start. Stop the conflicting process or change the exposed port in the Docker Compose file.
6. Shut down the deployment. To tear down the environment, run: `make <environment>-down` (i.e. `make docker-vm-single-down`).
   Note, this command also removes all attached volumes, so all the temporary created data will be removed too (i.e. Grafana dashboards or collected metrics).
