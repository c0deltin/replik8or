package controller

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReplicatorFinalizer                   = "c0deltin.io/replik8or"
	ReplicatorAllowedAnnotation           = "replik8or.c0deltin.io/reflection-allowed"
	ReplicatorAllowedNamespacesAnnotation = "replik8or.c0deltin.io/allowed-namespaces"
	ReplicatorSourceAnnotation            = "replik8or.c0deltin.io/replica-of"
	ReplicateScheduleRequeue              = "replik8or.c0deltin.io/schedule-requeue"
)

// AddAnnotation adds an annotation to an object.
func AddAnnotation(object client.Object, key, value string) {
	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	object.SetAnnotations(annotations)
}

// CopyFields copy fields of source to replica object.
func CopyFields(source, replica client.Object) {
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
