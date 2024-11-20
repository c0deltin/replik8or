package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"slices"
	"strings"
)

const (
	ReplicatorFinalizer                   = "c0deltin.io/replik8or"
	ReplicatorAllowedAnnotation           = "replik8or.c0deltin.io/reflection-allowed"
	ReplicatorAllowedNamespacesAnnotation = "replik8or.c0deltin.io/allowed-namespaces"
	ReplicatorSourceAnnotation            = "replik8or.c0deltin.io/replica-of"
	ReplicateScheduleRequeue              = "replik8or.c0deltin.io/schedule-requeue"
)

type DefaultController struct {
	client.Client

	DisallowedNamespaces []string
}

func (r *DefaultController) deleteAndFinalize(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	source, isReplica, err := r.isReplica(ctx, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !isReplica {
		targetNamespaces, err := r.targetNamespaces(ctx, obj)
		if err != nil {
			return ctrl.Result{}, err
		}

		for _, targetNamespace := range targetNamespaces {
			if err := r.deleteObject(ctx, obj, targetNamespace.Name); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if controllerutil.RemoveFinalizer(obj, ReplicatorFinalizer) {
		if err := r.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}

	if isReplica && source != nil {
		source.SetAnnotations(SetAnnotations(source.GetAnnotations(), ReplicateScheduleRequeue, "true"))
		if err := r.Update(ctx, source); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *DefaultController) deleteObject(ctx context.Context, obj client.Object, namespace string) error {
	var replica client.Object
	if _, v := obj.(*corev1.Secret); v {
		replica = &corev1.Secret{}
	}
	if _, v := obj.(*corev1.ConfigMap); v {
		replica = &corev1.Secret{}
	}

	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: namespace}, replica)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		return nil
	}
	if err := r.Delete(ctx, replica); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func (r *DefaultController) isReplica(ctx context.Context, obj client.Object) (client.Object, bool, error) {
	v, ok := obj.GetAnnotations()[ReplicatorSourceAnnotation]
	if !ok {
		return nil, false, nil
	}

	parts := strings.Split(v, "/")
	if len(parts) != 2 {
		return nil, false, fmt.Errorf("invalid source annotation, expected namespace/name but got: %s", v)
	}

	var source corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace: parts[0], Name: parts[1]}, &source); err != nil {
		if errors.IsNotFound(err) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("failed to get source of replicated object: %w", err)
	}

	return &source, true, nil
}

func (r *DefaultController) targetNamespaces(ctx context.Context, obj client.Object) ([]corev1.Namespace, error) {
	targetNS, err := r.allowedNamespaces(ctx, obj)
	if err != nil {
		return nil, err
	}
	if targetNS != nil {
		return targetNS, nil
	}

	var namespaces corev1.NamespaceList
	if err := r.List(ctx, &namespaces); err != nil {
		return nil, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name != obj.GetNamespace() && !slices.Contains(r.DisallowedNamespaces, ns.Name) {
			targetNS = append(targetNS, ns)
		}
	}

	return targetNS, nil
}

func (r *DefaultController) allowedNamespaces(ctx context.Context, obj client.Object) ([]corev1.Namespace, error) {
	v, ok := obj.GetAnnotations()[ReplicatorAllowedNamespacesAnnotation]
	if !ok {
		return nil, nil
	}

	var namespaces []corev1.Namespace
	for _, ns := range strings.Split(v, ",") {
		if ns != obj.GetNamespace() && !slices.Contains(r.DisallowedNamespaces, ns) {
			var namespace corev1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: ns}, &namespace); err != nil {
				return nil, err
			}
			namespaces = append(namespaces, namespace)
		}
	}

	return namespaces, nil
}
