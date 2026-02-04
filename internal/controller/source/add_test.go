package source

import (
	"context"
	"testing"

	"github.com/c0deltin/replik8or/internal/replicator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconciler_enqueueReplicas(t *testing.T) {
	r := Reconciler[*corev1.ConfigMap]{}

	replica := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "replica-name",
			Namespace: "replica-namespace",
			Labels: map[string]string{
				replicator.SourceNamespaceLabel: "source-namespace",
				replicator.SourceNameLabel:      "source-name",
			},
		},
	}

	expected := []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      "source-name",
				Namespace: "source-namespace",
			},
		},
	}

	actual := r.enqueueReplicas(context.TODO(), replica)

	assert.Equal(t, expected, actual)
}
