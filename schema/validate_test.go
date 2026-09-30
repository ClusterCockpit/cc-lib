// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestValidateCluster(t *testing.T) {
	json := []byte(`{
		"name": "emmy",
		"subClusters": [
			{
				"name": "main",
				"processorType": "Intel IvyBridge",
				"socketsPerNode": 2,
				"coresPerSocket": 10,
				"threadsPerCore": 2,
                "flopRateScalar": {
                  "unit": {
                    "prefix": "G",
                    "base": "F/s"
                  },
                  "value": 14
                },
                "flopRateSimd": {
                  "unit": {
                    "prefix": "G",
                    "base": "F/s"
                  },
                  "value": 112
                },
                "memoryBandwidth": {
                  "unit": {
                    "prefix": "G",
                    "base": "B/s"
                  },
                  "value": 24
                },
                "numberOfNodes": 70,
                "nodes": "w11[27-45,49-63,69-72]",
				"topology": {
					"node": [0,20,1,21,2,22,3,23,4,24,5,25,6,26,7,27,8,28,9,29,10,30,11,31,12,32,13,33,14,34,15,35,16,36,17,37,18,38,19,39],
					"socket": [
						[0,20,1,21,2,22,3,23,4,24,5,25,6,26,7,27,8,28,9,29],
						[10,30,11,31,12,32,13,33,14,34,15,35,16,36,17,37,18,38,19,39]
					],
					"memoryDomain": [
						[0,20,1,21,2,22,3,23,4,24,5,25,6,26,7,27,8,28,9,29],
						[10,30,11,31,12,32,13,33,14,34,15,35,16,36,17,37,18,38,19,39]
					],
					"core": [
						[0,20],[1,21],[2,22],[3,23],[4,24],[5,25],[6,26],[7,27],[8,28],[9,29],[10,30],[11,31],[12,32],[13,33],[14,34],[15,35],[16,36],[17,37],[18,38],[19,39]
					]
				}
			}
		],
		"metricConfig": [
			{
				"name": "cpu_load",
				"scope": "hwthread",
				"unit": {"base": ""},
                "aggregation": "avg",
				"timestep": 60,
			    "peak": 4,
                "normal": 2,
                "caution": 1,
                "alert": 0.25
			}
		]
}`)

	if err := Validate(ClusterCfg, bytes.NewReader(json)); err != nil {
		t.Errorf("Error is not nil! %v", err)
	}
}

// deviceClusterJSON returns a minimal cluster.json whose single subcluster
// topology carries the given extra topology members and whose metric has the
// given native scope.
func deviceClusterJSON(topologyExtra, scope string) string {
	return `{
		"name": "fritz",
		"subClusters": [{
			"name": "main",
			"processorType": "Intel Icelake",
			"socketsPerNode": 1,
			"coresPerSocket": 2,
			"threadsPerCore": 1,
			"flopRateScalar": {"unit": {"base": "F/s", "prefix": "G"}, "value": 10},
			"flopRateSimd": {"unit": {"base": "F/s", "prefix": "G"}, "value": 80},
			"memoryBandwidth": {"unit": {"base": "B/s", "prefix": "G"}, "value": 100},
			"nodes": "f[01-02]",
			"topology": {
				"node": [0, 1],
				"socket": [[0, 1]],
				"memoryDomain": [[0, 1]],
				"core": [[0], [1]]` + topologyExtra + `
			}
		}],
		"metricConfig": [{
			"name": "fs_read_bw",
			"scope": "` + scope + `",
			"unit": {"base": "B/s"},
			"aggregation": "sum",
			"timestep": 60,
			"peak": 1000,
			"normal": 100,
			"caution": 10,
			"alert": 1
		}]
	}`
}

func TestValidateCluster_DeviceTopology(t *testing.T) {
	const devices = `,
				"filesystems": [{"id": "/home", "type": "nfs"}, {"id": "/scratch", "type": "lustre"}],
				"networks": [{"id": "ib0", "type": "infiniband"}]`

	tests := []struct {
		name    string
		extra   string
		scope   string
		wantErr bool
	}{
		{"filesystem scope with devices", devices, "filesystem", false},
		{"network scope with devices", devices, "network", false},
		{"no device lists", "", "node", false},
		{"unknown scope", devices, "mountpoint", true},
		{"unknown filesystem type", `, "filesystems": [{"id": "/home", "type": "ntfs"}]`, "filesystem", true},
		{"filesystem without type", `, "filesystems": [{"id": "/home"}]`, "filesystem", true},
		{"unknown network type", `, "networks": [{"id": "eth0", "type": "token-ring"}]`, "network", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(ClusterCfg, bytes.NewReader([]byte(deviceClusterJSON(tt.extra, tt.scope))))
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCluster_DeviceTopologyDecodes(t *testing.T) {
	const devices = `,
				"filesystems": [{"id": "/home", "type": "nfs"}, {"id": "/scratch", "type": "lustre"}],
				"networks": [{"id": "ib0", "type": "infiniband"}]`

	var c Cluster
	if err := json.Unmarshal([]byte(deviceClusterJSON(devices, "filesystem")), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	topo := c.SubClusters[0].Topology
	if got := topo.GetFilesystemIDs(); !reflect.DeepEqual(got, []string{"/home", "/scratch"}) {
		t.Errorf("filesystem ids = %v", got)
	}
	if topo.Filesystems[1].Type != "lustre" {
		t.Errorf("filesystem type = %q, want lustre", topo.Filesystems[1].Type)
	}
	if got := topo.GetNetworkIDs(); !reflect.DeepEqual(got, []string{"ib0"}) {
		t.Errorf("network ids = %v", got)
	}
	if c.MetricConfig[0].Scope != MetricScopeFilesystem {
		t.Errorf("metric scope = %q, want filesystem", c.MetricConfig[0].Scope)
	}
}
