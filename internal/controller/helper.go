package controller

import (
	"fmt"
	"slices"

    corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddAnnotation(object client.Object, key, value string) {
	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	object.SetAnnotations(annotations)
}

func CopyFields(replica, source client.Object) {
	switch v := replica.(type) {
	case *corev1.Secret:
		v.Immutable = source.(*corev1.Secret).Immutable
		v.Data = source.(*corev1.Secret).Data
	case *corev1.ConfigMap:
		v.Immutable = source.(*corev1.ConfigMap).Immutable
		v.Data = source.(*corev1.ConfigMap).Data
		v.BinaryData = source.(*corev1.ConfigMap).BinaryData
	}
	AddAnnotation(replica, ReplicatorSourceAnnotation, fmt.Sprintf("%s/%s", source.GetNamespace(), source.GetName()))

	finalizers := replica.GetFinalizers()
	if !slices.Contains(finalizers, ReplicatorFinalizer) {
		finalizers = append(finalizers, ReplicatorFinalizer)
	}
	replica.SetFinalizers(finalizers)
}