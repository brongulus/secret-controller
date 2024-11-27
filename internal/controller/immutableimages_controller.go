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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;get;

// Add the given secret to the immutableSecretsList
func (r *ImmutableImagesReconciler) addSecretToImageMap(ctx context.Context, images *batchv1.ImmutableImages, imageName, secretName string) error {
	// log := log.FromContext(ctx)
	if !slices.Contains(images.Spec.ImmutableSecrets, secretName) {
		images.Spec.ImmutableSecrets = append(images.Spec.ImmutableSecrets, secretName)
		fmt.Printf("Adding secret %s to status\n", secretName)
	}

	if _, found := images.Spec.ImageSecretsMap[imageName]; found {
		if !slices.Contains(images.Spec.ImageSecretsMap[imageName], secretName) {
			// fmt.Printf("Adding secret %s to status\n", secretName)
			images.Spec.ImageSecretsMap[imageName] = append(images.Spec.ImageSecretsMap[imageName], secretName)
		}
	}
	return nil
}

// Checks if there are secrets for the pod satisfying the
// immutableimage criteria, add to immutableSecretsList
func (r *ImmutableImagesReconciler) fetchPodSecrets(ctx context.Context, images *batchv1.ImmutableImages, pod *corev1.Pod) (sets.Set[string], error) {
	// log := log.FromContext(ctx)
	secretList := sets.New[string]()

	// pod.Volumes.Secret.SecretName
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			// FIXME Check if the particular volume has an associated immutable image
			imageName := ""
			hasImmutableImage := slices.ContainsFunc(pod.Spec.Containers, func(container corev1.Container) bool {
				for _, mount := range container.VolumeMounts {
					if mount.Name == volume.Name {
						// Check if image is part of immutable map
						if _, found := images.Spec.ImageSecretsMap[container.Image]; found {
							imageName = container.Image
							return true
						}
					}
				}
				return false
			})
			if hasImmutableImage {
				secretName := volume.Secret.SecretName
				secretList.Insert(secretName)
				if err := r.addSecretToImageMap(ctx, images, imageName, secretName); err != nil {
					return secretList, err
				}
			}
		}
	}

	// Ref: https://stackoverflow.com/questions/46406596/how-to-identify-unused-secrets-in-kubernetes
	// Ref: https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/
	for _, container := range pod.Spec.Containers {
		// Check if image is part of immutable map
		_, hasImmutableImage := images.Spec.ImageSecretsMap[container.Image]
		imageName := container.Image
		// pod.Containers.Env.ValueFrom.SecretKeyRef.Name
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && hasImmutableImage {
				secretName := env.ValueFrom.SecretKeyRef.Name
				secretList.Insert(secretName)
				if err := r.addSecretToImageMap(ctx, images, imageName, secretName); err != nil {
					return secretList, err
				}
			}
		}
		// pod.Containers.EnvFrom.SecretRef.Name
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && hasImmutableImage {
				secretName := envFrom.SecretRef.Name
				secretList.Insert(secretName)
				if err := r.addSecretToImageMap(ctx, images, imageName, secretName); err != nil {
					return secretList, err
				}
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

	log.V(1).Info("Inside reconcile <<<")
	images := &batchv1.ImmutableImages{}

	if err := r.Get(ctx, req.NamespacedName, images); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Could not fetch imagelist")
			return ctrl.Result{}, err
		}
		log.Info("Ignoring not found since imagelist is deleted or not created")
		return ctrl.Result{}, nil
	}
	// TODO: Updates to the CR
	// CR deletion
	// imageFinalizer := "batch.github.com/finalizer"
	// // Check if the object is being deleted
	// if images.GetDeletionTimestamp().IsZero() {
	// 	if !controllerutil.ContainsFinalizer(images, imageFinalizer) {
	// 		controllerutil.AddFinalizer(images, imageFinalizer)
	// 		if err := r.Update(ctx, images); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		// log.V(1).Info("finalizer added ===")
	// 	}
	// } else {
	// 	// Object being deleted
	// 	if controllerutil.ContainsFinalizer(images, imageFinalizer) {
	// 		// our finalizer is present, so lets handle any external dependency
	// 		fmt.Println("=== CR has finalizer")
	// 		// if err := r.removeImmutableRestartSecret(ctx, req, images); err != nil {
	// 		// 	return ctrl.Result{}, err
	// 		// }

	// 		// remove our finalizer from the list and update it.
	// 		controllerutil.RemoveFinalizer(images, imageFinalizer)
	// 		// FIXME
	// 		if err := r.Update(ctx, images); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// 	// Stop reconciliation as the item is being deleted
	// 	return ctrl.Result{}, nil
	// }

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(req.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range podList.Items {
		// Reconcile
		fmt.Printf("Pod is %s\n", pod.Name)

		// Get list of all the secrets attached to a pod
		secretList, err := r.fetchPodSecrets(ctx, images, &pod)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get pod secrets: %w", err)
		}
		if err := r.Update(ctx, images); err != nil { // FIXME
			log.Error(err, "Could not update immutable secret list")
			return ctrl.Result{}, err
		}

		// Tag each secret as immutable
		for secret := range secretList {
			fmt.Printf("Secret is %s\n", secret)
		}
	}
	log.V(1).Info(">>> Reconcile Over")
	fmt.Println("=======================================")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImmutableImagesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.ImmutableImages{}).
		Watches(
			&corev1.Pod{}, // Ref: https://squiggly.dev/2023/07/enqueue-your-father-was-a-mapfunc/
			handler.EnqueueRequestsFromMapFunc(
				func(ctx context.Context, obj client.Object) []reconcile.Request {
					pod := obj.(*corev1.Pod)
					// Get all images in pod namespace and create a request for them
					var immutableList batchv1.ImmutableImagesList
					if err := r.List(ctx, &immutableList, client.InNamespace(pod.Namespace)); err != nil {
						return nil
					}
					var requests []reconcile.Request
					for _, immutable := range immutableList.Items {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      immutable.Name,
								Namespace: pod.Namespace,
							},
						})
					}
					return requests
				},
			),
		).
		Named("immutableimages").
		Complete(r)
}

// func (r *ImmutableImagesReconciler) removeImmutableRestartSecret(ctx context.Context, req ctrl.Request, images *batchv1.ImmutableImages) error {
// 	log := log.FromContext(ctx)
// }
