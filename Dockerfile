FROM gcr.io/distroless/static-debian12

COPY victoria-traces /victoria-traces

ENTRYPOINT ["/victoria-traces"]