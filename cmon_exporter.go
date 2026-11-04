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
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	refreshDuration = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "exporter", "refresh_duration_seconds"),
		"How long the last collection from CMON took.",
		nil, nil,
	)
	lastRefresh = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "exporter", "last_refresh_timestamp_seconds"),
		"Unix time when the last collection from CMON finished.",
		nil, nil,
	)
)

const defaultRefreshInterval = 30 * time.Second

type Exporter struct {
	client *cmonClient
	// controllerId is the last one CMON reported, kept for cmon_up when CMON fails.
	controllerId string

	mu      sync.RWMutex
	metrics []prometheus.Metric
}

func ScanCoredumps(coredumpDir string) float64 {
	totalCoredumps := 0

	err := filepath.Walk(coredumpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasPrefix(filepath.Base(path), "core.") {
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
		client: newCmonClient(cmonEndpoint, cmonUsername, cmonPassword, 30*time.Second),
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
	ch <- refreshDuration
	ch <- lastRefresh
}

// Collect serves the metrics of the last refresh and never calls CMON.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mu.RLock()
	metrics := e.metrics
	e.mu.RUnlock()

	for _, m := range metrics {
		ch <- m
	}
}

// Run refreshes the metrics from CMON forever, waiting interval after each refresh.
func (e *Exporter) Run(interval time.Duration) {
	for {
		time.Sleep(interval)
		e.refresh()
	}
}

func (e *Exporter) refresh() {
	start := time.Now()

	ch := make(chan prometheus.Metric)
	go func() {
		e.collectFromCmon(ch)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for m := range ch {
		metrics = append(metrics, m)
	}

	end := time.Now()
	metrics = append(metrics,
		prometheus.MustNewConstMetric(refreshDuration, prometheus.GaugeValue, end.Sub(start).Seconds()),
		prometheus.MustNewConstMetric(lastRefresh, prometheus.GaugeValue, float64(end.Unix())))

	e.mu.Lock()
	e.metrics = metrics
	e.mu.Unlock()
}

func (e *Exporter) collectFromCmon(ch chan<- prometheus.Metric) {

	ch <- prometheus.MustNewConstMetric(
		coredumpDetectedTotal, prometheus.GaugeValue, ScanCoredumps("/etc/cmon.d"))

	clusters, controllerId, err := e.client.getAllClusterInfo()
	if err != nil {
		log.Println("Error getting clusters info:", err)
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0, e.client.version, e.controllerId)
		return
	}
	e.controllerId = controllerId

	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1, e.client.version, controllerId)

	alarmsByCluster := map[uint64][]cmonAlarm{}
	alarms, alarmsErr := e.client.getAlarms()
	if alarmsErr != nil {
		log.Println("Error getting alarms:", alarmsErr)
	} else {
		for _, alarm := range alarms {
			alarmsByCluster[alarm.ClusterID] = append(alarmsByCluster[alarm.ClusterID], alarm)
		}
	}

	totalCriticalAlarms, totalCount, totalStarted, totalDegraded, totalStopped, totalUnknown, totalFailed, totalBackupFailedAlarms, totalClusterFailedInitAlarms := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	for _, cluster := range clusters {
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
		if alarmsErr == nil {
			failedUploadBackupAlarms, failedBackupAlarms, clusterFailedInitAlarms := 0.0, 0.0, 0.0

			for _, alarm := range alarmsByCluster[cluster.ClusterID] {
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

	if alarmsErr != nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		criticalAlarmsTotal, prometheus.CounterValue, totalCriticalAlarms, controllerId)
	ch <- prometheus.MustNewConstMetric(
		backupFailedTotal, prometheus.CounterValue, totalBackupFailedAlarms, controllerId)
	ch <- prometheus.MustNewConstMetric(
		clusterFailedInitTotal, prometheus.CounterValue, totalClusterFailedInitAlarms, controllerId)
}

func main() {
	flag.Parse()

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

	refreshInterval := defaultRefreshInterval
	if s := os.Getenv("CMON_REFRESH_INTERVAL"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			log.Fatalf("Env variable CMON_REFRESH_INTERVAL=%q is not a positive duration.", s)
		}
		refreshInterval = d
	}

	exporter := NewExporter(cmonEndpoint, cmonUsername, cmonPassword)
	prometheus.MustRegister(exporter)
	log.Printf("Using connection endpoint: %s, refresh interval: %s", cmonEndpoint, refreshInterval)

	exporter.refresh()
	go exporter.Run(refreshInterval)

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
