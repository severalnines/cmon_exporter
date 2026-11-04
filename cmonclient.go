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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const requestStatusOk = "Ok"

// cmonClient is a minimal CMON RPCv2 client. It keeps one session and logs in
// again only when CMON asks for it.
type cmonClient struct {
	endpoint, username, password string

	http *http.Client
	// version is the CMON version from the Server header of the last reply.
	version string
}

type cmonCluster struct {
	ClusterID   uint64 `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	State       string `json:"state"`
}

type cmonAlarm struct {
	ClusterID    uint64 `json:"cluster_id"`
	SeverityName string `json:"severity_name"`
	TypeName     string `json:"type_name"`
}

type cmonReply struct {
	RequestStatus string `json:"request_status"`
	ErrorString   string `json:"error_string"`
	ControllerID  string `json:"controller_id"`
}

type cmonClustersReply struct {
	cmonReply
	Clusters []cmonCluster `json:"clusters"`
}

type cmonAlarmsReply struct {
	cmonReply
	Alarms []cmonAlarm `json:"alarms"`
}

func newCmonClient(endpoint, username, password string, timeout time.Duration) *cmonClient {
	jar, _ := cookiejar.New(nil) // never fails with nil options

	return &cmonClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		username: username,
		password: password,
		http: &http.Client{
			Timeout: timeout,
			Jar:     jar,
			Transport: &http.Transport{
				// CMON listens on 127.0.0.1 with a self-signed certificate
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
	}
}

// getAllClusterInfo returns the clusters and the controller ID.
func (c *cmonClient) getAllClusterInfo() ([]cmonCluster, string, error) {
	var reply cmonClustersReply
	err := c.call("clusters", map[string]any{
		"operation":          "getAllClusterInfo",
		"with_hosts":         false,
		"with_license_check": false,
	}, &reply)

	return reply.Clusters, reply.ControllerID, err
}

// getAlarms returns the alarms of every cluster the user can read.
func (c *cmonClient) getAlarms() ([]cmonAlarm, error) {
	var reply cmonAlarmsReply
	err := c.call("alarm", map[string]any{"operation": "getAlarms"}, &reply)

	return reply.Alarms, err
}

func (c *cmonClient) authenticate() error {
	var reply cmonReply

	return c.post("auth", map[string]any{
		"operation": "authenticateWithPassword",
		"user_name": c.username,
		"password":  c.password,
	}, &reply)
}

// call posts the request, logging in and retrying once when CMON has no session for us.
func (c *cmonClient) call(module string, req any, reply interface{ status() *cmonReply }) error {
	err := c.post(module, req, reply)
	if reply.status().RequestStatus != "AuthRequired" {
		return err
	}

	if err := c.authenticate(); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	return c.post(module, req, reply)
}

func (c *cmonClient) post(module string, req any, reply interface{ status() *cmonReply }) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.http.Post(c.endpoint+"/v2/"+module, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if server := strings.SplitN(resp.Header.Get("Server"), "/", 2); len(server) == 2 {
		c.version = strings.TrimSpace(server[1])
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	*reply.status() = cmonReply{}
	if err := json.Unmarshal(data, reply); err != nil {
		return fmt.Errorf("%s: HTTP %d: %w", module, resp.StatusCode, err)
	}

	if st := reply.status(); st.RequestStatus != requestStatusOk {
		return fmt.Errorf("%s: %s: %s", module, st.RequestStatus, st.ErrorString)
	}

	return nil
}

func (r *cmonReply) status() *cmonReply { return r }
