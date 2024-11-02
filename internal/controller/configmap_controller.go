package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ReplicatorFinalizer                   = "c0deltin.io/replik8or"
	ReplicatorAllowedAnnotation           = "replik8or.c0deltin.io/reflection-allowed"
	ReplicatorAllowedNamespacesAnnotation = "replik8or.c0deltin.io/allowed-namespaces"
	ReplicatorSourceAnnotation            = "replik8or.c0deltin.io/replica-of"
)

type ConfigMapReconciler struct {
	client.Client

	DisallowedNamespaces []string
}

func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &configMap); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !configMap.GetDeletionTimestamp().IsZero() {
		return r.deleteAndFinalize(ctx, &configMap)
	}

	if controllerutil.AddFinalizer(&configMap, ReplicatorFinalizer) {
		if err := r.Update(ctx, &configMap); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	targetNamespaces, err := r.targetNamespaces(ctx, configMap.Namespace, configMap.GetAnnotations())
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, targetNamespace := range targetNamespaces {
		replica := corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      configMap.Name,
				Namespace: targetNamespace.Name,
			},
		}

		err := r.Get(ctx, types.NamespacedName{Name: replica.Name, Namespace: replica.Namespace}, &replica)
		if err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		replica.Annotations = setAnnotations(replica.GetAnnotations(), configMap.Namespace, configMap.Name)
		replica.Immutable = configMap.Immutable
		replica.Data = configMap.Data
		replica.BinaryData = configMap.BinaryData

		if errors.IsNotFound(err) {
			err = r.Create(ctx, &replica)
		} else {
			err = r.Update(ctx, &replica)
		}

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func setAnnotations(annotations map[string]string, namespace, name string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[ReplicatorSourceAnnotation] = fmt.Sprintf("%s/%s", namespace, name)
	return annotations
}

func (r *ConfigMapReconciler) deleteAndFinalize(ctx context.Context, configMap *corev1.ConfigMap) (ctrl.Result, error) {
	targetNamespaces, err := r.targetNamespaces(ctx, configMap.Namespace, configMap.GetAnnotations())
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, targetNamespace := range targetNamespaces {
		var replica corev1.ConfigMap
		if err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: targetNamespace.Name}, &replica); err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		if err := r.Delete(ctx, &replica); err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	if controllerutil.RemoveFinalizer(configMap, ReplicatorFinalizer) {
		if err := r.Update(ctx, configMap); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) targetNamespaces(ctx context.Context, objNS string, annotations map[string]string) ([]corev1.Namespace, error) {
	targetNS, err := r.allowedNamespaces(ctx, objNS, annotations)
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
		if ns.Name != objNS && !slices.Contains(r.DisallowedNamespaces, ns.Name) {
			targetNS = append(targetNS, ns)
		}
	}

	return targetNS, nil
}

func (r *ConfigMapReconciler) allowedNamespaces(ctx context.Context, objNS string, annotations map[string]string) ([]corev1.Namespace, error) {
	v, ok := annotations[ReplicatorAllowedNamespacesAnnotation]
	if !ok {
		return nil, nil
	}

	var namespaces []corev1.Namespace
	for _, ns := range strings.Split(v, ",") {
		if ns != objNS && !slices.Contains(r.DisallowedNamespaces, ns) {
			var namespace corev1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: ns}, &namespace); err != nil {
				return nil, err
			}
			namespaces = append(namespaces, namespace)
		}
	}

	return namespaces, nil
}

func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(func(o client.Object) bool {
				log.FromContext(context.Background()).Info("got object", "name", o.GetName(), "namespace", o.GetNamespace())
				_, ok := o.GetAnnotations()[ReplicatorAllowedAnnotation] // todo also allow source annotations to revert overrides
				return ok
			}),
		)).
		Complete(r)
}
