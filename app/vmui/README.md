# vmui

Web UI for VictoriaTraces

* [Static build](#static-build)
* [Updating vmui embedded into VictoriaTraces](#updating-vmui-embedded-into-victoriatraces)

----

### Static build

Run the following command from the root of VictoriaTraces repository for building `vmui` static contents:

```sh
make vmui-build
```

The built static contents is put into `app/vmui/packages/vmui/` directory.

### Updating vmui embedded into VictoriaTraces

Run the following command from the root of VictoriaTraces repository for updating `vmui` embedded into VictoriaTraces:

```sh
make vmui-update
```

This command should update `vmui` static files at `app/vtselect/vmui` directory. Commit changes to these files if needed.

Then build VictoriaTraces with the following command:

```sh
make victoria-traces
```

Then run the built binary with the following command:

```sh
bin/victoria-traces
```

Then navigate to `http://localhost:10428/vmui/`. See [these docs](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui) for more details.
