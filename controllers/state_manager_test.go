/*
Copyright (C) 2023, Advanced Micro Devices, Inc. - All rights reserved

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"testing"

	policyv1 "github.com/xilinx/fpga-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestGetRuntimeString(t *testing.T) {
	testCases := []struct {
		description     string
		runtimeVer      string
		expectedRuntime policyv1.Runtime
	}{
		{
			"containerd",
			"containerd://1.0.0",
			policyv1.Containerd,
		},
		{
			"docker",
			"docker://1.0.0",
			policyv1.Docker,
		},
		{
			"unknown",
			"unknown://1.0.0",
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			node := corev1.Node{
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						ContainerRuntimeVersion: tc.runtimeVer,
					},
				},
			}
			runtime, _ := getRuntimeString(node)
			if runtime != tc.expectedRuntime {
				t.Errorf("expected %s but got %s", tc.expectedRuntime.String(), runtime.String())
			}
		})
	}
}
