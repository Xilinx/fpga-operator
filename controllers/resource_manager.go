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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type assetsFromFile []byte

type Resources struct {
	Daemonsets   []appsv1.DaemonSet
	RuntimeClass nodev1.RuntimeClass
}

func filePathWalkDir(ctrl *ClusterPolicyController, root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ctrl.rec.Log.Error(err, "error in filepath.Walk on %s", root)
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func getAssetsFrom(ctrl *ClusterPolicyController, path string) []assetsFromFile {
	manifests := []assetsFromFile{}
	files, err := filePathWalkDir(ctrl, path)
	if err != nil {
		panic(err)
	}
	sort.Strings(files)
	for _, file := range files {

		buffer, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		manifests = append(manifests, buffer)
	}
	return manifests
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func addResourceControls(ctrl *ClusterPolicyController, path string) (Resources, controlFuncs) {
	res := Resources{}
	ctrlFuncs := controlFuncs{}

	ctrl.rec.Log.Info("Getting assets from:", "path:", path)

	manifests := getAssetsFrom(ctrl, path)
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	// regexp to find the asset kind
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	for _, m := range manifests {
		// get the kind of this resource
		kind := reg.FindString(string(m))
		slice := strings.Split(kind, ":")
		kind = strings.TrimSpace(slice[1])
		ctrl.rec.Log.Info(fmt.Sprintf("Looking for %s in %s", kind, path))

		switch kind {
		// add control func for each kind of resource
		case "DaemonSet":
			// _, _, err := s.Decode(m, nil, &res.DaemonSet)
			ds := appsv1.DaemonSet{}
			_, _, err := s.Decode(m, nil, &ds)
			panicIfError(err)

			// found DaemonSet
			ctrl.rec.Log.Info("Found DaemonSet", "Name", ds.Name, "Path", path)
			if res.Daemonsets == nil {
				res.Daemonsets = []appsv1.DaemonSet{}
				ctrlFuncs = append(ctrlFuncs, DaemonSet)
			}
			res.Daemonsets = append(res.Daemonsets, ds)
		case "RuntimeClass":
			_, _, err := s.Decode(m, nil, &res.RuntimeClass)
			panicIfError(err)

			// found RuntimeClass
			ctrl.rec.Log.Info("Found RuntimeClass", "Name", &res.RuntimeClass.Name)
			ctrlFuncs = append(ctrlFuncs, RuntimeClass)
		}
	}
	return res, ctrlFuncs
}
