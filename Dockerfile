FROM alpine:3.7

RUN apk add --no-cache ca-certificates
ADD anodot-prometheus-remote-write /go/bin/anodot-prometheus-remote-write
EXPOSE 1234
ENTRYPOINT ["/go/bin/anodot-prometheus-remote-write"]