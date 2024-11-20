/*
Copyright 2024.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Sets the immutable field for the given secret as true
func (r *PodReconciler) tagSecretAsImmutable(ctx context.Context, req ctrl.Request, secretName string) (string, error) {
	log := log.FromContext(ctx)
	log.V(1).Info(secretName)

	// Get Secret
	secret := &corev1.Secret{}
	secretNamespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, secretNamespacedName, secret); err != nil {
		log.Error(err, "Could not fetch secret")
		return secretName, client.IgnoreNotFound(err)
	}

	// Tag Immutable if not
	if secret.Immutable == nil || *secret.Immutable != true {
		shouldBeImmutable := true
		secret.Immutable = &shouldBeImmutable
		if err := r.Update(ctx, secret); err != nil {
			log.Error(err, "Could not update secret")
			return secretName, err
		}
		log.V(1).Info("Secret is made immutable")
	} else {
		log.V(1).Info("Secret is already set")
	}
	// Return
	return secretName, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		log.Error(err, "Could not fetch pod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: Check if namespace is not kube-system etc
	if req.Namespace == "kube-system" || req.Namespace == "local-path-storage" {
		return ctrl.Result{}, nil
	}

	log.Info(pod.Name)

	// Get list of all the secrets attached to a pod
	secretList := sets.New[string]()

	// pod.Volumes.Secret.SecretName
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			secretList.Insert(volume.Secret.SecretName)
		}
	}

	// Ref: https://stackoverflow.com/questions/46406596/how-to-identify-unused-secrets-in-kubernetes
	// Ref: https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/
	for _, container := range pod.Spec.Containers {
		// pod.Containers.Env.ValueFrom.SecretKeyRef.Name
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secretList.Insert(env.ValueFrom.SecretKeyRef.Name)
			}
		}
		// pod.Containers.EnvFrom.SecretRef.Name
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil {
				secretList.Insert(envFrom.SecretRef.Name)
			}
		}
	}
	// pod.ImagePullSecrets.Name
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		if imagePullSecret.Name != "" { // FIXME
			secretList.Insert(imagePullSecret.Name)
		}
	}

	// Tag each secret as immutable
	for secret := range secretList {
		// FIXME: Is it an issue if early return happens i.e.
		// it errors out while only tagging one secret and the
		// rest aren't updated? Maybe call reconcile again
		_, err := r.tagSecretAsImmutable(ctx, req, secret)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Named("pod").
		Complete(r)
}
