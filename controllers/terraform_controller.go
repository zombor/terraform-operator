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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "github.com/zombor/terraform-operator/api/v1alpha1"
	"github.com/zombor/terraform-operator/pkg/os"
	"github.com/zombor/terraform-operator/pkg/terraform"
)

// TerraformReconciler reconciles a Terraform object
type TerraformReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const finalizer = "terraform.zombor.net/finalizer"

//+kubebuilder:rbac:groups=infra.terraform.zombor.net,resources=terraforms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.terraform.zombor.net,resources=terraforms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.terraform.zombor.net,resources=terraforms/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *TerraformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		status string
		err    error

		logger = log.FromContext(ctx)
	)

	logger.Info("running reconcile")

	tf := &infrav1alpha1.Terraform{}
	err = r.Get(ctx, req.NamespacedName, tf)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("object not found, must be deleted")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&tf.Status.Conditions, metav1.Condition{
		Type:    "Apply",
		Status:  metav1.ConditionTrue,
		Reason:  "Applying",
		Message: "",
	})

	err = r.Status().Update(ctx, tf)

	err = os.Write("terraform-operator", []string{tf.Spec.HCL})
	if err != nil {
		return ctrl.Result{}, err
	}

	defer os.Delete("terraform-operator")

	// Check if the instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if tf.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(tf, finalizer) {
		logger.Info("destroying")

		if status, err = terraform.Destroy("/tmp/terraform-operator"); err != nil {
			meta.SetStatusCondition(&tf.Status.Conditions, metav1.Condition{
				Type:   "Apply",
				Status: metav1.ConditionFalse,
				Reason: "Finished",
				Message: func() string {
					if err != nil {
						return err.Error()
					}

					return status
				}(),
			})

			if serr := r.Status().Update(ctx, tf); serr != nil {
				return ctrl.Result{}, serr
			}
			return ctrl.Result{}, err
		}

		// remove finalizer
		controllerutil.RemoveFinalizer(tf, finalizer)

		return ctrl.Result{}, r.Update(ctx, tf)
	}

	status, err = terraform.Apply("/tmp/terraform-operator")

	meta.SetStatusCondition(&tf.Status.Conditions, metav1.Condition{
		Type:   "Apply",
		Status: metav1.ConditionFalse,
		Reason: "Finished",
		Message: func() string {
			if err != nil {
				return err.Error()
			}

			return status
		}(),
	})

	if serr := r.Status().Update(ctx, tf); serr != nil {
		return ctrl.Result{}, serr
	}

	if !controllerutil.ContainsFinalizer(tf, finalizer) {
		controllerutil.AddFinalizer(tf, finalizer)
		err = r.Update(ctx, tf)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TerraformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.Terraform{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}). // only trigger when generation changes
		Complete(r)
}
