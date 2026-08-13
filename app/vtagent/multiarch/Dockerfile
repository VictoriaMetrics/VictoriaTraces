ARG root_image=non-existing

FROM $root_image
EXPOSE 10429
ENTRYPOINT ["/vtagent-prod"]
ARG TARGETARCH
COPY vtagent-linux-${TARGETARCH}-prod ./vtagent-prod
