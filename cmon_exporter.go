// Copyright 2022 Severalnines
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/severalnines/cmon-proxy/cmon"
	"github.com/severalnines/cmon-proxy/cmon/api"
	//"encoding/json"
	"github.com/severalnines/cmon-proxy/config"
)

const namespace = "cmon"

var (
	labels     = []string{"ClusterName", "ClusterID", "ControllerId"}
	labels2    = []string{"ControllerId"}
	labelsCmon = []string{"CmonVersion", "ControllerId"}
	up         = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last  CMON query successful.",
		labelsCmon, nil,
	)

	clusterUp = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_up"),
		"Is the cluster up (STARTED) or not.",
		labels, nil,
	)

	clusterFailed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_failure"),
		"Is the cluster up (FAILURE) or not.",
		labels, nil,
	)

	clusterDegraded = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_degraded"),
		"Is the cluster up (DEGRADED) or not.",
		labels, nil,
	)

	totalClustersCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_total"),
		"Total number of clusters.",
		labels2, nil,
	)

	clusterFailedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_failed_total"),
		"Total number of clusters in failed state.",
		labels2, nil,
	)

	clusterStartedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_started_total"),
		"Total number of clusters in started state.",
		labels2, nil,
	)
	clusterDegradedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_degraded_total"),
		"Total number of clusters in degraded state.",
		labels2, nil,
	)

	clusterStoppedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_stopped_total"),
		"Total number of clusters in stopped state.",
		labels2, nil,
	)
	//x                   = []string{"cluster"}
	clusterUnknownTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_unknown_total"),
		"Total number of clusters in unknown state.",
		labels2,
		nil,
	)

	criticalAlarmsTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "alarms_critical_total"),
		"Total number of clusters in unknown state.",
		labels2,
		nil,
	)
	backupFailedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "alarms_backup_failed_total"),
		"Total number of failed backups alarms.",
		labels2,
		nil,
	)
	clusterBackupFailed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_backup_failed"),
		"Is there an active backup failed alarm on the cluster.",
		labels, nil,
	)

	clusterBackupUploadFailed = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_backup_upload_failed"),
		"Is there an active backup failed alarm on the cluster.",
		labels, nil,
	)

	listenAddress = flag.String("web.listen-address", ":9954",
		"Address to listen on for telemetry")
	metricsPath = flag.String("web.telemetry-path", "/metrics",
		"Path under which to expose metrics")
)

type Exporter struct {
	cmonEndpoint, cmonUsername, cmonPassword string
}

func Dummy() error {
	return nil
}

func NewExporter(cmonEndpoint string, cmonUsername string, cmonPassword string) *Exporter {
	return &Exporter{
		cmonEndpoint: cmonEndpoint,
		cmonUsername: cmonUsername,
		cmonPassword: cmonPassword,
	}
}
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- clusterUp
	ch <- clusterFailed
	ch <- clusterDegraded
	ch <- totalClustersCount
	ch <- clusterFailedTotal
	ch <- clusterStartedTotal
	ch <- clusterDegradedTotal
	ch <- clusterStoppedTotal
	ch <- clusterUnknownTotal
	ch <- criticalAlarmsTotal
	ch <- backupFailedTotal
	ch <- clusterBackupFailed
	ch <- clusterBackupUploadFailed
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	client := cmon.NewClient(&config.CmonInstance{
		Url:      e.cmonEndpoint,
		Username: e.cmonUsername,
		Password: e.cmonPassword},
		30)

	err := client.Authenticate()
	if err != nil {
		res, err := client.Ping()
		log.Println("Test: ", err, res)
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0, "")
		return
	}
	controllerId := client.ControllerID()
	serverVersion := client.ServerVersion()

	res, err := client.GetAllClusterInfo(&api.GetAllClusterInfoRequest{
		WithOperation:    &api.WithOperation{Operation: "getAllClusterInfo"},
		WithSheetInfo:    false,
		WithDatabases:    false,
		WithLicenseCheck: false,
		WithHosts:        false,
	})
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	//else {
	//	_, _ := json.Marshal(res)
	//	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1, serverVersion, controllerId)
	totalCriticalAlarms, totalCount, totalStarted, totalDegraded, totalStopped, totalUnknown, totalFailed, totalBackupFailedAlarms := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	for _, cluster := range res.Clusters {
		//		log.Println(cluster.Hosts[0])
		clusterIdStr := strconv.FormatUint(cluster.ClusterID, 10)
		if cluster.State == "STARTED" {
			totalStarted++
			ch <- prometheus.MustNewConstMetric(
				clusterUp, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			ch <- prometheus.MustNewConstMetric(
				clusterFailed, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
			ch <- prometheus.MustNewConstMetric(
				clusterDegraded, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
		} else {
			ch <- prometheus.MustNewConstMetric(
				clusterUp, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
		}
		if cluster.State == "FAILURE" {
			ch <- prometheus.MustNewConstMetric(
				clusterFailed, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			totalFailed++
		}
		if cluster.State == "DEGRADED" {
			ch <- prometheus.MustNewConstMetric(
				clusterDegraded, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			totalDegraded++
		}
		if cluster.State == "UNKNOWN" {
			totalUnknown++
		}
		if cluster.State == "STOPPED" {
			totalStopped++
		}

		totalCount++
		res3, err := client.GetAlarms(cluster.ClusterID)
		if err != nil {
			log.Println("getting alarms for", cluster.ClusterID, err)
		} else {
			failedUploadBackupAlarms, failedBackupAlarms := 0.0, 0.0
			for _, alarm := range res3.Alarms {
				if alarm.SeverityName == "ALARM_CRITICAL" {
					totalCriticalAlarms++
				}
				if alarm.TypeName == "BackupFailed" {
					totalBackupFailedAlarms++
					failedBackupAlarms++
				}
				if alarm.TypeName == "BackupUploadToCloudFailed" {
					failedUploadBackupAlarms++
				}
			}
			ch <- prometheus.MustNewConstMetric(
				clusterBackupFailed, prometheus.CounterValue, failedBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
			ch <- prometheus.MustNewConstMetric(
				clusterBackupUploadFailed, prometheus.CounterValue, failedUploadBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
		}

	}

	ch <- prometheus.MustNewConstMetric(
		clusterFailedTotal, prometheus.CounterValue, totalFailed, controllerId)
	ch <- prometheus.MustNewConstMetric(
		clusterStartedTotal, prometheus.CounterValue, totalStarted, controllerId)
	ch <- prometheus.MustNewConstMetric(
		clusterDegradedTotal, prometheus.CounterValue, totalDegraded, controllerId)
	ch <- prometheus.MustNewConstMetric(
		clusterStoppedTotal, prometheus.CounterValue, totalStopped, controllerId)
	ch <- prometheus.MustNewConstMetric(
		clusterUnknownTotal, prometheus.CounterValue, totalUnknown, controllerId)
	ch <- prometheus.MustNewConstMetric(
		totalClustersCount, prometheus.CounterValue, totalCount, controllerId)
	ch <- prometheus.MustNewConstMetric(
		criticalAlarmsTotal, prometheus.CounterValue, totalCriticalAlarms, controllerId)
	ch <- prometheus.MustNewConstMetric(
		backupFailedTotal, prometheus.CounterValue, totalBackupFailedAlarms, controllerId)

	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0)
		log.Println(err)
	}
}

/*
	func parseStatusField(value string) int64 {
		switch value {
		case "UP", "UP 1/3", "UP 2/3", "OPEN", "no check", "DRAIN":
			return 1
		case "DOWN", "DOWN 1/2", "NOLB", "MAINT", "MAINT(via)", "MAINT(resolution)":
			return 0
		default:
			return 0
		}
	}
*/
func main() {

	cmonEndpoint := os.Getenv("CMON_ENDPOINT")
	cmonUsername := os.Getenv("CMON_USERNAME")
	cmonPassword := os.Getenv("CMON_PASSWORD")

	if cmonEndpoint == "" {
		cmonEndpoint = "https://127.0.0.1:9501"
	}

	if cmonUsername == "" {
		log.Fatalf("Env variable CMON_USERNAME is not set.")
	}

	if cmonPassword == "" {
		log.Fatalf("Env variable CMON_PASSWORD is not set.")
	}

	exporter := NewExporter(cmonEndpoint, cmonUsername, cmonPassword)
	prometheus.MustRegister(exporter)
	log.Printf("Using connection endpoint: %s", cmonEndpoint)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>Cmon Channel Exporter</title></head>
             <body>
             <h1>Cmon Channel Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
