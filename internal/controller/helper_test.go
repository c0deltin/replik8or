package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/c0deltin/replikor/internal/utils"
)

func TestAddAnnotation(t *testing.T) {
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-object",
			Namespace: "default",
		},
	}

	AddAnnotation(obj, "my-annotation", "a-value")

	assert.Contains(t, obj.Annotations, "my-annotation")
	assert.Equal(t, obj.Annotations["my-annotation"], "a-value")
}

func TestCopyFields(t *testing.T) {
	t.Run("corev1.Secret", func(t *testing.T) {
		source := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret",
				Namespace: "default",
			},
			Immutable: utils.Ptr(true),
			Data: map[string][]byte{
				"testValue1": []byte("foobar"),
				"fooBar":     []byte("ASuperSecretValue"),
			},
		}

		replica := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret",
				Namespace: "replica-namespace",
			},
			Immutable: utils.Ptr(false),
			Data: map[string][]byte{
				"differentField": []byte("asdf123"),
			},
		}

		CopyFields(source, replica)

		assert.Equal(t, replica.Annotations[ReplicatorSourceAnnotation], fmt.Sprintf("%s/%s", source.GetNamespace(), source.GetName()))
		assert.Contains(t, replica.Finalizers, ReplicatorFinalizer)
		assert.Equal(t, source.Immutable, replica.Immutable)
		assert.Equal(t, source.Data, replica.Data)
	})
	t.Run("corev1.ConfigMap", func(t *testing.T) {
		source := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-configmap",
				Namespace: "default",
			},
			Immutable: utils.Ptr(true),
			Data: map[string]string{
				"testValue1": "foobar",
				"fooBar":     "aSuperDooperVlaue13",
			},
			BinaryData: map[string][]byte{
				"binary": []byte("ThisCouldBeYourBinaryValue!"),
			},
		}

		replica := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-configmap",
				Namespace: "replica-namespace",
			},
			Immutable: utils.Ptr(false),
			Data: map[string]string{
				"differentField": "asdf123",
			},
		}

		CopyFields(source, replica)

		assert.Equal(t, replica.Annotations[ReplicatorSourceAnnotation], fmt.Sprintf("%s/%s", source.GetNamespace(), source.GetName()))
		assert.Contains(t, replica.Finalizers, ReplicatorFinalizer)
		assert.Equal(t, source.Immutable, replica.Immutable)
		assert.Equal(t, source.Data, replica.Data)
		assert.Equal(t, source.BinaryData, replica.BinaryData)
	})
}
