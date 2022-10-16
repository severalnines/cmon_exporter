# cmon-exporter
Exporter for CMON part of the ClusterControl project http://www.severalnines.com/.

## Build
```
make
```

## How to run
```
CMON_USERNAME=johan CMON_PASSWORD=secret CMON_ENDPOINT=https://127.0.0.1:9501 ./cmon_exporter
```

## Systemd

Download
https://github.com/severalnines/cmon_exporter/edit/main/systemd/cmon_exporter.service

Copy the file to :
/etc/systemd/system/cmon_exporter.service

Change USER and SECRET to a ClusterControl (cmon user) that has access to all database clusters.

Enable systemd:
```
systemctl enable  /etc/systemd/system/cmon_exporter.service
systemctl restart cmon_exporter
```

## Docker:
```
sudo docker run --net=host -it  -e CMON_USERNAME=johan -e CMON_PASSWORD=secret -e CMON_ENDPOINT=https://127.0.0.1:9501  severalnines/cmon_exporter
```
## Check you get metrics:
```
curl 127.0.0.1:9954/metrics |grep cmon_up
```

## Current metrics
```
cmon_cluster_degraded_total 0
cmon_cluster_failed_total 1
cmon_cluster_started_total 5
cmon_cluster_stopped_total 0
cmon_cluster_total 6
cmon_cluster_unknown_total 0
cmon_cluster_up{name="MDB103"} 1
cmon_cluster_up{name="PG14"} 1
cmon_cluster_up{name="PSQL"} 1
cmon_cluster_up{name="PXC57"} 1
cmon_cluster_up{name="REDIS3"} 1
cmon_cluster_up{name="cluster_608"} 0
cmon_up 1
```
