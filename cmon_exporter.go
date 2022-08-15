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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/severalnines/cmon-proxy/cmon"
	//	"github.com/severalnines/cmon-proxy/cmon/api"
	//"encoding/json"
	"github.com/severalnines/cmon-proxy/config"
	"log"
	"net/http"
	"os"
)

const namespace = "cmon"

var (
	labels = []string{"name"}
	up     = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last  CMON query successful.",
		nil, nil,
	)

	clusterUp = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_up"),
		"Is the cluster up (STARTED) or not.",
		labels, nil,
	)

	totalClustersCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_total"),
		"Total number of clusters.",
		nil, nil,
	)

	clusterFailedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_failed_total"),
		"Total number of clusters in failed state.",
		nil, nil,
	)

	clusterStartedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_started_total"),
		"Total number of clusters in started state.",
		nil, nil,
	)
	clusterDegradedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_degraded_total"),
		"Total number of clusters in degraded state.",
		nil, nil,
	)

	clusterStoppedTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_stopped_total"),
		"Total number of clusters in stopped state.",
		nil, nil,
	)
	//x                   = []string{"cluster"}
	clusterUnknownTotal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cluster_unknown_total"),
		"Total number of clusters in unknown state.",
		nil,
		nil,
	)

	listenAddress = flag.String("web.listen-address", ":9954",
		"Address to listen on for telemetry")
	metricsPath = flag.String("web.telemetry-path", "/metrics",
		"Path under which to expose metrics")
)

type Exporter struct {
	cmonEndpoint, cmonUsername, cmonPassword string
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
	ch <- totalClustersCount
	ch <- clusterFailedTotal
	ch <- clusterStartedTotal
	ch <- clusterDegradedTotal
	ch <- clusterStoppedTotal
	ch <- clusterUnknownTotal
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
			up, prometheus.GaugeValue, 0)
		return
	}
	res, err := client.GetAllClusterInfo(nil)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0)
		log.Println(err)
	}
	//else {
	//	_, _ := json.Marshal(res)
	//	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1)
	totalCount, totalStarted, totalDegraded, totalStopped, totalUnknown, totalFailed := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	for _, cluster := range res.Clusters {
		if cluster.State == "STARTED" {
			totalStarted++
			ch <- prometheus.MustNewConstMetric(
				clusterUp, prometheus.CounterValue, 1, cluster.ClusterName)
		} else {
			ch <- prometheus.MustNewConstMetric(
				clusterUp, prometheus.CounterValue, 0, cluster.ClusterName)
		}
		if cluster.State == "FAILURE" {
			totalFailed++
		}
		if cluster.State == "DEGRADED" {
			totalDegraded++
		}
		if cluster.State == "UNKNOWN" {
			totalUnknown++
		}
		if cluster.State == "STOPPED" {
			totalStopped++
		}
		totalCount++
	}

	ch <- prometheus.MustNewConstMetric(
		clusterFailedTotal, prometheus.CounterValue, totalFailed)
	ch <- prometheus.MustNewConstMetric(
		clusterStartedTotal, prometheus.CounterValue, totalStarted)
	ch <- prometheus.MustNewConstMetric(
		clusterDegradedTotal, prometheus.CounterValue, totalDegraded)
	ch <- prometheus.MustNewConstMetric(
		clusterStoppedTotal, prometheus.CounterValue, totalStopped)
	ch <- prometheus.MustNewConstMetric(
		clusterUnknownTotal, prometheus.CounterValue, totalUnknown)

	ch <- prometheus.MustNewConstMetric(
		totalClustersCount, prometheus.CounterValue, totalCount)

}

/*func parseStatusField(value string) int64 {
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
