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
	"path"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure"

	policyv1 "github.com/xilinx/fpga-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultRuntimeClass     = "xilinx"
	DefaultXCRInstallDir    = "/usr/bin"
	DefaultXCRConfigDir     = "/etc/xilinx-container-runtime"
	DefaultDockerConfig     = "/etc/docker/daemon.json"
	DefaultDockerSocket     = "/var/run/docker.sock"
	DefaultContainerdConfig = "/etc/containerd/config.toml"
	DefaultContainerdSocket = "/var/run/containerd/containerd.sock"
	XilinxAnnotationHashKey = "xilinx.com/last-applied-hash"
)

// Error to state spec not found for a daemonset
type NoSpecError struct{}

func (e *NoSpecError) Error() string {
	return "no spec found"
}

type controlFuncs []func(ctrl ClusterPolicyController) (policyv1.State, error)

// getRuntimeClass return the name of runtime class to be created
func getRuntimeClass(config *policyv1.ClusterPolicySpec) string {
	if config.ContainerRuntime.RuntimeClass != "" {
		return config.ContainerRuntime.RuntimeClass
	}
	return DefaultRuntimeClass
}

// RuntimeClass creates RuntimeClass object
func RuntimeClass(n ClusterPolicyController) (policyv1.State, error) {
	// get runtimeclass template
	obj := n.resources[n.idx].RuntimeClass.DeepCopy()

	// apply runtime class name as per ClusterPolicy
	obj.Name = getRuntimeClass(&n.singleton.Spec)
	obj.Handler = getRuntimeClass(&n.singleton.Spec)

	logger := n.rec.Log.WithValues("RuntimeClass", obj.Name)

	// check if state is disabled
	if !n.isStateEnabled(n.stateNames[n.idx]) {
		logger.Info("State disabled, not creating resource")
		err := n.rec.Client.Delete(context.TODO(), obj)
		if err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Couldn't delete")
			return policyv1.NotReady, nil
		}
		return policyv1.Disabled, nil
	}

	// set controller reference
	if err := controllerutil.SetControllerReference(n.singleton, obj, n.rec.Scheme); err != nil {
		return policyv1.NotReady, err
	}

	found := &nodev1.RuntimeClass{}

	// create a new runtimeclass
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating...")
		err = n.rec.Client.Create(context.TODO(), obj)
		if err != nil {
			logger.Error(err, "Couldn't create")
			return policyv1.NotReady, err
		}
		return policyv1.Ready, nil
	} else if err != nil {
		return policyv1.NotReady, err
	}

	// update an existing runtimeclass
	logger.Info("Found Resource, updating...")
	obj.ResourceVersion = found.ResourceVersion

	err = n.rec.Client.Update(context.TODO(), obj)
	if err != nil {
		logger.Error(err, "Couldn't update")
		return policyv1.NotReady, err
	}
	return policyv1.Ready, nil
}

func getDaemonsetHash(daemonset *appsv1.DaemonSet) string {
	hash, err := hashstructure.Hash(daemonset, nil)
	if err != nil {
		panic(err.Error())
	}
	return strconv.FormatUint(hash, 16)
}

func isDaemonSetReady(namespace string, name string, rec ClusterPolicyReconciler, logger logr.Logger) policyv1.State {
	logger.Info("Check DaemonSet ready")
	ds := &appsv1.DaemonSet{}
	err := rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, ds)

	if err != nil {
		rec.Log.Error(err, "Could not get DaemonSet")
		return policyv1.NotReady
	}

	// ds := list.Items[0]
	logger.Info(fmt.Sprintf("Daemonset has %d pod(s) unavailable", ds.Status.NumberUnavailable))

	if ds.Status.NumberUnavailable != 0 {
		return policyv1.NotReady
	}

	return policyv1.Ready
}

func isDaemonsetSpecChanged(current *appsv1.DaemonSet, new *appsv1.DaemonSet) bool {
	if current == nil && new != nil {
		return true
	}
	if current.Annotations == nil || new.Annotations == nil {
		panic("appsv1.DaemonSet.Annotations must be allocated prior to calling isDaemonsetSpecChanged()")
	}

	hashStr := getDaemonsetHash(new)
	foundHashAnnotation := false

	for annotation, value := range current.Annotations {
		if annotation == XilinxAnnotationHashKey {
			if value != hashStr {
				// update annotation to be added to Daemonset as per new spec and indicate spec update is required
				new.Annotations[XilinxAnnotationHashKey] = hashStr
				return true
			}
			foundHashAnnotation = true
			break
		}
	}

	if !foundHashAnnotation {
		// update annotation to be added to Daemonset as per new spec and indicate spec update is required
		new.Annotations[XilinxAnnotationHashKey] = hashStr
		return true
	}
	return false
}

// setDaemonSetSelctor add nodeSelector for daemonset
func setDaemonSetSelector(obj *appsv1.DaemonSet, labels map[string]string) {
	if obj.Spec.Template.Spec.NodeSelector == nil {
		obj.Spec.Template.Spec.NodeSelector = labels
	} else {
		// append labels to exsiting selector
		for k, v := range labels {
			obj.Spec.Template.Spec.NodeSelector[k] = v
		}
	}
}

// setContainerEnv add environment variables for container
func setContainerEnv(c *corev1.Container, key, value string) {
	for i, val := range c.Env {
		if val.Name != key {
			continue
		}

		c.Env[i].Value = value
		return
	}
	c.Env = append(c.Env, corev1.EnvVar{Name: key, Value: value})
}

// getRuntimeConfig returns Docker/containerd config filepath
func getRuntimeConfig(runtime policyv1.Runtime) string {
	switch runtime {
	case policyv1.Docker:
		return DefaultDockerConfig
	case policyv1.Containerd:
		return DefaultContainerdConfig
	default:
		return ""
	}
}

// getRuntimeSocket returns Docker/containerd socket filepath
func getRuntimeSocket(runtime policyv1.Runtime) string {
	switch runtime {
	case policyv1.Docker:
		return DefaultDockerSocket
	case policyv1.Containerd:
		return DefaultContainerdSocket
	default:
		return ""
	}
}

// preProcessDaemonset update the daemonset object base on the state name
func preProcessDaemonSet(obj *appsv1.DaemonSet, ctrl ClusterPolicyController) error {
	logger := ctrl.rec.Log
	transformations := map[string]func(*appsv1.DaemonSet, *policyv1.ClusterPolicySpec, ClusterPolicyController) error{
		"xilinx-container-runtime-daemonset": TransformContainerRuntime,
		"device-plugin-daemonset":            TransformDevicePlugin,
		"host-setup-ubuntu18-daemonset":      TransformHostSetup,
		"host-setup-ubuntu20-daemonset":      TransformHostSetup,
		"host-setup-ubuntu22-daemonset":      TransformHostSetup,
		"host-setup-al2-daemonset":           TransformHostSetup,
		"host-setup-centos7-daemonset":       TransformHostSetup,
		"host-setup-centos8-daemonset":       TransformHostSetup,
	}

	t, ok := transformations[obj.Name]
	if !ok {
		logger.Info(fmt.Sprintf("No transformation for Daemonset '%s'", obj.Name))
		return nil
	}

	err := t(obj, &ctrl.singleton.Spec, ctrl)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to apply transformation '%s' with error: '%v'", obj.Name, err))
		return err
	}
	return nil
}

// DaemonSet creates DaemonSet resource
func DaemonSet(ctrl ClusterPolicyController) (policyv1.State, error) {
	idx := ctrl.idx
	result := policyv1.Ready

	ctrl.rec.Log.Info(fmt.Sprintf("There is %d DaemonSets to be created",
		len(ctrl.resources[idx].Daemonsets)), "State", ctrl.stateNames[idx])

	for _, daemonSet := range ctrl.resources[idx].Daemonsets {
		obj := daemonSet.DeepCopy()
		// obj := ctrl.resources[idx].DaemonSet.DeepCopy()
		obj.Namespace = ctrl.operatorNamespace

		logger := ctrl.rec.Log.WithValues("DaemonSet", obj.Name, "Namespace", obj.Namespace)

		// Check if state is disabled and cleanup resource if exists
		if !ctrl.isStateEnabled(ctrl.stateNames[ctrl.idx]) {
			err := ctrl.rec.Client.Delete(context.TODO(), obj)
			if err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Couldn't delete")
				result = policyv1.NotReady
				continue
			}
			result = policyv1.Disabled
			continue
		}

		// pre-process DaemonSet object, edit specs for different state
		err := preProcessDaemonSet(obj, ctrl)
		if err != nil {
			if _, ok := err.(*NoSpecError); ok {
				// no spec found for this daemonset
				logger.Info(err.Error())
				// delete the daemonset if it was deployed before
				e := ctrl.rec.Client.Delete(context.TODO(), obj)
				if e != nil && !errors.IsNotFound(e) {
					logger.Error(err, "Couldn't delete")
					result = policyv1.NotReady
				}
				if result == policyv1.Ready {
					result = policyv1.Ignored
				}
			} else {
				logger.Error(err, "Could not pre-process")
				result = policyv1.NotReady
			}
			continue
		}

		if err := controllerutil.SetControllerReference(ctrl.singleton, obj, ctrl.rec.Scheme); err != nil {
			logger.Info("SetControllerReference failed", "Error", err)
			result = policyv1.NotReady
			continue
		}

		// Daemonsets will always have at least one annotation applied, so allocate if necessary
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}

		// check from the existed DaemonSet
		found := &appsv1.DaemonSet{}
		err = ctrl.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("DaemonSet not found, creating")
			// generate hash for the spec to create
			hashStr := getDaemonsetHash(obj)
			// add annotation to the Daemonset with hash value during creation
			obj.Annotations[XilinxAnnotationHashKey] = hashStr
			err = ctrl.rec.Client.Create(context.TODO(), obj)
			if err != nil {
				logger.Error(err, "Couldn't create DaemonSet")
				result = policyv1.NotReady
			}
			status := isDaemonSetReady(obj.Namespace, obj.Name, *ctrl.rec, logger)
			if status == policyv1.NotReady {
				result = policyv1.NotReady
			}
			continue
		} else if err != nil {
			logger.Error(err, "Failed to get DaemonSet from client")
			result = policyv1.NotReady
			continue
		}

		// update the DaemonSet if it is existed already and needing to be updated
		changed := isDaemonsetSpecChanged(found, obj)
		if changed {
			logger.Info("DaemonSet is different, updating")
			err = ctrl.rec.Client.Update(context.TODO(), obj)
			if err != nil {
				logger.Error(err, "Failed to update DaemonSet")
				result = policyv1.NotReady
				continue
			}
		} else {
			logger.Info("DaemonSet identical, skipping update")
		}

		if isDaemonSetReady(obj.Namespace, obj.Name, *ctrl.rec, logger) == policyv1.NotReady {
			result = policyv1.NotReady
		}
	}
	return result, nil
}

// TransformContainerRuntime transforms Xilinx container runtime daemonset with required config as per ClusterPolicy
func TransformContainerRuntime(obj *appsv1.DaemonSet, config *policyv1.ClusterPolicySpec, ctrl ClusterPolicyController) error {

	// udpate image and pull policy
	image := policyv1.ImagePath(config.ContainerRuntime.Repository,
		config.ContainerRuntime.Image, config.ContainerRuntime.Tag)
	obj.Spec.Template.Spec.Containers[0].Image = image
	obj.Spec.Template.Spec.Containers[0].ImagePullPolicy = policyv1.ImagePullPolicy(config.ContainerRuntime.ImagePullPolicy)

	// set image pull secrets
	if len(config.ContainerRuntime.ImagePullSecrets) > 0 {
		for _, secret := range config.ContainerRuntime.ImagePullSecrets {
			obj.Spec.Template.Spec.ImagePullSecrets = append(
				obj.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
	}

	// set/append environment variables for runtime container
	if len(config.ContainerRuntime.Env) > 0 {
		for _, env := range config.ContainerRuntime.Env {
			setContainerEnv(&(obj.Spec.Template.Spec.Containers[0]), env.Name, env.Value)
		}
	}

	// add mounts required for xilinx-container-runtime
	// XCR install directory mount
	installDir := DefaultXCRInstallDir
	if config.ContainerRuntime.InstallDir != "" {
		installDir = config.ContainerRuntime.InstallDir
	}
	installDirVolName := "install-dir"
	installDirMountPath := "/host-usr/bin"
	obj.Spec.Template.Spec.Volumes = append(obj.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: installDirVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: installDir,
			},
		},
	})
	obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		obj.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      installDirVolName,
			MountPath: installDirMountPath,
		})

	// XCR config directory mount
	configDir := DefaultXCRConfigDir
	configDirVolName := "config-dir"
	configDirMountPath := "/host-etc/xilinx-container-runtime"
	obj.Spec.Template.Spec.Volumes = append(obj.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: configDirVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: configDir,
			},
		},
	})
	obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		obj.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      configDirVolName,
			MountPath: configDirMountPath,
		})

	// runtime config mount
	runtimeConfig := getRuntimeConfig(ctrl.runtime)
	runtimeConfigVolName := "runtime-config"
	runtimeConfigMountPath := "/runtime/config/" + path.Base(runtimeConfig)
	obj.Spec.Template.Spec.Volumes = append(obj.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: runtimeConfigVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: runtimeConfig,
			},
		},
	})
	obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		obj.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      runtimeConfigVolName,
			MountPath: runtimeConfigMountPath,
		})

	// runtime socket mount
	runtimeSocket := getRuntimeSocket(ctrl.runtime)
	runtimeSocketVolName := "runtime-socket"
	runtimeSocketMountPath := "/runtime/socket/" + path.Base(runtimeSocket)
	obj.Spec.Template.Spec.Volumes = append(obj.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: runtimeSocketVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: runtimeSocket,
			},
		},
	})
	obj.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		obj.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      runtimeSocketVolName,
			MountPath: runtimeSocketMountPath,
		})

	// set args for xilinx-container-toolkit
	toolkitArgStrFmt := "xilinx-container-toolkit install --install-dir %s --config-dir %s; xilinx-container-toolkit %s setup -p %s -r %s -c %s -s %s; while true; do sleep 3600; done"
	if config.ContainerRuntime.SetAsDefault != nil && *config.ContainerRuntime.SetAsDefault {
		toolkitArgStrFmt = "xilinx-container-toolkit install --install-dir %s --config-dir %s; xilinx-container-toolkit %s setup -p %s -r %s -c %s -s %s --set-as-default; while true; do sleep 3600; done"
	}
	toolkitArg := fmt.Sprintf(toolkitArgStrFmt, installDirMountPath, configDirMountPath, ctrl.runtime.String(), installDir, getRuntimeClass(&ctrl.singleton.Spec), runtimeConfigMountPath, runtimeSocketMountPath)
	obj.Spec.Template.Spec.Containers[0].Command = []string{"/bin/bash"}
	obj.Spec.Template.Spec.Containers[0].Args = []string{"-c", toolkitArg}

	// set nodeSelector for daemonset
	setDaemonSetSelector(obj, fpgaNodeLabels)

	return nil
}

func TransformDevicePlugin(obj *appsv1.DaemonSet, config *policyv1.ClusterPolicySpec, ctrl ClusterPolicyController) error {
	// update image and pull policy
	obj.Spec.Template.Spec.Containers[0].Image = policyv1.ImagePath(
		config.DevicePlugin.Repository, config.DevicePlugin.Image, config.DevicePlugin.Tag)
	obj.Spec.Template.Spec.Containers[0].ImagePullPolicy = policyv1.ImagePullPolicy(
		config.DevicePlugin.ImagePullPolicy)

	// set image pull secrets
	if len(config.DevicePlugin.ImagePullSecrets) > 0 {
		for _, secret := range config.DevicePlugin.ImagePullSecrets {
			obj.Spec.Template.Spec.ImagePullSecrets = append(
				obj.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
	}

	// set/append environment virables
	if len(config.DevicePlugin.Env) > 0 {
		for _, env := range config.DevicePlugin.Env {
			setContainerEnv(&(obj.Spec.Template.Spec.Containers[0]), env.Name, env.Value)
		}
	}

	// set node selector
	setDaemonSetSelector(obj, fpgaNodeLabels)
	return nil
}

func TransformHostSetup(obj *appsv1.DaemonSet, config *policyv1.ClusterPolicySpec, ctrl ClusterPolicyController) error {
	// get node selector from daemonset template
	if obj.Spec.Template.Spec.NodeSelector == nil {
		return fmt.Errorf("node selector of OS ID and version is missing")
	}
	selector := obj.Spec.Template.Spec.NodeSelector

	ctrl.rec.Log.Info("Trying to update daemonset", "Name", obj.Name)

	for _, osDistSpec := range config.HostSetup.OsDists {
		// find the correct spec based on node selector
		if !strings.EqualFold(osDistSpec.OsId, selector[nfdLabelOSReleaseID]) ||
			!strings.EqualFold(osDistSpec.OsMajorVersion, selector[nfdLabelOsMajorVersion]) {
			continue
		}

		ctrl.rec.Log.Info("Found spec for daemonset", "Name", obj.Name)

		// update iamge and pull policy
		image := policyv1.ImagePath(osDistSpec.Repository, osDistSpec.Image, osDistSpec.Tag)
		imagePullPolicy := policyv1.ImagePullPolicy(osDistSpec.ImagePullPolicy)
		obj.Spec.Template.Spec.InitContainers[0].Image = image
		obj.Spec.Template.Spec.InitContainers[0].ImagePullPolicy = imagePullPolicy
		obj.Spec.Template.Spec.InitContainers[1].Image = image
		obj.Spec.Template.Spec.InitContainers[1].ImagePullPolicy = imagePullPolicy
		obj.Spec.Template.Spec.Containers[0].Image = image
		obj.Spec.Template.Spec.Containers[0].ImagePullPolicy = imagePullPolicy

		// set image pull secrets
		if len(osDistSpec.ImagePullSecrets) > 0 {
			for _, secret := range osDistSpec.ImagePullSecrets {
				obj.Spec.Template.Spec.ImagePullSecrets = append(
					obj.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
			}
		}

		// install xrt
		xrtInstallationStr := fmt.Sprintf("source ./host_setup.sh -y --skip-shell-flash -v %s", osDistSpec.Version)
		obj.Spec.Template.Spec.InitContainers[0].Command = []string{"/bin/bash"}
		obj.Spec.Template.Spec.InitContainers[0].Args = []string{"-c", xrtInstallationStr}

		// flash card
		// check if shell flash is enabled
		if osDistSpec.ShellFlashEnabled != nil && !*osDistSpec.ShellFlashEnabled {
			obj.Spec.Template.Spec.InitContainers[1].Command = []string{"/bin/bash"}
			obj.Spec.Template.Spec.InitContainers[1].Args = []string{"-c", "echo card flash is disabled"}
		} else {

			// basic flash card string
			cardFlashStr := fmt.Sprintf("source ./host_setup.sh -y --skip-xrt-install -v %s", osDistSpec.Version)

			// check if cards is specified
			if osDistSpec.Cards != nil && len(osDistSpec.Cards) > 0 {
				multiSetupArgStr := ""
				for _, card := range osDistSpec.Cards {
					line := cardFlashStr + " -p " + card + "; "
					multiSetupArgStr += line
				}
				// cardFlashStr = multiSetupArgStr[0 : len(multiSetupArgStr)-1]
			}

			// set command args
			obj.Spec.Template.Spec.InitContainers[1].Command = []string{"/bin/bash"}
			obj.Spec.Template.Spec.InitContainers[1].Args = []string{"-c", cardFlashStr}

		}
		// set command args
		obj.Spec.Template.Spec.Containers[0].Command = []string{"/bin/bash"}
		obj.Spec.Template.Spec.Containers[0].Args = []string{"-c", "echo host_setup complete, please refer to logs from init containers for further steps; while true; do sleep 3600; done"}
		return nil
	}

	// no spec for this os dist found
	ctrl.rec.Log.Info("No spec found for daemonset", "Name", obj.Name)
	return &NoSpecError{}
}
