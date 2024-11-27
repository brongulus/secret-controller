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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "github.com/brongulus/secret-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Secret Webhook", func() {
	var (
		newObj    *corev1.Secret
		oldObj    *corev1.Secret
		imageList *batchv1.ImmutableImages
		validator SecretCustomValidator
		timeout   = time.Second * 10
		interval  = time.Millisecond * 250
	)

	BeforeEach(func() {
		newObj = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: "default",
			},
			StringData: map[string]string{
				"password.txt": "newpass",
			},
			Type: "Opaque",
		}
		oldObj = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: "default",
			},
			StringData: map[string]string{
				"password.txt": "oldpass",
			},
			Type: "Opaque",
		}
		validator = SecretCustomValidator{
			client: k8sClient,
		}
		imageList := &batchv1.ImmutableImages{}
		typeNamespacedName := types.NamespacedName{
			Name:      "imagelist",
			Namespace: "default",
		}
		ctx := context.Background()
		By("creating the custom resource for the Kind ImmutableImages")
		err := k8sClient.Get(ctx, typeNamespacedName, imageList)
		if err != nil && errors.IsNotFound(err) {
			imageList := &batchv1.ImmutableImages{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "imagelist",
					Namespace: "default",
				},
				Spec: batchv1.ImmutableImagesSpec{
					ImageSecretsMap: map[string][]string{
						"alpine:latest": {},
						"nginx:0.3":     {},
					},
					ImmutableSecrets: []string{
						"secret-2",
					},
				},
			}
			Expect(k8sClient.Create(ctx, imageList)).To(Succeed())
		}

		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(newObj).NotTo(BeNil(), "Expected newObj to be initialized")
		Expect(imageList).NotTo(BeNil(), "Expected CR to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When updating Secret under Validating Webhook", func() {
		It("Should update when secret is not in immutable list", func() {
			By("adding secret to list")
			imageLookupKey := types.NamespacedName{
				Name:      "imagelist",
				Namespace: "default",
			}
			createdImage := &batchv1.ImmutableImages{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, imageLookupKey, createdImage)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			fmt.Printf("ImmutableSecretlist is %v\n", createdImage.Spec.ImmutableSecrets)
			fmt.Printf("Imagelist is %v\n", imageList)
			By("simulating a valid update scenario")
			newObj.StringData["password.txt"] = "newpassword"
			Expect(validator.ValidateUpdate(ctx, oldObj, newObj)).To(BeNil(),
				"Expected validation to update the secret password.txt")
		})

		It("Should fail for update in immutable secret", func() {
			By("adding secret to list")
			imageLookupKey := types.NamespacedName{
				Name:      "imagelist",
				Namespace: "default",
			}
			createdImage := &batchv1.ImmutableImages{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, imageLookupKey, createdImage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			createdImage.Spec.ImmutableSecrets = append(createdImage.Spec.ImmutableSecrets, "secret-1")
			Expect(k8sClient.Update(ctx, createdImage)).To(Succeed())

			fmt.Printf("ImmutableSecretlist is %v\n", createdImage.Spec.ImmutableSecrets)
			By("simulating a invalid update scenario")
			newObj.StringData["password.txt"] = "passupdatefail"
			Expect(validator.ValidateUpdate(ctx, oldObj, newObj)).Error().To(HaveOccurred(),
				"Expected validation to fail for updating immutable secret")
		})
	})

})
