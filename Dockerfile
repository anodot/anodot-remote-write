FROM alpine:3.7

RUN apk add --no-cache ca-certificates
ADD anodot-remote-write /go/bin/anodot-remote-write
EXPOSE 1234
ENTRYPOINT ["/go/bin/anodot-remote-write"]