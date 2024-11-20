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
	"slices"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "github.com/brongulus/secret-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// ImmutableImagesReconciler reconciles a ImmutableImages object
type ImmutableImagesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.github.com,resources=immutableimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.github.com,resources=immutableimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.github.com,resources=immutableimages/finalizers,verbs=update

// Sets the immutable field for the given secret as true
func (r *ImmutableImagesReconciler) tagSecretAsImmutable(ctx context.Context, req ctrl.Request, secretName string) (string, error) {
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

// Checks if there are secrets for the pod satisfying the
// immutableimage criteria, returns their set
func (r *ImmutableImagesReconciler) fetchPodSecrets(imageList batchv1.ImmutableImages, pod corev1.Pod) (sets.Set[string], error) {
	secretList := sets.New[string]()

	// pod.Volumes.Secret.SecretName
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			// Check if the particular volume has an associated immutable image
			hasImmutableImage := slices.ContainsFunc(pod.Spec.Containers, func(container corev1.Container) bool {
				for _, mount := range container.VolumeMounts {
					if mount.Name == volume.Name {
						return slices.Contains(imageList.Spec.Images, container.Image)
					}
				}
				return false
			})
			// log.V(1).Info(fmt.Sprint(hasImmutableImage))
			if hasImmutableImage {
				secretList.Insert(volume.Secret.SecretName)
			}
		}
	}

	// Ref: https://stackoverflow.com/questions/46406596/how-to-identify-unused-secrets-in-kubernetes
	// Ref: https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/
	for _, container := range pod.Spec.Containers {
		hasImmutableImage := slices.Contains(imageList.Spec.Images, container.Image)
		// pod.Containers.Env.ValueFrom.SecretKeyRef.Name
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && hasImmutableImage {
				secretList.Insert(env.ValueFrom.SecretKeyRef.Name)
			}
		}
		// pod.Containers.EnvFrom.SecretRef.Name
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && hasImmutableImage {
				secretList.Insert(envFrom.SecretRef.Name)
			}
		}
	}

	return secretList, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImmutableImages object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ImmutableImagesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var imageList batchv1.ImmutableImages
	if err := r.Get(ctx, req.NamespacedName, &imageList); err != nil {
		log.Error(err, "Could not fetch imagelist")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// FIXME: watch pod? fix GET call
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range podList.Items {
		// TODO: Check if namespace is not kube-system etc
		if req.Namespace == "kube-system" || req.Namespace == "local-path-storage" {
			return ctrl.Result{}, nil
		}

		log.Info(pod.Name)

		// Get list of all the secrets attached to a pod
		secretList, err := r.fetchPodSecrets(imageList, pod)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get secrets: %w", err)
		}

		// Tag each secret as immutable
		for secret := range secretList {
			// log.V(1).Info(fmt.Sprint(secret))

			// FIXME: Is it an issue if early return happens i.e.
			// it errors out while only tagging one secret and the
			// rest aren't updated? Maybe call reconcile again
			_, err := r.tagSecretAsImmutable(ctx, req, secret)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	// log.V(1).Info("Reconcile Over")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImmutableImagesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.ImmutableImages{}). // TODO: add pod watcher

		Named("immutableimages").
		Complete(r)
}
