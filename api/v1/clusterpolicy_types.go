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

package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// Docker runtime
	Docker Runtime = "docker"
	// Containerd runtime
	Containerd Runtime = "containerd"
)

// Runtime defines container runtime type
type Runtime string

func (r Runtime) String() string {
	switch r {
	case Docker:
		return "docker"
	case Containerd:
		return "containerd"
	default:
		return ""
	}
}

type OperatorSpec struct {
	// +kubebuilder:validation:Enum=docker;containerd
	// +kubebuilder:default=containerd
	DefaultRuntime Runtime `json:"defaultRuntime"`
}

type ContainerRuntimeSpec struct {
	// Enabled indicates if deployment of Xilinx Container Toolkit through operator is enabled
	Enabled *bool `json:"enabled,omitempty"`

	// +kubebuilder:default=xilinx
	RuntimeClass string `json:"runtimeClass,omitempty"`

	// set as default
	SetAsDefault *bool `json:"setAsDefault,omitempty"`

	// Xilinx Container Toolkit image repo
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// Xilinx Container Toolkit image name
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Image string `json:"image,omitempty"`

	// Xilinx Container toolkit image tag
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`

	// Image pull policy
	// +kubebuilder:validation:Optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Image pull secrets
	// +kubebuilder:validation:Optional
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Optional: List of arguments
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables
	Env []corev1.EnvVar `json:"env,omitempty"`

	// XCR install directory on the host
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=/usr/bin
	InstallDir string `json:"installDir,omitempty"`

	// // Docker or Containerd config file
	// RuntimeConfig string `json:"runtimeConfig,omitempty"`

	// // Docker or Containerd socket file
	// RuntimeSocket string `json:"runtimeSocket,omitempty"`
}

type DevicePluginSpec struct {
	// Enabled indicates if deployment of Xilinx Container Toolkit through operator is enabled
	Enabled *bool `json:"enabled,omitempty"`

	// device-plugin image repo
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// device-plugin image name
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Image string `json:"image,omitempty"`

	// device-plugin image tag
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`

	// Image pull policy
	// +kubebuilder:validation:Optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Image pull secrets
	// +kubebuilder:validation:Optional
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Optional: List of environment variables
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type OsDistSetupSpec struct {
	// OS distribution, eg. ubuntu, centos
	// +kubebuilder:validation:Enum=ubuntu;centos;amzn;rhel
	OsId string `json:"osId"`

	// OS major version, eg. 18, 20
	OsMajorVersion string `json:"osMajorVersion"`

	// The version to be setup
	Version string `json:"version,omitempty"`

	// XRM installation enabled
	XrmInstallation *bool `json:"xrmInstallation,omitempty"`

	// shell flash enabled
	ShellFlashEnabled *bool `json:"shellFlashEnabled,omitempty"`

	// Cards on the host, empty to perform setup for all cards
	// +kubebuilder:validation:Optional
	Cards []string `json:"cards"`

	// host-setup image repo
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// host-setup image name
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Image string `json:"image,omitempty"`

	// host-setup image tag
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`

	// Image pull policy
	// +kubebuilder:validation:Optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Image pull secrets
	// +kubebuilder:validation:Optional
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

type HostSetupSpec struct {
	// Enabled indicates if deployment of Xilinx Container Toolkit through operator is enabled
	Enabled *bool `json:"enabled,omitempty"`

	// Setup per os distributions, eg. ubuntu18, ubuntu20
	OsDists []OsDistSetupSpec `json:"osDists"`
}

// ClusterPolicySpec defines the desired state of ClusterPolicy
type ClusterPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Operator component spec
	Operator OperatorSpec `json:"operator"`

	// ContainerRuntime component spec
	ContainerRuntime ContainerRuntimeSpec `json:"containerRuntime"`

	// DevicePlugin component spec
	DevicePlugin DevicePluginSpec `json:"devicePlugin"`

	// HostSetup component spec
	HostSetup HostSetupSpec `json:"hostSetup"`
}

// State indicates state of GPU operator components
type State string

const (
	// Ignored indicates duplicate ClusterPolicy instances and rest are ignored.
	Ignored State = "ignored"
	// Ready indicates all components of ClusterPolicy are ready
	Ready State = "ready"
	// NotReady indicates some/all components of ClusterPolicy are not ready
	NotReady State = "notReady"
	// Disabled indicates if the state is disabled
	Disabled State = "disabled"
)

// ClusterPolicyStatus defines the observed state of ClusterPolicy
type ClusterPolicyStatus struct {
	// +kubebuilder:validation:Enum=ignored;ready;notReady;disabled
	// State indicates status of ClusterPolicy
	State State `json:"state"`
	// Namespace indicates a namespace in which the operator is installed
	Namespace string `json:"namespace,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ClusterPolicy is the Schema for the clusterpolicies API
type ClusterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPolicySpec   `json:"spec,omitempty"`
	Status ClusterPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterPolicyList contains a list of ClusterPolicy
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterPolicy{}, &ClusterPolicyList{})
}

// SetStatus sets state and namespace of ClusterPolicy instance
func (p *ClusterPolicy) SetStatus(s State, ns string) {
	p.Status.State = s
	p.Status.Namespace = ns
}

func (crs *ContainerRuntimeSpec) IsEnabled() bool {
	if crs.Enabled == nil {
		return true
	}
	return *crs.Enabled
}

func (dps *DevicePluginSpec) IsEnabled() bool {
	if dps.Enabled == nil {
		return true
	}
	return *dps.Enabled
}

func (hss *HostSetupSpec) IsEnabled() bool {
	if hss.Enabled == nil {
		return true
	}
	return *hss.Enabled
}

func ImagePath(repo string, image string, tag string) string {
	var imagePath string
	if repo == "" && tag == "" {
		if image != "" {
			imagePath = image
		}
	} else {
		// use @ if image digest is specified instead of tag
		if strings.HasPrefix(tag, "sha256:") {
			imagePath = repo + "/" + image + "@" + tag
		} else {
			imagePath = repo + "/" + image + ":" + tag
		}
	}
	return imagePath
}

// ImagePullPolicy sets image pull policy
func ImagePullPolicy(pullPolicy string) corev1.PullPolicy {
	var imagePullPolicy corev1.PullPolicy
	switch pullPolicy {
	case "Always":
		imagePullPolicy = corev1.PullAlways
	case "Never":
		imagePullPolicy = corev1.PullNever
	case "IfNotPresent":
		imagePullPolicy = corev1.PullIfNotPresent
	default:
		imagePullPolicy = corev1.PullIfNotPresent
	}
	return imagePullPolicy
}
