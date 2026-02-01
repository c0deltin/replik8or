package replicator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEmptyConfigMap(t *testing.T) {
	configMap := EmptyConfigMap()
	assert.Equal(t, &corev1.ConfigMap{}, configMap)
}

func TestEmptyConfigMapList(t *testing.T) {
	configMapList := EmptyConfigMapList()
	assert.Equal(t, &corev1.ConfigMapList{}, configMapList)
}

func TestEmptySecret(t *testing.T) {
	secret := EmptySecret()
	assert.Equal(t, &corev1.Secret{}, secret)
}

func TestEmptySecretList(t *testing.T) {
	secretList := EmptySecretList()
	assert.Equal(t, &corev1.SecretList{}, secretList)
}
