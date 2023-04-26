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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	policyv1 "github.com/xilinx/fpga-operator/api/v1"
)

const (
	minDelayCR   = 100 * time.Millisecond
	maxDelayCR   = 30 * time.Second
	requeueDealy = 10 * time.Second
)

// blank assignment to verify that ReconcileClusterPolicy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ClusterPolicyReconciler{}
var clusterPolicyCtrl ClusterPolicyController

// ClusterPolicyReconciler reconciles a ClusterPolicy object
type ClusterPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=policy.xilinx.com,resources=clusterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy.xilinx.com,resources=clusterpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy.xilinx.com,resources=clusterpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces;serviceaccounts;pods;services;services/finalizers;endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims;events;configmaps;secrets;nodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.k8s.io,resources=runtimeclasses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ClusterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	_ = context.Background()
	_ = r.Log.WithValues("Reconciling ClusterPolicy", req.NamespacedName)

	// fetch the ClusterPolicy instance
	instance := &policyv1.ClusterPolicy{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		// clusterPolicyCtrl.operatorMetrics.reconciliationStatus.Set(reconciliationStatusClusterPolicyUnavailable)
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// TODO: Handle deletion of the main ClusterPolicy and cycle to the next one.
	// We already have a main Clusterpolicy
	if clusterPolicyCtrl.singleton != nil && clusterPolicyCtrl.singleton.ObjectMeta.Name != instance.ObjectMeta.Name {
		instance.SetStatus(policyv1.Ignored, clusterPolicyCtrl.operatorNamespace)
		// spurious reconciliation
		return ctrl.Result{}, err
	}

	err = clusterPolicyCtrl.init(r, instance)
	if err != nil {
		r.Log.Error(err, "Failed to initialize ClusterPolicy controller")

		// if clusterPolicyCtrl.operatorMetrics != nil {
		// 	clusterPolicyCtrl.operatorMetrics.reconciliationStatus.Set(reconciliationStatusClusterPolicyUnavailable)
		// }
		return ctrl.Result{}, err
	}

	// perform the oprator steps
	overallStatus := policyv1.Ready
	statesNotReady := []string{}
	for {
		status, statusError := clusterPolicyCtrl.step()
		if statusError != nil {
			updateCRState(r, req.NamespacedName, policyv1.NotReady)
			return ctrl.Result{RequeueAfter: requeueDealy}, statusError
		}
		if status == policyv1.NotReady {
			// if CR was previously set to ready(prior reboot etc), reset it to current state
			if instance.Status.State == policyv1.Ready {
				updateCRState(r, req.NamespacedName, policyv1.NotReady)
			}
			overallStatus = policyv1.NotReady
			statesNotReady = append(statesNotReady, clusterPolicyCtrl.stateNames[clusterPolicyCtrl.idx-1])
		}
		r.Log.Info("ClusterPolicy step completed",
			"state", clusterPolicyCtrl.stateNames[clusterPolicyCtrl.idx-1],
			"status", status)

		if clusterPolicyCtrl.last() {
			break
		}
	}

	// if any state is not ready, requeue for reconfile after 5 seconds
	if overallStatus != policyv1.Ready {

		r.Log.Info("ClusterPolicy isn't ready", "states not ready", statesNotReady)

		return ctrl.Result{RequeueAfter: requeueDealy}, nil
	}

	// Update CR state as ready as all states are complete
	updateCRState(r, req.NamespacedName, policyv1.Ready)
	return ctrl.Result{}, nil
}

func updateCRState(r *ClusterPolicyReconciler, namespacedName types.NamespacedName, state policyv1.State) error {
	// Fetch latest instance and update state to avoid version mismatch
	instance := &policyv1.ClusterPolicy{}
	err := r.Client.Get(context.TODO(), namespacedName, instance)
	if err != nil {
		r.Log.Error(err, "Failed to get ClusterPolicy instance for status update")
		return err
	}
	if instance.Status.State == state {
		// state is unchanged
		return nil
	}
	// Update the CR state
	instance.SetStatus(state, clusterPolicyCtrl.operatorNamespace)
	err = r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		r.Log.Error(err, "Failed to update ClusterPolicy status")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create a new controller
	c, err := controller.New("clusterpolicy-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 1,
			RateLimiter:             workqueue.NewItemExponentialFailureRateLimiter(minDelayCR, maxDelayCR),
		},
	)
	if err != nil {
		return err
	}

	// watch for changes to primary resource ClusterPolicy
	err = c.Watch(&source.Kind{Type: &policyv1.ClusterPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Daemonsets and requeue the owner ClusterPolicy
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &policyv1.ClusterPolicy{},
	})
	if err != nil {
		return err
	}

	return nil
}
