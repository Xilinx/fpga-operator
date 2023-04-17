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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// apiconfigv1 "github.com/openshift/api/config/v1"
	// apiimagev1 "github.com/openshift/api/image/v1"
	// secv1 "github.com/openshift/api/security/v1"
	// promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	policyv1 "github.com/xilinx/fpga-operator/api/v1"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	nfdLabelPrefix         = "feature.node.kubernetes.io/"
	nfdLabelOSReleaseID    = "feature.node.kubernetes.io/system-os_release.ID"
	nfdLabelOSVersionID    = "feature.node.kubernetes.io/system-os_release.VERSION_ID"
	nfdLabelOsMajorVersion = "feature.node.kubernetes.io/system-os_release.VERSION_ID.major"
)

var fpgaNodeLabels = map[string]string{
	// "feature.node.kubernetes.io/pci-10ee.present":      "true",
	"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
}

// ClusterPolicyController represents clusterpolicy controller spec for FPGA operator
type ClusterPolicyController struct {
	singleton         *policyv1.ClusterPolicy
	operatorNamespace string

	resources    []Resources
	controlFuncs []controlFuncs
	stateNames   []string
	rec          *ClusterPolicyReconciler
	idx          int

	k8sVersion string

	runtime      policyv1.Runtime
	osDists      map[string]bool
	hasFPGANodes bool
	hasNFDLabels bool
}

// hasNFDLabels return true if node labels contain NFD labels
func hasNFDLabels(labels map[string]string) bool {
	for key := range labels {
		if strings.HasPrefix(key, nfdLabelPrefix) {
			return true
		}
	}
	return false
}

func hasFPGALables(labels map[string]string) bool {
	for key, val := range labels {
		if _, ok := fpgaNodeLabels[key]; ok {
			if fpgaNodeLabels[key] == val {
				return true
			}
		}
	}
	return false
}

func (ctrl *ClusterPolicyController) getFPGANodeCount() (bool, int, error) {
	// fetch all nodes
	opts := []client.ListOption{}
	nodes := &corev1.NodeList{}
	err := ctrl.rec.Client.List(context.TODO(), nodes, opts...)
	if err != nil {
		return false, 0, fmt.Errorf("unable to list nodes to check labels, err %s", err.Error())
	}

	clusterHasNFDLabels := false
	fpgaNodesTotal := 0
	for _, node := range nodes.Items {
		// get node labels
		labels := node.GetLabels()
		if !clusterHasNFDLabels {
			clusterHasNFDLabels = hasNFDLabels(labels)
		}
		if hasFPGALables(labels) {
			fpgaNodesTotal++
			ctrl.rec.Log.Info("Node has FGPA(s)", "NodeName", node.ObjectMeta.Name)
		}
	}

	return clusterHasNFDLabels, fpgaNodesTotal, nil
}

func (ctrl *ClusterPolicyController) getOsDistributions() (map[string]bool, error) {
	// fetch all nodes
	opts := []client.ListOption{}
	nodes := &corev1.NodeList{}
	err := ctrl.rec.Client.List(context.TODO(), nodes, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to list nodes to check labels, err %s", err.Error())
	}

	dists := make(map[string]bool)
	for _, node := range nodes.Items {
		// get node labels
		labels := node.GetLabels()
		release := labels[nfdLabelOSReleaseID]
		version := labels[nfdLabelOSVersionID]
		dists[release+version] = true
	}
	return dists, nil
}

func addState(ctrl *ClusterPolicyController, path string) error {
	res, ctrlFunc := addResourceControls(ctrl, path)
	ctrl.resources = append(ctrl.resources, res)
	ctrl.controlFuncs = append(ctrl.controlFuncs, ctrlFunc)
	ctrl.stateNames = append(ctrl.stateNames, filepath.Base(path))
	return nil
}

// init adds all the states declared in specs
func (ctrl *ClusterPolicyController) init(reconciler *ClusterPolicyReconciler, clusterPolicy *policyv1.ClusterPolicy) error {
	ctrl.singleton = clusterPolicy
	ctrl.rec = reconciler
	ctrl.idx = 0

	if len(ctrl.controlFuncs) == 0 {
		ctrl.operatorNamespace = os.Getenv("OPERATOR_NAMESPACE")
		if clusterPolicyCtrl.operatorNamespace == "" {
			ctrl.rec.Log.Info("OPERATOR_NAMESPACE environment variable not set, using default")
			ctrl.operatorNamespace = "default"
		}

		k8sVersion, err := kubernetesVersion()
		if err != nil {
			return err
		}
		if !semver.IsValid(k8sVersion) {
			return fmt.Errorf("k8s version detected '%s' is not a valid semantic version", k8sVersion)
		}
		ctrl.k8sVersion = k8sVersion
		ctrl.rec.Log.Info("Kubernetes version detected", "version", k8sVersion)

		// detect the container runtime on worker nodes
		err = ctrl.getRuntime()
		if err != nil {
			return err
		}
		ctrl.rec.Log.Info(fmt.Sprintf("Using container runtime: %s", ctrl.runtime.String()))

		// add components
		addState(ctrl, "/opt/fpga-operator/state-container-runtime")
		addState(ctrl, "/opt/fpga-operator/state-device-plugin")
		addState(ctrl, "/opt/fpga-operator/state-host-setup")
	}

	hasNFDLabels, fpgaNodeCount, err := ctrl.getFPGANodeCount()
	if err != nil {
		return err
	}
	ctrl.hasNFDLabels = hasNFDLabels
	ctrl.hasFPGANodes = fpgaNodeCount > 0

	osDists, err := ctrl.getOsDistributions()
	if err != nil {
		return err
	}
	ctrl.osDists = osDists
	return nil
}

func (ctrl *ClusterPolicyController) step() (policyv1.State, error) {
	result := policyv1.Ready

	for _, fs := range ctrl.controlFuncs[ctrl.idx] {
		state, err := fs(*ctrl)
		if err != nil {
			// failed to deploy resource
			return state, err
		}
		// deployed resource successfully, check the state
		if state != policyv1.Ready {
			result = state
		}
	}

	// move to next state
	ctrl.idx = ctrl.idx + 1
	return result, nil
}

func (ctrl ClusterPolicyController) last() bool {
	return ctrl.idx == len(ctrl.controlFuncs)
}

func (n ClusterPolicyController) isStateEnabled(stateName string) bool {
	clusterPolicySpec := &n.singleton.Spec

	switch stateName {
	case "state-container-runtime":
		return clusterPolicySpec.ContainerRuntime.IsEnabled()
	case "state-device-plugin":
		return clusterPolicySpec.DevicePlugin.IsEnabled()
	case "state-host-setup":
		return clusterPolicySpec.HostSetup.IsEnabled()
	default:
		n.rec.Log.Error(nil, "invalid state passed", "stateName", stateName)
		return false
	}
}

// kubernetesVersion fetches the Kubernetes API server version
func kubernetesVersion() (string, error) {
	cfg := config.GetConfigOrDie()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("error building discovery client: %v", err)
	}

	info, err := discoveryClient.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("unable to fetch server version information: %v", err)
	}

	return info.GitVersion, nil
}

func getRuntimeString(node corev1.Node) (policyv1.Runtime, error) {
	// ContainerRuntimeVersion string will look like <runtime>://<x.y.z>
	runtimeVer := node.Status.NodeInfo.ContainerRuntimeVersion
	var runtime policyv1.Runtime
	if strings.HasPrefix(runtimeVer, "docker") {
		runtime = policyv1.Docker
	} else if strings.HasPrefix(runtimeVer, "containerd") {
		runtime = policyv1.Containerd
	} else {
		return "", fmt.Errorf("runtime not recognized: %s", runtimeVer)
	}
	return runtime, nil
}

// getRuntime will detect the container runtime used by nodes in the
// cluster and correctly set the value for clusterPolicyController.runtime
// The default runtime is containerd -- if >=1 node is configured with containerd, set
// clusterPolicyController.runtime = containerd
func (ctrl *ClusterPolicyController) getRuntime() error {

	list := &corev1.NodeList{}
	err := ctrl.rec.Client.List(context.TODO(), list)
	if err != nil {
		return fmt.Errorf("unable to list nodes prior to checking container runtime: %v", err)
	}

	var runtime policyv1.Runtime
	for _, node := range list.Items {
		rt, err := getRuntimeString(node)
		if err != nil {
			ctrl.rec.Log.Info(fmt.Sprintf("Unable to get runtime info for node %s: %v", node.Name, err))
			continue
		}
		runtime = rt
		if runtime == policyv1.Containerd {
			// default to containerd if >=1 node running containerd
			break
		}
	}

	if runtime.String() == "" {
		ctrl.rec.Log.Info("Unable to get runtime info from the cluster, using default")
		runtime = ctrl.singleton.Spec.Operator.DefaultRuntime
	}
	ctrl.runtime = runtime
	return nil
}
