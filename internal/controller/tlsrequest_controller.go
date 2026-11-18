/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	// logf "sigs.k8s.io/controller-runtime/pkg/log"

	certsv1alpha1 "example.com/mini-cert-operator/api/v1alpha1"
)

// TLSRequestReconciler reconciles a TLSRequest object
type TLSRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=certs.example.com,resources=tlsrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certs.example.com,resources=tlsrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=certs.example.com,resources=tlsrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TLSRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *TLSRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var parent certsv1alpha1.TLSRequest
	if err := r.Get(ctx, req.NamespacedName, &parent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !parent.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	child := &certsv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parent.Name,
			Namespace: parent.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, child, func() error {
		if child.ResourceVersion != "" && !metav1.IsControlledBy(child, &parent) {
			return fmt.Errorf("Certificate %s already exists with a different owner", client.ObjectKeyFromObject(child))
		}
		if err := ctrl.SetControllerReference(&parent, child, r.Scheme); err != nil {
			return err
		}

		child.Spec.SecretName = parent.Spec.SecretName
		child.Spec.DNSNames = slices.Clone(parent.Spec.DNSNames)
		child.Spec.Duration = strings.Clone(parent.Spec.Duration)
		child.Spec.RenewBefore = strings.Clone(parent.Spec.RenewBefore)
		return nil
	})

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TLSRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certsv1alpha1.TLSRequest{}).
		Owns(&certsv1alpha1.Certificate{}).
		Named("tlsrequest").
		Complete(r)
}
