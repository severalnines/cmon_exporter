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

- Build or download a released version: https://github.com/severalnines/cmon_exporter/releases
- copy the cmon_exporter binary to /usr/local/bin

Download
https://github.com/severalnines/cmon_exporter/edit/main/systemd/cmon_exporter.service

Copy the file to :
/etc/systemd/system/cmon_exporter.service

Change USER and SECRET to a ClusterControl (cmon user) that has access to all database clusters.

Enable and start:
```
systemctl enable  /etc/systemd/system/cmon_exporter.service
systemctl restart cmon_exporter
systemctl status cmon_exporter
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
cmon_alarms_backup_failed_total 0
cmon_alarms_critical_total 2
cmon_cluster_backup_failed{cid="749",name="cluster_749"} 0
cmon_cluster_backup_failed{cid="750",name="MSSQL-1N"} 0
cmon_cluster_backup_upload_failed{cid="749",name="cluster_749"} 0
cmon_cluster_backup_upload_failed{cid="750",name="MSSQL-1N"} 0
cmon_cluster_degraded{cid="749",name="cluster_749"} 1
cmon_cluster_degraded{cid="750",name="MSSQL-1N"} 0
cmon_cluster_degraded_total 1
cmon_cluster_failed_total 0
cmon_cluster_failure{cid="750",name="MSSQL-1N"} 0
cmon_cluster_started_total 1
cmon_cluster_stopped_total 0
cmon_cluster_total 2
cmon_cluster_unknown_total 0
cmon_cluster_up{cid="749",name="cluster_749"} 0
cmon_cluster_up{cid="750",name="MSSQL-1N"} 1
cmon_up 1
```
