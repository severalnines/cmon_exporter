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
# HELP cmon_alarms_backup_failed_total Total number of failed backups alarms.
# TYPE cmon_alarms_backup_failed_total counter
cmon_alarms_backup_failed_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_alarms_critical_total Total number of clusters in unknown state.
# TYPE cmon_alarms_critical_total counter
cmon_alarms_critical_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_backup_failed Is there an active backup failed alarm on the cluster.
# TYPE cmon_cluster_backup_failed counter
cmon_cluster_backup_failed{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 0
cmon_cluster_backup_failed{ClusterID="402",ClusterName="d34fe2fd-17a4-4c1f-8cb9-730331c305bd",ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_backup_upload_failed Is there an active backup failed alarm on the cluster.
# TYPE cmon_cluster_backup_upload_failed counter
cmon_cluster_backup_upload_failed{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 0
cmon_cluster_backup_upload_failed{ClusterID="402",ClusterName="d34fe2fd-17a4-4c1f-8cb9-730331c305bd",ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_degraded Is the cluster up (DEGRADED) or not.
# TYPE cmon_cluster_degraded counter
cmon_cluster_degraded{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_degraded_total Total number of clusters in degraded state.
# TYPE cmon_cluster_degraded_total counter
cmon_cluster_degraded_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_failed_init Cluster failed to initialize.
# TYPE cmon_cluster_failed_init counter
cmon_cluster_failed_init{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 0
cmon_cluster_failed_init{ClusterID="402",ClusterName="d34fe2fd-17a4-4c1f-8cb9-730331c305bd",ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_failed_init_total Total number of clusters that failed to initialize..
# TYPE cmon_cluster_failed_init_total counter
cmon_cluster_failed_init_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_failed_total Total number of clusters in failed state.
# TYPE cmon_cluster_failed_total counter
cmon_cluster_failed_total{ControllerId="00000000-0000-0000-0000-000000000000"} 1
# HELP cmon_cluster_failure Is the cluster up (FAILURE) or not.
# TYPE cmon_cluster_failure counter
cmon_cluster_failure{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 0
cmon_cluster_failure{ClusterID="402",ClusterName="d34fe2fd-17a4-4c1f-8cb9-730331c305bd",ControllerId="00000000-0000-0000-0000-000000000000"} 1
# HELP cmon_cluster_started_total Total number of clusters in started state.
# TYPE cmon_cluster_started_total counter
cmon_cluster_started_total{ControllerId="00000000-0000-0000-0000-000000000000"} 1
# HELP cmon_cluster_stopped_total Total number of clusters in stopped state.
# TYPE cmon_cluster_stopped_total counter
cmon_cluster_stopped_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_total Total number of clusters.
# TYPE cmon_cluster_total counter
cmon_cluster_total{ControllerId="00000000-0000-0000-0000-000000000000"} 2
# HELP cmon_cluster_unknown_total Total number of clusters in unknown state.
# TYPE cmon_cluster_unknown_total counter
cmon_cluster_unknown_total{ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_cluster_up Is the cluster up (STARTED) or not.
# TYPE cmon_cluster_up counter
cmon_cluster_up{ClusterID="397",ClusterName="251635de-08f5-4099-8cf0-6637c4fbeddc",ControllerId="00000000-0000-0000-0000-000000000000"} 1
cmon_cluster_up{ClusterID="402",ClusterName="d34fe2fd-17a4-4c1f-8cb9-730331c305bd",ControllerId="00000000-0000-0000-0000-000000000000"} 0
# HELP cmon_up Was the last  CMON query successful.
# TYPE cmon_up gauge
cmon_up{CmonVersion="1.9.8.7039",ControllerId="00000000-0000-0000-0000-000000000000"} 1

```
