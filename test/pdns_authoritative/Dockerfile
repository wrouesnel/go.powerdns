ARG DOCKER_PREFIX=

FROM ${DOCKER_PREFIX}ubuntu:xenial

ARG URL_P2CLI=https://github.com/wrouesnel/p2cli/releases/download/r4/p2
ARG URL_GOSU=https://github.com/tianon/gosu/releases/download/1.10/gosu-amd64

ARG URL_PDNS_REPO=http://repo.powerdns.com/ubuntu

COPY tree-preinstall/ /

COPY *.asc /tmp/

RUN apt-key add /tmp/*.asc && rm -f /tmp/*.asc

ARG PDNS_AUTH_REPO_TAG

RUN echo "deb [arch=amd64] $URL_PDNS_REPO $PDNS_AUTH_REPO_TAG main" > /etc/apt/sources.list.d/pdns.list

RUN apt-get update && apt-get install --no-install-recommends -y \
    ca-certificates \
    pdns-server pdns-backend-sqlite3 wget sqlite3 && \
    # gosu
    wget -O /usr/local/bin/gosu "${URL_GOSU}" && \
    chmod +x /usr/local/bin/gosu && \
    # Templating engine
    wget -O /usr/local/bin/p2 "${URL_P2CLI}" && \
    chmod +x /usr/local/bin/p2
    
COPY tree-postinstall/ /

EXPOSE 53/udp 53/tcp 8080/tcp

ENTRYPOINT [ "/entrypoint.bsh" ]
