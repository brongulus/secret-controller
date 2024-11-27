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
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	batchv1 "github.com/brongulus/secret-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ImmutableImages Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName   = "test-resource-podwatch"
			testNamespace  = "default"
			testSecretName = "test-secret"

			timeout  = time.Second * 10
			duration = time.Second * 10
			interval = time.Millisecond * 250
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: testNamespace, // TODO(user):Modify as needed
		}
		immutableimages := &batchv1.ImmutableImages{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ImmutableImages")
			err := k8sClient.Get(ctx, typeNamespacedName, immutableimages)
			if err != nil && errors.IsNotFound(err) {
				resource := &batchv1.ImmutableImages{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: testNamespace,
					},
					// TODO(user): Specify other spec details if needed.
					Spec: batchv1.ImmutableImagesSpec{
						ImageSecretsMap: map[string][]string{
							"alpine:latest": {},
							"nginx:0.3":     {},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &batchv1.ImmutableImages{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ImmutableImages")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {

			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.

			By("By creating a new Secret")
			ctx := context.Background()
			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())

			secretLookupKey := types.NamespacedName{Name: testSecretName, Namespace: testNamespace}
			createdSecret := &corev1.Secret{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, secretLookupKey, createdSecret)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By creating another Secret")
			testSecret2 := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName + "22",
					Namespace: testNamespace,
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, testSecret2)).To(Succeed())

			secretLookupKey2 := types.NamespacedName{Name: testSecretName + "22", Namespace: testNamespace}
			createdSecret2 := &corev1.Secret{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, secretLookupKey2, createdSecret2)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By creating a new Pod")
			testPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: testSecretName,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "secret-container",
							Image: "alpine:latest",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "credentials",
									MountPath: "/etc/credentials",
									ReadOnly:  true,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testPod)).To(Succeed())

			podLookupKey := types.NamespacedName{Name: "test-pod", Namespace: testNamespace}
			createdPod := &corev1.Pod{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By creating a second Pod")
			testPod2 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-2",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: testSecretName + "22",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "secret-container",
							Image: "alpine:latest",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "credentials",
									MountPath: "/etc/credentials",
									ReadOnly:  true,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testPod2)).To(Succeed())

			podLookupKey = types.NamespacedName{Name: "test-pod-2", Namespace: testNamespace}
			createdPod2 := &corev1.Pod{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod2)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			resource := &batchv1.ImmutableImages{}

			By("Checking that the attached secret is immutable")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed(), "should GET the CR")
				g.Expect(k8sClient.Get(ctx, secretLookupKey, createdSecret)).To(Succeed(), "should GET the Secret")
				g.Expect(slices.Contains(resource.Spec.ImmutableSecrets, createdSecret.Name)).To(Equal(true), "secret should be in Immutable list")
				g.Expect(k8sClient.Get(ctx, secretLookupKey2, createdSecret2)).To(Succeed(), "should GET the Secret")
				g.Expect(slices.Contains(resource.Spec.ImmutableSecrets, createdSecret2.Name)).To(Equal(true), "secret should be in Immutable list")
			}, timeout, interval).Should(Succeed(), "should attach our secret to the pod")

		})
	})
})
