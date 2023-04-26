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
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/bombsimon/logrusr/v3"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
	policyv1 "github.com/xilinx/fpga-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterPolicyPath           = "config/samples/policy_v1_clusterpolicy.yaml"
	clusterPolicyName           = "fpga-clusterpolicy"
	devicePluginAssestsPath     = "assets/state-device-plugin"
	containerRuntimeAssestsPath = "assets/state-container-runtime"
	hostSetupAssestsPath        = "assets/state-host-setup"
)

type testConfig struct {
	root      string
	nodeCount int
}

type commonDaemonsetSpec struct {
	repository       string
	image            string
	tag              string
	imagePullPolicy  string
	imagePullSecrets []corev1.LocalObjectReference
	env              []corev1.EnvVar
	resources        *corev1.ResourceRequirements
}

var (
	cfg                     *testConfig
	clusterPolicyController ClusterPolicyController
	clusterPolicyReconciler ClusterPolicyReconciler
	clusterPolicy           policyv1.ClusterPolicy
	boolTrue                *bool
	boolFalse               *bool
)

var nfdLabels = map[string]string{
	"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
	nfdLabelOSReleaseID:    "ubuntu",
	nfdLabelOsMajorVersion: "18",
}

var nodes = []corev1.Node{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ubuntu18",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "ubuntu",
				nfdLabelOsMajorVersion: "18",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ubuntu20",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "ubuntu",
				nfdLabelOsMajorVersion: "20",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ubuntu22",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "ubuntu",
				nfdLabelOsMajorVersion: "22",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "al2",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "amzn",
				nfdLabelOsMajorVersion: "2",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "centos7",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "centos",
				nfdLabelOsMajorVersion: "7",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "centos8",
			Labels: map[string]string{
				"feature.node.kubernetes.io/pci-1200_10ee.present": "true",
				nfdLabelOSReleaseID:    "centos",
				nfdLabelOsMajorVersion: "8",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	},
}

var kubernetesResources = []client.Object{
	&appsv1.DaemonSet{},
	&nodev1.RuntimeClass{},
}

func getModuleRoot(dir string) (string, error) {
	if dir == "" || dir == "/" {
		return "", fmt.Errorf("module root not found")
	}

	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err != nil {
		return getModuleRoot(filepath.Dir(dir))
	}

	// go.mod was found in dir
	return dir, nil
}

// newCluster creates a mock kubernetes cluster and returns the corresponding client object
func newCluster(nodeCount int, s *runtime.Scheme) (client.Client, error) {
	// Build fake client
	cl := fake.NewClientBuilder().WithScheme(s).Build()

	for i := 0; i < nodeCount; i++ {

		err := cl.Create(context.TODO(), &nodes[i])
		if err != nil {
			return nil, fmt.Errorf("unable to create node in cluster: %v", err)
		}
	}

	return cl, nil
}

// setup creates a mock kubernetes cluster and client. Nodes are labeled with the minumum
// required NFD labels to be detected as FPGA nodes by the FPGA Operator. A sample
// ClusterPolicy resource is applied to the cluster. The ClusterPolicyController
// object is initialized with the mock kubernetes client as well as other steps
// mimicking init() in state_manager.go
func setup() error {
	// Used when updating ClusterPolicy spec
	boolFalse = new(bool)
	boolTrue = new(bool)
	*boolTrue = true

	// add env for calls that we cannot mock
	os.Setenv("UNIT_TEST", "true")

	s := scheme.Scheme
	if err := policyv1.AddToScheme(s); err != nil {
		return fmt.Errorf("unable to add ClusterPolicy v1 schema: %v", err)
	}

	client, err := newCluster(cfg.nodeCount, s)
	if err != nil {
		return fmt.Errorf("unable to create cluster: %v", err)
	}

	// Get a sample ClusterPolicy manifest
	manifests := getAssetsFrom(&clusterPolicyController, filepath.Join(cfg.root, clusterPolicyPath))
	clusterPolicyManifest := manifests[0]
	ser := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	_, _, err = ser.Decode(clusterPolicyManifest, nil, &clusterPolicy)
	if err != nil {
		return fmt.Errorf("failed to decode sample ClusterPolicy manifest: %v", err)
	}

	err = client.Create(context.TODO(), &clusterPolicy)
	if err != nil {
		return fmt.Errorf("failed to create ClusterPolicy resource: %v", err)
	}

	// Confirm ClusterPolicy is deployed in mock cluster
	cp := &policyv1.ClusterPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: clusterPolicyName}, cp)
	if err != nil {
		return fmt.Errorf("unable to get ClusterPolicy from client: %v", err)
	}

	logger := logrusr.New(logrus.New())
	ctrl.SetLogger(logger)

	clusterPolicyReconciler = ClusterPolicyReconciler{
		Client: client,
		Log:    ctrl.Log.WithName("controller").WithName("ClusterPolicy"),
		Scheme: s,
	}

	clusterPolicyController = ClusterPolicyController{
		singleton: cp,
		rec:       &clusterPolicyReconciler,
	}

	hasNFDLabels, fpgaNodeCount, err := clusterPolicyController.getFPGANodeCount()
	if err != nil {
		return fmt.Errorf("unable to label nodes in cluster: %v", err)
	}
	if fpgaNodeCount == 0 {
		return fmt.Errorf("no gpu nodes in mock cluster")
	}

	clusterPolicyController.hasFPGANodes = fpgaNodeCount != 0
	clusterPolicyController.hasNFDLabels = hasNFDLabels

	return nil
}

func TestMain(m *testing.M) {
	_, filename, _, _ := goruntime.Caller(0)
	moduleRoot, err := getModuleRoot(filename)
	if err != nil {
		log.Fatalf("error in test setup: could not get module root: %v", err)
	}
	cfg = &testConfig{root: moduleRoot, nodeCount: 6}

	err = setup()
	if err != nil {
		log.Fatalf("error in test setup: could not setup mock k8s: %v", err)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

// updateClusterPolicy updates an existing ClusterPolicy instance
func updateClusterPolicy(n *ClusterPolicyController, cp *policyv1.ClusterPolicy) error {
	n.singleton = cp
	err := n.rec.Client.Update(context.TODO(), cp)
	if err != nil && !errors.IsConflict(err) {
		return fmt.Errorf("failed to update ClusterPolicy: %v", err)
	}
	return nil
}

// removeState deletes all resources, controls, and stateNames tracked
// by ClusterPolicyController at a specific index. It also deletes
// all objects from the mock k8s client
func removeState(n *ClusterPolicyController, idx int) error {
	var err error
	for _, res := range kubernetesResources {
		// TODO: use n.operatorNamespace once MR is merged
		err = n.rec.Client.DeleteAllOf(context.TODO(), res)
		if err != nil {
			return fmt.Errorf("error deleting objects from k8s client: %v", err)
		}
	}
	n.resources = append(n.resources[:idx], n.resources[idx+1:]...)
	n.controlFuncs = append(n.controlFuncs[:idx], n.controlFuncs[idx+1:]...)
	n.stateNames = append(n.stateNames[:idx], n.stateNames[idx+1:]...)
	return nil
}

// getImagePullSecrets converts a slice of strings (pull secrets)
// to the corev1 type used by k8s
func getImagePullSecrets(secrets []string) []corev1.LocalObjectReference {
	var ret []corev1.LocalObjectReference
	for _, secret := range secrets {
		ret = append(ret, corev1.LocalObjectReference{Name: secret})
	}
	return ret
}

func testDaemonsetCommon(t *testing.T, cp *policyv1.ClusterPolicy, component string, numDaemonsets int) ([]appsv1.DaemonSet, error) {
	var spec commonDaemonsetSpec
	var dsLabel, mainCtrName, manifestFile string
	var err error

	switch component {
	case "DevicePlugin":
		spec = commonDaemonsetSpec{
			repository:       cp.Spec.DevicePlugin.Repository,
			image:            cp.Spec.DevicePlugin.Image,
			tag:              cp.Spec.DevicePlugin.Tag,
			imagePullPolicy:  cp.Spec.DevicePlugin.ImagePullPolicy,
			imagePullSecrets: getImagePullSecrets(cp.Spec.DevicePlugin.ImagePullSecrets),
			env:              cp.Spec.DevicePlugin.Env,
		}
		dsLabel = "device-plugin"
		mainCtrName = "device-plugin"
		manifestFile = filepath.Join(cfg.root, devicePluginAssestsPath)
	case "ContainerRuntime":
		spec = commonDaemonsetSpec{
			repository:       cp.Spec.ContainerRuntime.Repository,
			image:            cp.Spec.ContainerRuntime.Image,
			tag:              cp.Spec.ContainerRuntime.Tag,
			imagePullPolicy:  cp.Spec.ContainerRuntime.ImagePullPolicy,
			imagePullSecrets: getImagePullSecrets(cp.Spec.ContainerRuntime.ImagePullSecrets),
			env:              cp.Spec.ContainerRuntime.Env,
		}
		dsLabel = "xilinx-container-runtime"
		mainCtrName = "xilinx-container-runtime"
		manifestFile = filepath.Join(cfg.root, containerRuntimeAssestsPath)
	case "HostSetup":
		spec = commonDaemonsetSpec{
			repository:       cp.Spec.HostSetup.OsDists[0].Repository,
			image:            cp.Spec.HostSetup.OsDists[0].Image,
			tag:              cp.Spec.HostSetup.OsDists[0].Tag,
			imagePullPolicy:  cp.Spec.HostSetup.OsDists[0].ImagePullPolicy,
			imagePullSecrets: getImagePullSecrets(cp.Spec.HostSetup.OsDists[0].ImagePullSecrets),
		}
		dsLabel = "host-setup"
		mainCtrName = "host-setup-ubuntu"
		manifestFile = filepath.Join(cfg.root, hostSetupAssestsPath)

	default:
		return nil, fmt.Errorf("invalid component for testDaemonsetCommon(): %s", component)
	}

	// update cluster policy
	err = updateClusterPolicy(&clusterPolicyController, cp)
	if err != nil {
		t.Fatalf("error in test setup: %v", err)
	}
	// add manifests
	err = addState(&clusterPolicyController, manifestFile)
	if err != nil {
		t.Fatalf("unable to add state: %v", err)
	}
	// create resources
	_, err = clusterPolicyController.step()
	if err != nil {
		t.Errorf("error creating resources: %v", err)
	}
	// get daemonsets
	opts := []client.ListOption{
		client.MatchingLabels{"app": dsLabel},
	}
	list := &appsv1.DaemonSetList{}
	err = clusterPolicyController.rec.Client.List(context.TODO(), list, opts...)
	if err != nil {
		t.Fatalf("could not get DaemonSetList from client: %v", err)
	}

	// compare daemonset with expected output
	require.Equal(t, numDaemonsets, len(list.Items), "unexpected # of daemonsets")
	if numDaemonsets == 0 || len(list.Items) == 0 {
		return nil, nil
	}

	mainCtrIdx := -1
	var mainCtr corev1.Container

	for _, ds := range list.Items {
		if mainCtrIdx != -1 {
			break
		}
		for i, container := range ds.Spec.Template.Spec.Containers {
			if strings.Contains(container.Name, mainCtrName) {
				mainCtrIdx = i
				mainCtr = ds.Spec.Template.Spec.Containers[mainCtrIdx]
				break
			}
		}

	}

	if mainCtrIdx == -1 {
		return nil, fmt.Errorf("could not find main container index")
	}

	require.Equal(t, policyv1.ImagePullPolicy(spec.imagePullPolicy), mainCtr.ImagePullPolicy, "unexpected ImagePullPolicy")
	for _, env := range spec.env {
		require.Contains(t, mainCtr.Env, env, "env var not present")
	}
	return list.Items, nil
}

func getDevicePluginTestInput(testCase string) *policyv1.ClusterPolicy {
	// default cluster policy
	cp := clusterPolicy.DeepCopy()

	cp.Spec.DevicePlugin.Repository = "public.ecr.aws/xilinx_dcg"
	cp.Spec.DevicePlugin.Image = "k8s-device-plugin"
	cp.Spec.DevicePlugin.Tag = "1.1.0"

	switch testCase {
	case "default":
		// Do nothing
	default:
		return nil
	}
	return cp
}

func getDevicePluginTestOutput(testCase string) map[string]interface{} {
	// default output
	output := map[string]interface{}{
		"numDaemonsets": 1,
		"image":         "public.ecr.aws/xilinx_dcg/k8s-device-plugin:1.1.0",
	}

	return output
}

func TestDevicePlugin(t *testing.T) {
	testCases := []struct {
		description   string
		clusterpolicy *policyv1.ClusterPolicy
		output        map[string]interface{}
	}{
		{
			"default",
			getDevicePluginTestInput("default"),
			getDevicePluginTestOutput("default"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dsList, err := testDaemonsetCommon(t, tc.clusterpolicy, "DevicePlugin", tc.output["numDaemonsets"].(int))
			if err != nil {
				t.Fatalf("error in testDaemonsetCommon(): %v", err)
			}
			if dsList == nil {
				return
			}

			image := dsList[0].Spec.Template.Spec.Containers[0].Image
			require.Equal(t, tc.output["image"], image, "Unexpected configuration for device-plugin image")

			// cleanup by deleting all kubernetes objects
			err = removeState(&clusterPolicyController, clusterPolicyController.idx-1)
			if err != nil {
				t.Fatalf("error removing state: %v", err)
			}
			clusterPolicyController.idx--
		})
	}
}

func getContainerRuntimeTestInput(testCase string) *policyv1.ClusterPolicy {
	// default cluster policy
	cp := clusterPolicy.DeepCopy()
	cp.Spec.Operator.DefaultRuntime = policyv1.Containerd
	cp.Spec.ContainerRuntime.Repository = "xilinxatg"
	cp.Spec.ContainerRuntime.Image = "xilinx-container-runtime"
	cp.Spec.ContainerRuntime.Tag = "ubuntu18.04"

	switch testCase {
	case "default":
		// Do nothing
	default:
		return nil
	}

	return cp
}

func getContainerRuntimeTestOutput(testCase string) map[string]interface{} {
	// default output
	output := map[string]interface{}{
		"numDaemonSets": 1,
		"image":         "xilinxatg/xilinx-container-runtime:ubuntu18.04",
	}

	switch testCase {
	case "default":
		// Do nothing
	default:
		return nil
	}
	return output
}

func TestContainerRuntime(t *testing.T) {
	testCases := []struct {
		description   string
		clusterpolicy *policyv1.ClusterPolicy
		output        map[string]interface{}
	}{
		{
			"default",
			getContainerRuntimeTestInput("default"),
			getContainerRuntimeTestOutput("default"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dsList, err := testDaemonsetCommon(t, tc.clusterpolicy, "ContainerRuntime", tc.output["numDaemonSets"].(int))
			if err != nil {
				t.Fatalf("error in testDaemonsetCommon(): %v", err)
			}
			if dsList == nil {
				return
			}

			image := dsList[0].Spec.Template.Spec.Containers[0].Image
			require.Equal(t, tc.output["image"], image, "Unexpected configuration for container-runtime image")

			// cleanup by deleting all kubernetes objects
			err = removeState(&clusterPolicyController, clusterPolicyController.idx-1)
			if err != nil {
				t.Fatalf("error removing state: %v", err)
			}
			clusterPolicyController.idx--
		})
	}
}

func getHostSetupTestInput(testCase string) *policyv1.ClusterPolicy {
	// default cluster policy
	cp := clusterPolicy.DeepCopy()

	switch testCase {
	case "default":
		// Do nothing
	default:
		return nil
	}

	return cp
}

func getHostSetupTestOutput(testCase string) map[string]interface{} {
	// default output
	output := map[string]interface{}{
		"numDaemonSets": 4,
	}

	switch testCase {
	case "default":
		// Do nothing
	default:
		return nil
	}

	return output
}

func TestHostSetup(t *testing.T) {
	testCases := []struct {
		description   string
		clusterpolicy *policyv1.ClusterPolicy
		output        map[string]interface{}
	}{
		{
			"default",
			getHostSetupTestInput("default"),
			getHostSetupTestOutput("default"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dsList, err := testDaemonsetCommon(t, tc.clusterpolicy, "HostSetup", tc.output["numDaemonSets"].(int))
			if err != nil {
				t.Fatalf("error in testDaemonsetCommon(): %v", err)
			}
			if dsList == nil {
				return
			}

			// cleanup by deleting all kubernetes objects
			err = removeState(&clusterPolicyController, clusterPolicyController.idx-1)
			if err != nil {
				t.Fatalf("error removing state: %v", err)
			}
			clusterPolicyController.idx--
		})
	}
}
