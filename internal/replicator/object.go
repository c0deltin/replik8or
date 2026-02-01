package replicator

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func EmptyConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{}
}

func EmptyConfigMapList() client.ObjectList {
	return &corev1.ConfigMapList{}
}

func EmptySecret() *corev1.Secret {
	return &corev1.Secret{}
}

func EmptySecretList() client.ObjectList {
	return &corev1.SecretList{}
}
