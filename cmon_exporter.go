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
		"Unix time when the last collection that got both clusters and alarms from CMON finished.",
		nil, nil,
	)
)

const defaultRefreshInterval = 30 * time.Second

type Exporter struct {
	client *cmonClient
	// controllerId is the last one CMON reported, kept for cmon_up when CMON fails.
	controllerId string
	// alarms is the last successful getAlarms result, reused when the call fails.
	alarms     []cmonAlarm
	haveAlarms bool
	// alarmClusters are the clusters that existed when CMON returned alarms.
	alarmClusters map[uint64]bool
	// lastSuccess is when CMON last answered both getAllClusterInfo and getAlarms.
	lastSuccess time.Time

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
	metrics, ok := e.collectFromCmon()
	end := time.Now()

	if ok {
		e.lastSuccess = end
	}

	metrics = append(metrics,
		prometheus.MustNewConstMetric(refreshDuration, prometheus.GaugeValue, end.Sub(start).Seconds()))
	if !e.lastSuccess.IsZero() {
		metrics = append(metrics,
			prometheus.MustNewConstMetric(lastRefresh, prometheus.GaugeValue, float64(e.lastSuccess.Unix())))
	}

	e.mu.Lock()
	e.metrics = metrics
	e.mu.Unlock()
}

// collectFromCmon returns the metrics and whether CMON answered both getAllClusterInfo and getAlarms.
func (e *Exporter) collectFromCmon() ([]prometheus.Metric, bool) {
	var metrics []prometheus.Metric
	add := func(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labels ...string) {
		metrics = append(metrics, prometheus.MustNewConstMetric(desc, valueType, value, labels...))
	}

	add(coredumpDetectedTotal, prometheus.GaugeValue, ScanCoredumps("/etc/cmon.d"))

	clusters, controllerId, err := e.client.getAllClusterInfo()
	if err != nil {
		log.Println("Error getting clusters info:", err)
		add(up, prometheus.GaugeValue, 0, e.client.version, e.controllerId)
		return metrics, false
	}
	if controllerId != "" {
		e.controllerId = controllerId
	}
	controllerId = e.controllerId

	add(up, prometheus.GaugeValue, 1, e.client.version, controllerId)

	alarms, err := e.client.getAlarms()
	alarmsOk := err == nil
	if alarmsOk {
		e.alarms, e.haveAlarms = alarms, true
		e.alarmClusters = map[uint64]bool{}
		for _, cluster := range clusters {
			e.alarmClusters[cluster.ClusterID] = true
		}
	} else if e.haveAlarms {
		log.Println("Error getting alarms, reporting the last ones:", err)
	} else {
		log.Println("Error getting alarms:", err)
	}

	alarmsByCluster := map[uint64][]cmonAlarm{}
	for _, alarm := range e.alarms {
		alarmsByCluster[alarm.ClusterID] = append(alarmsByCluster[alarm.ClusterID], alarm)
	}

	totalCriticalAlarms, totalCount, totalStarted, totalDegraded, totalStopped, totalUnknown, totalFailed, totalBackupFailedAlarms, totalClusterFailedInitAlarms := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	for _, cluster := range clusters {
		//		log.Println(cluster.Hosts[0])
		clusterIdStr := strconv.FormatUint(cluster.ClusterID, 10)
		if cluster.State == "STARTED" {
			totalStarted++
			add(clusterUp, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			add(clusterFailed, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
			add(clusterDegraded, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
		} else {
			add(clusterUp, prometheus.CounterValue, 0, cluster.ClusterName, clusterIdStr, controllerId)
		}
		if cluster.State == "FAILURE" {
			add(clusterFailed, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			totalFailed++
		}
		if cluster.State == "DEGRADED" {
			add(clusterDegraded, prometheus.CounterValue, 1, cluster.ClusterName, clusterIdStr, controllerId)
			totalDegraded++
		}
		if cluster.State == "UNKNOWN" {
			totalUnknown++
		}
		if cluster.State == "STOPPED" {
			totalStopped++
		}

		totalCount++
		if e.alarmClusters[cluster.ClusterID] {
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

			add(clusterBackupFailed, prometheus.CounterValue, failedBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
			add(clusterBackupUploadFailed, prometheus.CounterValue, failedUploadBackupAlarms, cluster.ClusterName, clusterIdStr, controllerId)
			add(clusterFailedInit, prometheus.CounterValue, clusterFailedInitAlarms, cluster.ClusterName, clusterIdStr, controllerId)
		}
	}

	add(clusterFailedTotal, prometheus.CounterValue, totalFailed, controllerId)
	add(clusterStartedTotal, prometheus.CounterValue, totalStarted, controllerId)
	add(clusterDegradedTotal, prometheus.CounterValue, totalDegraded, controllerId)
	add(clusterStoppedTotal, prometheus.CounterValue, totalStopped, controllerId)
	add(clusterUnknownTotal, prometheus.CounterValue, totalUnknown, controllerId)
	add(totalClustersCount, prometheus.CounterValue, totalCount, controllerId)

	if e.haveAlarms {
		add(criticalAlarmsTotal, prometheus.CounterValue, totalCriticalAlarms, controllerId)
		add(backupFailedTotal, prometheus.CounterValue, totalBackupFailedAlarms, controllerId)
		add(clusterFailedInitTotal, prometheus.CounterValue, totalClusterFailedInitAlarms, controllerId)
	}

	return metrics, alarmsOk
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
