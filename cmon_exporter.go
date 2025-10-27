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
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/severalnines/cmon-proxy/cmon"
	"github.com/severalnines/cmon-proxy/cmon/api"
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

	clusterFailedInitTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_failed_init_total"),
		"Total number of clusters that failed to initialize..",
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

	clusterUnknownTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_unknown_total"),
		"Total number of clusters in unknown state.",
		labels2, nil,
	)

	criticalAlarmsTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "alarms_critical_total"),
		"Total number of clusters in unknown state.",
		labels2, nil,
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

	clusterFailedInit = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_failed_init"),
		"Cluster failed to initialize.",
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
	coredumpDetectedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "coredump_detected_total"),
		"Total number of coredumps detected",
		nil, nil,
	)
)

type Exporter struct {
	cmonEndpoint, cmonUsername, cmonPassword string
}

func ScanCoredumps(coredumpDir string) float64 {
	totalCoredumps := 0

	err := filepath.Walk(coredumpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path)[:5] == "core." {
			totalCoredumps++
		}
		return nil
	})

	if err != nil {
		log.Printf("Error scanning coredump directory: %v\n", err)
	}

	return float64(totalCoredumps)
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
	ch <- clusterFailedInit
	ch <- clusterFailedInitTotal
	ch <- clusterStartedTotal
	ch <- clusterDegradedTotal
	ch <- clusterStoppedTotal
	ch <- clusterUnknownTotal
	ch <- criticalAlarmsTotal
	ch <- backupFailedTotal
	ch <- clusterBackupFailed
	ch <- clusterBackupUploadFailed
	ch <- coredumpDetectedTotal
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
		log.Println("Error on cmon auth, also ping result:", err, res)
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0, "", "")
		return
	}
	coredumpDir := "/etc/cmon.d"
	totalCoredumps := ScanCoredumps(coredumpDir)
	ch <- prometheus.MustNewConstMetric(
		coredumpDetectedTotal, prometheus.GaugeValue, totalCoredumps)

	controllerId := client.PoolID()
	serverVersion := client.ServerVersion()

	res, err := client.GetAllClusterInfo(&api.GetAllClusterInfoRequest{
		WithOperation:    &api.WithOperation{Operation: "getAllClusterInfo"},
		WithSheetInfo:    false,
		WithDatabases:    false,
		WithLicenseCheck: false,
		WithHosts:        false,
	})
	if err != nil {
		log.Println("Error getting clusters info:", err)
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0, serverVersion, controllerId)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1, serverVersion, controllerId)

	totalCriticalAlarms, totalCount, totalStarted, totalDegraded, totalStopped, totalUnknown, totalFailed, totalBackupFailedAlarms, totalClusterFailedInitAlarms := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
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
			failedUploadBackupAlarms, failedBackupAlarms, clusterFailedInitAlarms := 0.0, 0.0, 0.0

			for _, alarm := range res3.Alarms {
				if alarm.SeverityName == "ALARM_WARNING" {
					continue
				}
				if alarm.SeverityName == "ALARM_CRITICAL" {
					totalCriticalAlarms++
				}
				if alarm.TypeName == "BackupFailed" {
					totalBackupFailedAlarms++
					failedBackupAlarms++
				}
				if alarm.TypeName == "ClusterFailedInit" {
					totalClusterFailedInitAlarms++
					clusterFailedInitAlarms++
				}
				if alarm.TypeName == "BackupUploadToCloudFailed" {
					failedUploadBackupAlarms++
				}
			}

			ch <- prometheus.MustNewConstMetric(
				clusterBackupFailed, prometheus.CounterValue, failedBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
			ch <- prometheus.MustNewConstMetric(
				clusterBackupUploadFailed, prometheus.CounterValue, failedUploadBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
			ch <- prometheus.MustNewConstMetric(
				clusterFailedInit, prometheus.CounterValue, clusterFailedInitAlarms, cluster.ClusterName, clusterIdStr, controllerId)
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
	ch <- prometheus.MustNewConstMetric(
		clusterFailedInitTotal, prometheus.CounterValue, totalClusterFailedInitAlarms, controllerId)
}

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
