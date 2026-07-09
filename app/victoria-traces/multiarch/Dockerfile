ARG root_image=non-existing

FROM $root_image
EXPOSE 10428
ENTRYPOINT ["/victoria-traces-prod"]
ARG TARGETARCH
COPY victoria-traces-linux-${TARGETARCH}-prod ./victoria-traces-prod
