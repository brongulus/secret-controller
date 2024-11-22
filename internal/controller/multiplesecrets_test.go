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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "github.com/brongulus/secret-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ImmutableImages Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName   = "test-resource"
			testNamespace  = "default"
			testSecretName = "test-secret"
			numSecrets     = 4

			timeout  = time.Second * 10
			duration = time.Second * 10
			interval = time.Millisecond * 250
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: testNamespace,
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
			resource := &batchv1.ImmutableImages{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ImmutableImages")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should make all relevant secrets immutable", func() {
			By("By creating multiple secrets")
			var secretList, createdSecretList [numSecrets]corev1.Secret
			for i := range numSecrets {
				ctx := context.Background()
				secretList[i] = corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testSecretName + fmt.Sprint(i),
						Namespace: testNamespace,
					},
					Type: "Opaque",
				}
				Expect(k8sClient.Create(ctx, &secretList[i])).To(Succeed())

				secretLookupKey := types.NamespacedName{Name: testSecretName + fmt.Sprint(i), Namespace: testNamespace}

				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, secretLookupKey, &createdSecretList[i])).To(Succeed())
				}, timeout, interval).Should(Succeed())

			}

			By("By creating a new Pod utilising all those secrets in various ways")
			testPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multiple-secret-pod",
					Namespace: testNamespace,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: testSecretName + "0",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "secret-container",
							Image: "nginx:0.3",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "credentials",
									MountPath: "/etc/credentials",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:  "envvar-container",
							Image: "alpine:edge",
							Env: []corev1.EnvVar{
								{
									Name: "secret-username",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "secretKeyRef-key",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: testSecretName + "1",
											},
										},
									},
								},
							},
						},
						{
							Name:  "envfrom-container",
							Image: "alpine:latest",
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: testSecretName + "2",
										},
									},
								},
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: testSecretName + "3", // FIXME Get panics for non-existent secret, maybe re-call reconcile after a while
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testPod)).To(Succeed())

			podLookupKey := types.NamespacedName{Name: "multiple-secret-pod", Namespace: testNamespace}
			createdPod := &corev1.Pod{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("Reconciling the created resource")
			controllerReconciler := &ImmutableImagesReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the correct secrets are immutable")
			Eventually(func(g Gomega) {
				for i := range numSecrets {
					secretLookupKey := types.NamespacedName{Name: testSecretName + fmt.Sprint(i), Namespace: testNamespace}
					g.Expect(k8sClient.Get(ctx, secretLookupKey, &createdSecretList[i])).To(Succeed(), "should GET the Secret")
					if i == 1 { // alpine:edge
						g.Expect(createdSecretList[i].Immutable).To(BeNil(), "Immutable should not be set")
					} else {
						g.Expect(createdSecretList[i].Immutable).To(HaveValue(Equal(true)), "secret should be Immutable") // Equal does strict type check, hance HaveValue
					}
				}
			}, timeout, interval).Should(Succeed(), "should attach our secret to the pod")
		})
	})
})
