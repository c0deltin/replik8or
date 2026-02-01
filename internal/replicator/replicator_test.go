package replicator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCopyFields(t *testing.T) {
	t.Run("Secret", func(t *testing.T) {
		source := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret",
				Namespace: "default",
				Annotations: map[string]string{
					"custom-annotation": "bar",
				},
				Labels: map[string]string{
					"custom-label": "bar",
				},
				ResourceVersion: "123",
			},
			Immutable: ptr.To(true),
			Data: map[string][]byte{
				".dockercfg": []byte("eyJhdXRocyI6eyJodHRwczovL2V4YW1wbGUvdjEvIjp7ImF1dGgiOiJvcGVuc2VzYW1lIn19fQo"),
			},
			Type: corev1.SecretTypeDockercfg,
		}

		var replica corev1.Secret
		replica.SetName(source.GetName())
		replica.SetNamespace("testing")
		err := CopyFields(source, &replica)

		var expected corev1.Secret
		source.DeepCopyInto(&expected)
		expected.ResourceVersion = ""  // reset
		expected.Namespace = "testing" // reset (namespace is usually set before CopyFields is called)
		expected.Labels[SourceNamespaceLabel] = source.GetNamespace()
		expected.Labels[SourceNameLabel] = source.GetName()
		expected.Annotations[SourceVersionAnnotation] = source.GetResourceVersion()

		assert.NoError(t, err)
		assert.True(t, reflect.DeepEqual(&expected, &replica))
	})
	t.Run("ConfigMap", func(t *testing.T) {
		source := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap",
				Namespace: "default",
				Annotations: map[string]string{
					"custom-annotation": "bar",
				},
				Labels: map[string]string{
					"custom-label": "bar",
				},
				ResourceVersion: "123",
			},
			Immutable: ptr.To(true),
			Data: map[string]string{
				"foo": "bar",
			},
			BinaryData: map[string][]byte{
				".dockercfg": []byte("eyJhdXRocyI6eyJodHRwczovL2V4YW1wbGUvdjEvIjp7ImF1dGgiOiJvcGVuc2VzYW1lIn19fQo"),
			},
		}

		var replica corev1.ConfigMap
		replica.SetName(source.GetName())
		replica.SetNamespace("testing")
		err := CopyFields(source, &replica)

		var expected corev1.ConfigMap
		source.DeepCopyInto(&expected)
		expected.ResourceVersion = ""  // reset
		expected.Namespace = "testing" // reset (namespace is usually set before CopyFields is called)
		expected.Labels[SourceNamespaceLabel] = source.GetNamespace()
		expected.Labels[SourceNameLabel] = source.GetName()
		expected.Annotations[SourceVersionAnnotation] = source.GetResourceVersion()

		assert.NoError(t, err)
		assert.True(t, reflect.DeepEqual(&expected, &replica))
	})
	t.Run("Unknown type", func(t *testing.T) {
		source := &corev1.Namespace{}
		replica := &corev1.Namespace{}

		err := CopyFields(source, replica)

		assert.Error(t, err)
	})
}

func TestNamespacedName(t *testing.T) {
	object := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap",
			Namespace: "default",
		},
	}

	objectKey := NamespacedName(object)

	assert.Equal(t, client.ObjectKey{Namespace: "default", Name: "configmap"}, objectKey)
}
