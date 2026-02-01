package replicator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasAnnotations(t *testing.T) {
	object := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ReplicationAllowedAnnotation: "true",
				DesiredNamespacesAnnotation:  "namespace",
			},
		},
	}

	t.Run("matching annotations", func(t *testing.T) {
		assert.True(t, HasAnnotations(object, ReplicationAllowedAnnotation))
	})
	t.Run("not matching", func(t *testing.T) {
		assert.False(t, HasAnnotations(object, ReplicationAllowedAnnotation, DesiredNamespacesAnnotation, "unknown"))
	})
}

func TestHasLabels(t *testing.T) {
	object := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				SourceNameLabel:      "source-name",
				SourceNamespaceLabel: "source-namespace",
			},
		},
	}

	t.Run("matching labels", func(t *testing.T) {
		assert.True(t, HasLabels(object, SourceNameLabel))
	})
	t.Run("not matching", func(t *testing.T) {
		assert.False(t, HasLabels(object, SourceNameLabel, SourceNamespaceLabel, "unknown"))
	})
}
