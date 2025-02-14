package controller

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GenericController corev1.Secret", Serial, Ordered, func() {
	var (
		ctx              = context.Background()
		targetNamespaces = []string{"gryffindor", "ravenclaw", "hufflepuff", "slytherin"}
		secret           = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "a-secret",
				Namespace:   "default",
				Annotations: map[string]string{ReplicatorAllowedAnnotation: "true"},
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
			},
		}
	)

	Context("When a new resource is created", func() {
		It("should successfully replicate the secret in all namespaces", func() {
			By("create namespaces to replicate")
			for _, ns := range targetNamespaces {
				err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
				Expect(client.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
			}

			By("creating a new secret")
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			for _, ns := range targetNamespaces {
				Eventually(func() error {
					var obj corev1.Secret
					if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns}, &obj); err != nil {
						return err
					}

					v, ok := obj.Annotations[ReplicatorSourceAnnotation]
					if !ok {
						return fmt.Errorf("missing annotation %s in object %s/%s", ReplicatorSourceAnnotation, obj.Namespace, obj.Name)
					}
					if v != fmt.Sprintf("%s/%s", secret.Namespace, secret.Name) {
						return fmt.Errorf("annotations %s of object %s/%s does not equals %s/%s", obj.Annotations, obj.Namespace, obj.Name, secret.Namespace, secret.Name)
					}

					return nil
				}).ShouldNot(HaveOccurred())
			}
		})
	})
	Context("When source resource is updated", func() {
		It("should successfully reconcile the change of all replicas", func() {
			var (
				// update values -> for comparison
				fooValue = []byte(fmt.Sprintf("%s another value", secret.Data["foo"]))
				bazValue = []byte("loremipsum")
			)

			By("fetching the latest version of the existing secret")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, secret)).To(Succeed())

			By("changing source secret")
			secret.Data["foo"] = fooValue
			secret.Data["baz"] = bazValue

			Expect(k8sClient.Update(ctx, secret)).To(Succeed())

			for _, ns := range targetNamespaces {
				Eventually(func() error {
					var obj corev1.Secret
					if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns}, &obj); err != nil {
						return err
					}

					for key, val := range map[string][]byte{"foo": fooValue, "baz": bazValue} {
						v, ok := obj.Data[key]
						if !ok {
							return fmt.Errorf("missing object key %s in object %s/%s", key, obj.Namespace, obj.Name)
						}

						if string(v) != string(val) {
							return fmt.Errorf("value for object key %s mismatch in object %s/%s", key, obj.Namespace, obj.Name)
						}
					}

					return nil
				}).ShouldNot(HaveOccurred())
			}
		})
	})
	Context("When a replica is updated", func() {
		It("should be overwritten with the data from the source object", func() {
			By("changing the data of a replica")
			var replica corev1.Secret
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: "ravenclaw"}, &replica)).To(Succeed())
			replica.Data["foo"] = []byte("i-was-updated")
			Expect(k8sClient.Update(ctx, &replica)).To(Succeed())

			Eventually(func() error {
				var source corev1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &source); err != nil {
					return fmt.Errorf("fetching source object: %w", err)
				}

				var replica corev1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: "ravenclaw"}, &replica); err != nil {
					return fmt.Errorf("fetching replica: %w", err)
				}

				if cmp.Equal(source.Data, replica.Data) {
					return fmt.Errorf("replica %s/%s was not overwritten with data of source object", "ravenclaw", replica.Name)
				}

				return nil
			}).ShouldNot(HaveOccurred())
		})
	})
	Context("When a replica is deleted", func() {
		It("should be re-created", func() {
			By("deleting the replica")
			var replica corev1.Secret
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: "ravenclaw"}, &replica)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &replica)).To(Succeed())

			Eventually(func() error {
				var replica corev1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: "ravenclaw"}, &replica); err != nil {
					return fmt.Errorf("fetching replica: %w", err)
				}

				// fetching source object for data comparsion
				var source corev1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &source); err != nil {
					return fmt.Errorf("fetching source object: %w", err)
				}

				if cmp.Equal(source.Data, replica.Data) {
					return fmt.Errorf("replica %s/%s was not overwritten with data of source object", "ravenclaw", replica.Name)
				}

				return nil
			}).ShouldNot(HaveOccurred())
		})
	})
	Context("When the source object has is deleted", func() {
		It("should delete all replicas", func() {
			By("deleting the source object")
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())

			for _, ns := range targetNamespaces {
				Eventually(func() error {
					var replica corev1.Secret
					err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns}, &replica)
					if err == nil || !apierrors.IsNotFound(err) {
						return fmt.Errorf("replica found or internal err: %w", err)
					}
					return nil
				}).ShouldNot(HaveOccurred())
			}
		})
	})
	Context("When a new resource with allowed-namespaces annotations is created", func() {
		It("should replicate only in the specified namespaces", func() {
			secretWithAnnotation := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a-secret-with-annotation",
					Namespace: "ravenclaw",
					Annotations: map[string]string{
						ReplicatorAllowedAnnotation:           "true",
						ReplicatorAllowedNamespacesAnnotation: "gryffindor,slytherin",
					},
				},
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
			}

			Expect(k8sClient.Create(ctx, secretWithAnnotation)).To(Succeed())

			for ns, shouldExist := range map[string]bool{"default": false, "hufflepuff": false, "gryffindor": true, "slytherin": true} {
				Eventually(func() error {
					var replica corev1.Secret
					err := k8sClient.Get(ctx, types.NamespacedName{Name: secretWithAnnotation.Name, Namespace: ns}, &replica)
					if shouldExist && err != nil {
						return fmt.Errorf("replica of %s/%s does not exist (but should) in namespace %s: %w",
							secretWithAnnotation.Namespace, secretWithAnnotation.Name, ns, err)
					} else if !shouldExist && !apierrors.IsNotFound(err) {
						return fmt.Errorf("replica of %s/%s exist (but should not) in namespace %s: %w",
							secretWithAnnotation.Namespace, secretWithAnnotation.Name, ns, err)
					}

					return nil
				}).ShouldNot(HaveOccurred())
			}
		})
	})
	Context("When a resource is created without any annotation", func() {
		It("should not be reconciled in any namespace", func() {
			secretWithoutAnnotation := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "just-a-secret",
					Namespace: "ravenclaw",
				},
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
			}

			Expect(k8sClient.Create(ctx, secretWithoutAnnotation)).To(Succeed())

			var namespaces corev1.NamespaceList
			Expect(k8sClient.List(ctx, &namespaces)).To(Succeed())

			for _, ns := range namespaces.Items {
				if ns.Name == secretWithoutAnnotation.Namespace {
					continue
				}

				Eventually(func() error {
					var replica corev1.Secret
					err := k8sClient.Get(ctx, types.NamespacedName{Name: secretWithoutAnnotation.Name, Namespace: ns.Name}, &replica)
					if apierrors.IsNotFound(err) {
						return nil
					}

					return fmt.Errorf("a replica of %s/%s exist but shouldn't in namespace %s: %w",
						secretWithoutAnnotation.Namespace, secretWithoutAnnotation.Name, ns.Name, err)
				}).ShouldNot(HaveOccurred())
			}
		})
	})
})
