/*
Copyright 2023.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dummyv1alpha1 "github.com/alexxsilvers/k8s-dummy-controller/api/v1alpha1"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const dummyFinalizer = "dummy/finalizer"

//+kubebuilder:rbac:groups=dummy.alexxsilvers,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dummy.alexxsilvers,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dummy.alexxsilvers,resources=dummies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dummy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dummy := &dummyv1alpha1.Dummy{}
	err := r.Get(ctx, req.NamespacedName, dummy)
	if err != nil {
		if errors.IsNotFound(err) { // Instance not found
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("Processing Dummy '%s', namespace '%s', message '%s'",
		dummy.Name,
		dummy.Namespace,
		dummy.Spec.Message,
	))

	if dummy.Status.SpecEcho != dummy.Spec.Message {
		logger.Info("Write new message to the 'status.specEcho'")
		dummy.Status.SpecEcho = dummy.Spec.Message
		statusUpdateErr := r.Status().Update(ctx, dummy)
		if statusUpdateErr != nil {
			logger.Error(statusUpdateErr, "Failed to update Dummy 'status.specEcho'")
			return ctrl.Result{}, statusUpdateErr
		}

		// Re-fetch the dummy resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		err = r.Get(ctx, req.NamespacedName, dummy)
		if err != nil {
			if errors.IsNotFound(err) { // Instance not found
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}
	}

	// Check, that we added finalizer If not add new one
	if !controllerutil.ContainsFinalizer(dummy, dummyFinalizer) {
		logger.Info("Add finalizer for Dummy")
		if ok := controllerutil.AddFinalizer(dummy, dummyFinalizer); !ok {
			logger.Error(err, "Failed to add finalizer to Dummy")
			return ctrl.Result{Requeue: true}, nil
		}

		updateErr := r.Update(ctx, dummy)
		if updateErr != nil {
			logger.Error(updateErr, "Failed to update Dummy to add finalizer")
			return ctrl.Result{}, updateErr
		}
	}

	// Check if the Dummy instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set, because we add finalizer above
	toBeDeleted := dummy.GetDeletionTimestamp() != nil
	if toBeDeleted && controllerutil.ContainsFinalizer(dummy, dummyFinalizer) {
		logger.Info("Performing Operations for Dummy before delete.")

		logger.Info("Removing finalizer for Dummy after successfully perform the operations")
		if ok := controllerutil.RemoveFinalizer(dummy, dummyFinalizer); !ok {
			logger.Error(err, "Failed to remove finalizer for Dummy")
			return ctrl.Result{Requeue: true}, nil
		}

		if err := r.Update(ctx, dummy); err != nil {
			logger.Error(err, "Failed to remove finalizer for Dummy")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Check that associated pod is created, if not - create a new one
	foundPod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Namespace: dummy.Namespace, Name: dummy.Name}, foundPod)
	if err != nil {
		logger.Error(err, "Failed to get Dummy's pod")
		if errors.IsNotFound(err) { // create new one
			pod, podDefinitionErr := r.createPodDefinition(dummy)
			if podDefinitionErr != nil {
				logger.Error(podDefinitionErr, "Create pod definition failed")
				return ctrl.Result{}, podDefinitionErr
			}

			createPodErr := r.Create(ctx, pod)
			if createPodErr != nil {
				logger.Error(createPodErr, "Create pod failed")
				return ctrl.Result{}, createPodErr
			}

			return ctrl.Result{}, nil
		}
	} else { // Pod founded - need to track pod status
		if foundPod.Status.Phase != dummy.Status.PodStatus {
			logger.Info("Write new status to the 'status.podStatus'")
			dummy.Status.PodStatus = foundPod.Status.Phase
			statusUpdateErr := r.Status().Update(ctx, dummy)
			if statusUpdateErr != nil {
				logger.Error(statusUpdateErr, "Failed to update Dummy 'status.podStatus'")
				return ctrl.Result{}, statusUpdateErr
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dummyv1alpha1.Dummy{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
