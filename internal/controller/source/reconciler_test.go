package source

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/c0deltin/replik8or/internal/replicator"
)

var _ = Describe("SourceReconciler reconciles corev1.ConfigMap", Ordered, func() {
	Context("when a new ConfigMap is created", func() {
		By("creating replica namespaces and source object")
		BeforeAll(func() {
			By("creating namespaces")
			for _, ns := range replicaNamespaces {
				namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
				if err := k8sClient.Create(ctx, &namespace); err != nil {
					Expect(ctrlclient.IgnoreAlreadyExists(err)).To(Succeed())
				}
			}

			By("creating source ConfigMap")
			source := sourceConfigMap.DeepCopy()
			Expect(ctrlclient.IgnoreAlreadyExists(k8sClient.Create(ctx, source))).To(Succeed())
		})

		It("should be replicated in all namespaces", func() {
			By("checking replicas")
			Eventually(func(g Gomega) {
				var replicaList corev1.ConfigMapList
				err := k8sClient.List(ctx, &replicaList, ctrlclient.MatchingLabels{
					replicator.SourceNameLabel:      sourceConfigMap.GetName(),
					replicator.SourceNamespaceLabel: sourceConfigMap.GetNamespace(),
				})
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(replicaList.Items).To(HaveLen(len(replicaNamespaces)))
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())

			By("ensuring no replica was created in disallowed namespaces")
			Eventually(func(g Gomega) {
				for _, ns := range systemNamespaces {
					var configMap corev1.ConfigMap
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{Name: sourceConfigMap.Name, Namespace: ns}, &configMap)
					g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				}
			}).Should(Succeed())
		})

		It("should recreate a replica when it was deleted", func() {
			var replica corev1.ConfigMap

			By("checking that replica exists")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ctrlclient.ObjectKey{Namespace: "testing", Name: sourceConfigMap.Name}, &replica)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(5 * time.Second).Should(Succeed())

			By("deleting replica")
			err := k8sClient.Delete(ctx, &replica)
			Expect(err).NotTo(HaveOccurred())

			By("checking that replica was deleted")
			Eventually(func(g Gomega) bool {
				err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(&replica), &replica)
				return apierrors.IsNotFound(err)
			}).WithTimeout(5 * time.Second).Should(BeTrue())

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(&replica), &replica)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(5 * time.Second).Should(Succeed())
		})

		It("should update a replica with source when it was updated", func() {
			Eventually(func(g Gomega) {
				var replica corev1.ConfigMap
				err := k8sClient.Get(ctx, ctrlclient.ObjectKey{Namespace: "testing", Name: sourceConfigMap.Name}, &replica)
				g.Expect(err).NotTo(HaveOccurred())

				replica.Data = map[string]string{
					"foo":   "bar",
					"lorem": "ipsum",
				}

				err = k8sClient.Update(ctx, &replica)
				g.Expect(err).NotTo(HaveOccurred())

				err = k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(&replica), &replica)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(reflect.DeepEqual(sourceConfigMap.Data, replica.Data)).To(BeFalse())
			}).Should(Succeed())
		})

		It("should update all replicas when the source object was updated", func() {
			Eventually(func(g Gomega) {
				var source corev1.ConfigMap
				err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(sourceConfigMap), &source)
				g.Expect(err).NotTo(HaveOccurred())

				source.Data["foo"] = "barbaz"

				err = k8sClient.Update(ctx, &source)
				g.Expect(err).NotTo(HaveOccurred())

				var replicaList corev1.ConfigMapList
				err = k8sClient.List(ctx, &replicaList, ctrlclient.MatchingLabels{
					replicator.SourceNameLabel:      source.GetName(),
					replicator.SourceNamespaceLabel: source.GetNamespace(),
				})
				g.Expect(err).NotTo(HaveOccurred())

				for _, replica := range replicaList.Items {
					g.Expect(replica.Data["foo"]).To(Equal("barbaz"))
				}
			}).Should(Succeed())
		})

		It("should create a new replica when a new namespace was added", func() {
			Eventually(func(g Gomega) {
				err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: newNamespace}})
				g.Expect(ctrlclient.IgnoreAlreadyExists(err)).NotTo(HaveOccurred())

				var replica corev1.ConfigMap
				err = k8sClient.Get(ctx, ctrlclient.ObjectKey{Name: sourceConfigMap.Name, Namespace: newNamespace}, &replica)
				g.Expect(err).NotTo(HaveOccurred())
			}).Should(Succeed())
		})

		It("should delete all replicas when the source object was deleted", func() {
			var source corev1.ConfigMap

			By("checking that source exists")
			Eventually(func(g Gomega) {
				getErr := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(sourceConfigMap), &source)
				g.Expect(ctrlclient.IgnoreNotFound(getErr)).NotTo(HaveOccurred())
			}).WithTimeout(5 * time.Second).Should(Succeed())

			By("deleting source")
			err := k8sClient.Delete(ctx, &source)
			Expect(err).NotTo(HaveOccurred())

			By("checking that replicas were deleted")
			Eventually(func(g Gomega) int {
				var replicaList corev1.ConfigMapList
				err := k8sClient.List(ctx, &replicaList, ctrlclient.MatchingLabels{
					replicator.SourceNameLabel:      sourceConfigMap.GetName(),
					replicator.SourceNamespaceLabel: sourceConfigMap.GetNamespace(),
				})
				g.Expect(err).NotTo(HaveOccurred())

				return len(replicaList.Items)
			}).Should(Equal(0))
		})
	})
})
