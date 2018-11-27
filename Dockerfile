FROM alpine:3.7

RUN apk add --no-cache ca-certificates
ADD server /go/bin/server
EXPOSE 1234
ENTRYPOINT ["/go/bin/server"]
