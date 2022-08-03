# cmon-exporter
Exporter for CMON part of the ClusterControl project http://www.severalnines.com/.

## Build
make

## How to run
CMON_USERNAME=johan CMON_PASSWORD=secret CMON_ENDPOINT=https://127.0.0.1:9501 ./cmon_exporter

## Docker:
sudo docker run --net=host -it  -e CMON_USERNAME=johan -e CMON_PASSWORD=secret -e CMON_ENDPOINT=https://127.0.0.1:9501  severalnines/cmon_exporter

## Check you get metrics:
curl 127.0.0.1:9954/metrics |grep cmon_up
