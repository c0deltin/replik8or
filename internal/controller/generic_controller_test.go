package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GenericController corev1.Secret", Serial, Ordered, func() {
	Context("When reconciling a resource", func() {
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

		BeforeAll(func() {
			By("create namespaces to replicate")
			for _, ns := range targetNamespaces {
				err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
				Expect(client.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())
			}
		})

		It("should successfully replicate the secret in all namespaces", func() {
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
})
