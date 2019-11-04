FROM alpine:3.7

ADD anodot-prometheus-remote-write /go/bin/anodot-prometheus-remote-write
# ln -s required for backward compatibility with older docker images.
RUN apk add --no-cache ca-certificates && ln -s /go/bin/anodot-prometheus-remote-write /go/bin/server
EXPOSE 1234
ENTRYPOINT ["/go/bin/anodot-prometheus-remote-write"]