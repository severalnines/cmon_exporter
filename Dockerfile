ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="Severalnines <info@severalniens.com>"

ARG ARCH="amd64"
ARG OS="linux"
#COPY .build/${OS}-${ARCH}/cmon_exporter /bin/cmon_exporter
COPY ./cmon_exporter /bin/cmon_exporter

EXPOSE      9954
USER        nobody
ENTRYPOINT  [ "/bin/cmon_exporter" ]
