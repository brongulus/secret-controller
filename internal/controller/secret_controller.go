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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here (check for existence of multiple secrets i.e. parse a list)
	secret := &corev1.Secret{}

	// if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
	// 	log.Error(err, "Could not fetch secret")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	// TODO: handle ac for adding the label
	// TODO: index of used secrets only
	// TODO: create a pod controller that adds readonly to secret volumes?

	log.Info(secret.Name)

	if secret.Immutable != nil {
		fmt.Println("Immutable is set to", *secret.Immutable)
		return ctrl.Result{}, nil
	}

	fmt.Println("Immutable is nil")
	if secret.Labels != nil && secret.Labels["check"] == "this" { // FIXME: flow for check != this
		fmt.Println("Label is present")
		shouldBeImmutable := true
		secret.Immutable = &shouldBeImmutable
		fmt.Println("Updating Secret")
		if err := r.Update(ctx, secret); err != nil {
			log.Error(err, "Could not update secret")
			return ctrl.Result{}, err
		}
		fmt.Println("Immutable tagged as true")
	} else {
		fmt.Println("Label is not present")
	}

	// WIP: Get list of unused secrets
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list pods: %w", err)
	}

	secretsOnPods := sets.New[string]() // FIXME

	// Process each pod (Maybe cache already checked pods for perf?)
	// TODO: Test for pod without secret as well, and think about creating a
	// CRO for podsWithSecretsList or something and update that rather than fetching
	// podList and doing all these secret checks every time reconcile is called
	for _, pod := range podList.Items {
		// pod.Volumes.Secret.SecretName
		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil {
				log.Info(pod.Name)
				secretsOnPods.Insert(volume.Secret.SecretName)
			}
		}

		// Ref: https://stackoverflow.com/questions/46406596/how-to-identify-unused-secrets-in-kubernetes
		for _, container := range pod.Spec.Containers {
			// pod.Containers.Env.ValueFrom.SecretKeyRef.Name
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					log.Info(pod.Name)
					secretsOnPods.Insert(env.ValueFrom.SecretKeyRef.Name)
				}
			}
			// pod.Containers.EnvFrom.SecretRef.Name
			for _, envFrom := range container.EnvFrom {
				if envFrom.SecretRef != nil {
					log.Info(pod.Name)
					secretsOnPods.Insert(envFrom.SecretRef.Name)
				}
			}
		}
		// pod.ImagePullSecrets.Name
		for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
			if imagePullSecret.Name != "" { // FIXME
				log.Info(pod.Name)
				secretsOnPods.Insert(imagePullSecret.Name)
			}
		}
	}

	// TODO: Remove secList and secretSet later, not needed with secretsOnPodsSet
	secList := &corev1.SecretList{}
	if err := r.List(ctx, secList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list secrets: %w", err)
	}

	secretSet := sets.New[string]() // Get all secrets
	for _, sec := range secList.Items {
		if sec.Name != "" { // FIXME
			secretSet.Insert(sec.Name)
		}
	}

	log.V(1).Info("processed pods for secret references",
		"uniqueSecrets", secretsOnPods.Len(), "totalSecrets", secretSet.Len())

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}). // FIXME
		Watches(               // Needed to check which secrets are being used by pods
			&corev1.Pod{},
			&handler.EnqueueRequestForObject{},
		).
		Named("secret").
		Complete(r)
}
