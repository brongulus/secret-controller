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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Secret Controller", func() {

	const (
		PodName        = "test-pod"
		TestNamespace  = "default"
		TestSecretName = "test-secret"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When adding a pod having a secret", func() {
		It("should count towards an element of podsWithSecretsList", func() {
			// DONE try changing the order of creation of secret and pod (doesnt matter)
			By("By creating a new Secret")
			ctx := context.Background()
			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      TestSecretName,
					Namespace: TestNamespace,
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())

			secretLookupKey := types.NamespacedName{Name: TestSecretName, Namespace: TestNamespace}
			createdSecret := &corev1.Secret{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, secretLookupKey, createdSecret)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("By creating a new Pod")
			testPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PodName,
					Namespace: TestNamespace,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "credentials",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: TestSecretName,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "secret-container",
							Image: "alpine:latest",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testPod)).To(Succeed())

			podLookupKey := types.NamespacedName{Name: PodName, Namespace: TestNamespace}
			createdPod := &corev1.Pod{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Attach secret to pod (// FIXME pod updates may not change fields other than...)
			// testPod.Spec.Volumes[0].Secret.SecretName = TestSecretName
			// Expect(k8sClient.Update(ctx, testPod)).To(Succeed())

			// TODO: add check for seeing if the controller adds the immutable field to the secret

			By("Checking that the secret is attached to the Pod")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod)).To(Succeed(), "should GET the Pod")
				g.Expect(createdPod.Spec.Volumes[0].Secret.SecretName).To(Equal(TestSecretName), "secret should be attached")
			}, timeout, interval).Should(Succeed(), "should attach our secret to the pod")
		})
	})
})
