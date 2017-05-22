FROM alpine
RUN set -x \
    && apk --update upgrade \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*
COPY haproxy_exporter /bin/haproxy_exporter
USER nobody:nobody

ENTRYPOINT ["/bin/haproxy_exporter"]
EXPOSE     9101
