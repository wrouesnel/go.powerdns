ARG DOCKER_PREFIX=

FROM ${DOCKER_PREFIX}ubuntu:xenial

ARG URL_PGBDR_REPO=http://packages.2ndquadrant.com/bdr/apt/
ARG URL_P2CLI=https://github.com/wrouesnel/p2cli/releases/download/r4/p2
ARG URL_GOSU=https://github.com/tianon/gosu/releases/download/1.10/gosu-amd64

RUN echo "deb [arch=amd64] $URL_PGBDR_REPO xenial-2ndquadrant main" > /etc/apt/sources.list.d/postgres.list

COPY *.asc /tmp/

RUN apt-key add /tmp/*.asc && rm -f /tmp/*.asc

RUN apt-get update && apt-get install --no-install-recommends -y \
    pdns-server pdns-backend-sqlite3 wget \
    # gosu
    wget -O /usr/local/bin/gosu $URL_GOSU && \
    chmod +x /usr/local/bin/gosu && \
    # Templating engine
    wget -O /usr/local/bin/p2 $URL_P2CLI && \
    chmod +x /usr/local/bin/p2
    
COPY tree-postinstall/ /

EXPOSE 53/udp 53/tcp 8080/tcp

ENTRYPOINT [ "/entrypoint.bsh" ]
