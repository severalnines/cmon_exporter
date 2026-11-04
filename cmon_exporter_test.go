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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const fakeClusters = `{"request_status":"Ok","controller_id":"ctrl","clusters":[
	{"cluster_id":1,"cluster_name":"one","state":"STARTED"},
	{"cluster_id":2,"cluster_name":"two","state":"FAILURE"}]}`

const fakeAlarms = `{"request_status":"Ok","alarms":[
	{"alarm_id":1,"cluster_id":1,"severity_name":"ALARM_CRITICAL","type_name":"BackupFailed"},
	{"alarm_id":2,"cluster_id":1,"severity_name":"ALARM_CRITICAL","type_name":"BackupUploadToCloudFailed"},
	{"alarm_id":3,"cluster_id":2,"severity_name":"ALARM_CRITICAL","type_name":"ClusterFailedInit"},
	{"alarm_id":4,"cluster_id":2,"severity_name":"ALARM_WARNING","type_name":"BackupFailed"},
	{"alarm_id":5,"cluster_id":0,"severity_name":"ALARM_CRITICAL","type_name":"BackupFailed"}]}`

type fakeCmon struct {
	alarmsBody string
	alarmReqs  []map[string]any
	authCalls  int
	// session is the cmon-sid CMON accepts. Each auth issues a new one.
	session string
	version string
}

func (f *fakeCmon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)

	w.Header().Set("Server", "cmon/"+f.version)

	if !strings.HasSuffix(r.URL.Path, "/v2/auth") {
		if c, err := r.Cookie("cmon-sid"); err != nil || f.session == "" || c.Value != f.session {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"request_status":"AuthRequired","error_string":"Authentication required"}`))
			return
		}
	}

	switch {
	case strings.HasSuffix(r.URL.Path, "/v2/auth"):
		f.authCalls++
		if req["password"] != "p" {
			_, _ = w.Write([]byte(`{"request_status":"AccessDenied","error_string":"Username or password is incorrect."}`))
			return
		}
		f.session = fmt.Sprintf("sid-%d", f.authCalls)
		http.SetCookie(w, &http.Cookie{Name: "cmon-sid", Value: f.session})
		_, _ = w.Write([]byte(`{"request_status":"Ok","controller_id":"ctrl","user":{"user_name":"u"}}`))
	case strings.HasSuffix(r.URL.Path, "/v2/clusters"):
		_, _ = w.Write([]byte(fakeClusters))
	case strings.HasSuffix(r.URL.Path, "/v2/alarm"):
		f.alarmReqs = append(f.alarmReqs, req)
		_, _ = w.Write([]byte(f.alarmsBody))
	default:
		http.NotFound(w, r)
	}
}

// collect refreshes a new exporter once and returns its metric values.
func collect(t *testing.T, endpoint string) map[string]float64 {
	t.Helper()

	e := NewExporter(endpoint, "u", "p")
	e.refresh()

	return gather(t, e)
}

// gather returns metric values keyed by name, plus "{ClusterName}" for per-cluster metrics.
func gather(t *testing.T, e *Exporter) map[string]float64 {
	t.Helper()

	reg := prometheus.NewRegistry()
	reg.MustRegister(e)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	values := map[string]float64{}
	for _, mf := range families {
		for _, m := range mf.GetMetric() {
			key := mf.GetName()
			for _, l := range m.GetLabel() {
				if l.GetName() == "ClusterName" {
					key += "{" + l.GetValue() + "}"
				}
			}
			values[key] = m.GetCounter().GetValue() + m.GetGauge().GetValue()
		}
	}

	return values
}

func TestCollectFetchesAlarmsOnceForAllClusters(t *testing.T) {
	fake := &fakeCmon{alarmsBody: fakeAlarms}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()

	got := collect(t, srv.URL)

	if len(fake.alarmReqs) != 1 {
		t.Fatalf("getAlarms calls = %d, want 1", len(fake.alarmReqs))
	}
	if _, ok := fake.alarmReqs[0]["cluster_id"]; ok {
		t.Errorf("getAlarms request has cluster_id, want none: %v", fake.alarmReqs[0])
	}

	want := map[string]float64{
		"cmon_up":                                1,
		"cmon_cluster_total":                     2,
		"cmon_alarms_critical_total":             3,
		"cmon_alarms_backup_failed_total":        1,
		"cmon_cluster_failed_init_total":         1,
		"cmon_cluster_backup_failed{one}":        1,
		"cmon_cluster_backup_upload_failed{one}": 1,
		"cmon_cluster_failed_init{one}":          0,
		"cmon_cluster_backup_failed{two}":        0,
		"cmon_cluster_backup_upload_failed{two}": 0,
		"cmon_cluster_failed_init{two}":          1,
	}
	for name, value := range want {
		v, ok := got[name]
		if !ok {
			t.Errorf("%s missing", name)
		} else if v != value {
			t.Errorf("%s = %v, want %v", name, v, value)
		}
	}
}

func TestCollectServesLastRefreshWithoutCallingCmon(t *testing.T) {
	fake := &fakeCmon{alarmsBody: fakeAlarms}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()

	e := NewExporter(srv.URL, "u", "p")

	if got := gather(t, e); len(got) != 0 {
		t.Errorf("metrics before the first refresh: %v", got)
	}

	e.refresh()
	first := gather(t, e)
	second := gather(t, e)

	if len(fake.alarmReqs) != 1 {
		t.Errorf("getAlarms calls = %d, want 1 for one refresh and two scrapes", len(fake.alarmReqs))
	}
	if first["cmon_cluster_total"] != 2 || second["cmon_cluster_total"] != 2 {
		t.Errorf("cmon_cluster_total = %v, %v, want 2 on both scrapes", first["cmon_cluster_total"], second["cmon_cluster_total"])
	}
	if _, ok := first["cmon_exporter_refresh_duration_seconds"]; !ok {
		t.Error("cmon_exporter_refresh_duration_seconds missing")
	}
	if first["cmon_exporter_last_refresh_timestamp_seconds"] <= 0 {
		t.Errorf("cmon_exporter_last_refresh_timestamp_seconds = %v, want > 0", first["cmon_exporter_last_refresh_timestamp_seconds"])
	}
}

func TestCollectSkipsAlarmMetricsWhenGetAlarmsFails(t *testing.T) {
	fake := &fakeCmon{alarmsBody: `{"request_status":"UnknownError","error_string":"boom"}`}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()

	got := collect(t, srv.URL)

	if got["cmon_up"] != 1 || got["cmon_cluster_total"] != 2 {
		t.Errorf("cluster metrics missing: up=%v total=%v", got["cmon_up"], got["cmon_cluster_total"])
	}
	for _, name := range []string{
		"cmon_cluster_backup_failed{one}",
		"cmon_cluster_failed_init{two}",
		"cmon_alarms_critical_total",
		"cmon_alarms_backup_failed_total",
		"cmon_cluster_failed_init_total",
	} {
		if _, ok := got[name]; ok {
			t.Errorf("%s reported although getAlarms failed", name)
		}
	}
}

func TestRefreshKeepsOneSessionAndReadsVersionFromEveryReply(t *testing.T) {
	fake := &fakeCmon{alarmsBody: fakeAlarms, version: "2.4.0.1"}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()

	e := NewExporter(srv.URL, "u", "p")
	e.refresh()
	e.refresh()

	if fake.authCalls != 1 {
		t.Errorf("auth calls after two refreshes = %d, want 1", fake.authCalls)
	}

	// CMON was upgraded and restarted with the session kept
	fake.version = "2.4.1.2"
	e.refresh()

	if got := cmonUpVersion(t, e); got != "2.4.1.2" {
		t.Errorf("CmonVersion = %q, want 2.4.1.2", got)
	}

	// the session expired
	fake.session = "expired"
	e.refresh()

	if fake.authCalls != 2 {
		t.Errorf("auth calls after the session expired = %d, want 2", fake.authCalls)
	}
	if got := gather(t, e)["cmon_up"]; got != 1 {
		t.Errorf("cmon_up after re-authentication = %v, want 1", got)
	}
}

func TestRefreshReportsDownWithLastLabelsWhenAuthFails(t *testing.T) {
	fake := &fakeCmon{alarmsBody: fakeAlarms, version: "2.4.0.1"}
	srv := httptest.NewTLSServer(fake)
	defer srv.Close()

	e := NewExporter(srv.URL, "u", "p")
	e.refresh()

	fake.session = "expired"
	e.client.password = "wrong"
	e.refresh()

	got := gather(t, e)
	if got["cmon_up"] != 0 {
		t.Errorf("cmon_up = %v, want 0", got["cmon_up"])
	}
	if v := cmonUpVersion(t, e); v != "2.4.0.1" {
		t.Errorf("CmonVersion = %q, want the last known 2.4.0.1", v)
	}
	if _, ok := got["cmon_cluster_total"]; ok {
		t.Error("cmon_cluster_total reported although CMON failed")
	}
}

func cmonUpVersion(t *testing.T, e *Exporter) string {
	t.Helper()

	reg := prometheus.NewRegistry()
	reg.MustRegister(e)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "cmon_up" {
			continue
		}
		for _, l := range mf.GetMetric()[0].GetLabel() {
			if l.GetName() == "CmonVersion" {
				return l.GetValue()
			}
		}
	}

	return ""
}

func TestScanCoredumpsCountsOnlyCoreFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"core.123", "core.cmon.456", "ab", "notcore.1", "core"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "core.dir"), 0o700); err != nil {
		t.Fatal(err)
	}

	if got := ScanCoredumps(dir); got != 2 {
		t.Errorf("ScanCoredumps = %v, want 2", got)
	}
}
