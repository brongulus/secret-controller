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

package v1

import (
	"context"
	"fmt"
	"slices"

	batchv1 "github.com/brongulus/secret-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var secretlog = logf.Log.WithName("secret-resource")

// SetupSecretWebhookWithManager registers the webhook for Secret in the manager.
func SetupSecretWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Secret{}).
		WithValidator(&SecretCustomValidator{
			client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-secret,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=secrets,verbs=update,versions=v1,name=vsecret-v1.kb.io,admissionReviewVersions=v1

// SecretCustomValidator struct is responsible for validating the Secret resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SecretCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
	client client.Client
}

var _ webhook.CustomValidator = &SecretCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Secret.
func (v *SecretCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("expected a Secret object but got %T", obj)
	}
	secretlog.Info("Validation for Secret upon creation", "name", secret.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Secret.
func (v *SecretCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	secret, ok := newObj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("expected a Secret object for the newObj but got %T", newObj)
	}
	secretlog.Info("Validation for Secret upon update", "name", secret.GetName())

	// TODO(user): fill in your validation logic upon object update.
	// How to link secret obj with the imagelist CR map[image]sec, list of actively blacklisted secrets, this should only check the list
	// However reconcile looks at the map to update the blacklisted secret list on deletion/updation in CR

	// DONE: panic!! Get CR list, check if secret is contained in any of their status
	immutableImagesList := &batchv1.ImmutableImagesList{}

	if err := v.client.List(ctx, immutableImagesList); err != nil {
		// TODO: Do not error out if imagelist does not exist
		return nil, fmt.Errorf("failed to list immutableImages: %w", err)
	}

	for _, images := range immutableImagesList.Items {
		fmt.Printf("SecretList: %v, key: %v\n", images.Spec.ImmutableSecrets, secret.Name)
		if slices.Contains(images.Spec.ImmutableSecrets, secret.Name) {
			return nil, fmt.Errorf("attempting to update immutable secret %s",
				secret.Name)
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Secret.
func (v *SecretCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("expected a Secret object but got %T", obj)
	}
	secretlog.Info("Validation for Secret upon deletion", "name", secret.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
