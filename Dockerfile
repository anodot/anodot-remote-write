FROM registry.access.redhat.com/ubi8/ubi-minimal

ENV ANODOT_TAGS="source=prometheus-remote-write"
ARG VERSION=latest

### Required OpenShift Labels
LABEL name="Anodot Prometheus Remote Write" \
      maintainer="support@anodot.com" \
      vendor="Anodot" \
      version=${VERSION} \
      release="1" \
      summary="https://github.com/anodot/anodot-remote-write" \
      description="Service which receives Prometheus metrics through remote_write, converts metrics and sends them into Anodot"
# Always include a software license in the default location
COPY LICENSE.md /licenses/

COPY anodot-prometheus-remote-write /go/bin/anodot-prometheus-remote-write
# ln -s required for backward compatibility with older docker images.
RUN ln -s /go/bin/anodot-prometheus-remote-write /go/bin/server
EXPOSE 1234
ENTRYPOINT ["/go/bin/anodot-prometheus-remote-write"]
