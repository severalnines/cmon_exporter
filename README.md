# cmon-exporter
Exporter for CMON part of the ClusterControl project http://www.severalnines.com/.

## Build
```
make
```

## Create a CMON user for the cmon_exporter
```
s9s user --create --group=admins --generate-key --controller=https://127.0.0.1:9501 --new-password="SECRET" --email-address="admin@example.com" cmon_exporter
```


## How to run
```
CMON_USERNAME=cmon_exporter CMON_PASSWORD=SECRET CMON_ENDPOINT=https://127.0.0.1:9501 ./cmon_exporter
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
sudo docker run --net=host -it  -e CMON_USERNAME=cmon_exporter -e CMON_PASSWORD=SECRET -e CMON_ENDPOINT=https://127.0.0.1:9501  severalnines/cmon_exporter
```
## Check you get metrics:
```
curl 127.0.0.1:9954/metrics |grep cmon_up
```

## Current metrics
```
cmon_alarms_backup_failed_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_alarms_critical_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_backup_failed{ClusterID="807",ClusterName="PXC57",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_backup_failed{ClusterID="808",ClusterName="PXC80",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_backup_upload_failed{ClusterID="807",ClusterName="PXC57",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_backup_upload_failed{ClusterID="808",ClusterName="PXC80",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_degraded{ClusterID="807",ClusterName="PXC57",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_degraded{ClusterID="808",ClusterName="PXC80",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_degraded_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_failed_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_failure{ClusterID="807",ClusterName="PXC57",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_failure{ClusterID="808",ClusterName="PXC80",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_started_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 2
cmon_cluster_stopped_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 2
cmon_cluster_unknown_total{ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 0
cmon_cluster_up{ClusterID="807",ClusterName="PXC57",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 1
cmon_cluster_up{ClusterID="808",ClusterName="PXC80",ControllerId="8c11355d-04c6-4c7c-a623-7d166af7657a"} 1
cmon_up{CmonVersion="1.9.7"} 1
```
