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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// logf "sigs.k8s.io/controller-runtime/pkg/log"

	certsv1alpha1 "example.com/mini-cert-operator/api/v1alpha1"
	"example.com/mini-cert-operator/internal/certificates"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// CertificateReconciler reconciles a Certificate object
type CertificateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=certs.example.com,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certs.example.com,resources=certificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=certs.example.com,resources=certificates/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Certificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = logf.FromContext(ctx)

	var cert certsv1alpha1.Certificate
	if err := r.Get(ctx, req.NamespacedName, &cert); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !cert.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	duration, err := time.ParseDuration(cert.Spec.Duration)
	if err != nil || duration < time.Minute {
		return ctrl.Result{}, errors.New("spec.duration must be a valid duration of at least 1m")
	}
	renewBefore, err := time.ParseDuration(cert.Spec.RenewBefore)
	if err != nil || renewBefore <= 0 || renewBefore >= duration {
		return ctrl.Result{}, errors.New("spec.renewBefore must be positive and shorter than duration")
	}

	key := client.ObjectKey{
		Namespace: cert.Namespace,
		Name:      cert.Spec.SecretName,
	}
	var secret corev1.Secret
	err = r.Get(ctx, key, &secret)
	missing := apierrors.IsNotFound(err)

	var notAfter time.Time
	switch {
	case missing:
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
		}
		if err := ctrl.SetControllerReference(&cert, &secret, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
	case err != nil:
		return ctrl.Result{}, err
	default:
		if !metav1.IsControlledBy(&secret, &cert) {
			return ctrl.Result{}, fmt.Errorf(
				"Secret %s/%s already exists and is owned by another resource or has on owner",
				key.Namespace, key.Name,
			)
		}
		notAfter = existingExpiry(&secret, cert.Spec.DNSNames)
	}

	if notAfter.IsZero() || time.Until(notAfter.Add(-renewBefore)) <= 0 {
		material, err := certificates.Generate(cert.Spec.DNSNames, duration)
		if err != nil {
			return ctrl.Result{}, err
		}
		secret.Type = corev1.SecretTypeTLS
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[corev1.TLSCertKey] = material.CertificatePEM
		secret.Data[corev1.TLSPrivateKeyKey] = material.PrivateKeyPEM

		if missing {
			err = r.Create(ctx, &secret)
		} else {
			err = r.Update(ctx, &secret)
		}
		if err != nil {
			return ctrl.Result{}, err
		}
		notAfter = material.NotAfter
	}
	renewalTime := notAfter.Add(-renewBefore)
	if err := r.recordReady(ctx, &cert, notAfter, renewalTime); err != nil {
		return ctrl.Result{}, err
	}
	delay := max(time.Second, time.Until(renewalTime))
	return ctrl.Result{RequeueAfter: delay}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certsv1alpha1.Certificate{}).
		Owns(&corev1.Secret{}).
		Named("certificate").
		Complete(r)
}

func (r *CertificateReconciler) recordReady(
	ctx context.Context,
	cert *certsv1alpha1.Certificate,
	notAfter, renewalTime time.Time,
) error {
	before := cert.DeepCopy()
	expiry := metav1.NewTime(notAfter)
	renewal := metav1.NewTime(renewalTime)

	cert.Status.ObservedGeneration = cert.Generation
	cert.Status.NotAfter = &expiry
	cert.Status.RenewalTime = &renewal
	meta.SetStatusCondition(&cert.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: cert.Generation,
		Reason:             "SecretReady",
		Message:            "TLS Secret is ready",
	})

	if apiequality.Semantic.DeepEqual(before.Status, cert.Status) {
		return nil
	}
	return r.Status().Update(ctx, cert)
}

func existingExpiry(secret *corev1.Secret, dnsNames []string) time.Time {
	if secret.Type != corev1.SecretTypeTLS {
		return time.Time{}
	}
	pair, err := tls.X509KeyPair(
		secret.Data[corev1.TLSCertKey],
		secret.Data[corev1.TLSPrivateKeyKey],
	)
	if err != nil {
		return time.Time{}
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil || !slices.Equal(leaf.DNSNames, dnsNames) {
		return time.Time{}
	}
	return leaf.NotAfter
}
