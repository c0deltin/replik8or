package replicator

import (
	"testing"

	"github.com/c0deltin/replik8or/internal/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReplicator_ListTargetNamespaces(t *testing.T) {
	fakeClient := fake.NewFakeClient()
	r := Replicator[*corev1.ConfigMap]{
		client: fakeClient,
		config: &config.Config{
			DisallowedNamespaces: []string{"disallowed-by-config"},
		},
	}

	err := fakeClient.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testing"}})
	assert.NoError(t, err)
	err = fakeClient.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	assert.NoError(t, err)
	err = fakeClient.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "disallowed-by-config"}})
	assert.NoError(t, err)
	err = fakeClient.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}})
	assert.NoError(t, err)

	t.Run("desired namespace annotations", func(t *testing.T) {
		source := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-name",
				Namespace: "source-namespaces",
				Annotations: map[string]string{
					DesiredNamespacesAnnotation: "testing,source-namespaces,foo,disallowed-by-config",
				},
			},
		}

		namespaces, err := r.ListTargetNamespaces(t.Context(), source)

		assert.NoError(t, err)
		assert.Equal(t, []string{"testing", "foo"}, namespaces)
	})

	t.Run("cluster namespaces", func(t *testing.T) {
		source := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-name",
				Namespace: "source-namespaces",
			},
		}

		namespaces, err := r.ListTargetNamespaces(t.Context(), source)

		assert.NoError(t, err)
		assert.Equal(t, []string{"foo", "kube-system", "testing"}, namespaces)
	})
}
