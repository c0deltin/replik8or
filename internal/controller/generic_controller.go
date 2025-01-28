package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/c0deltin/replikor/internal/config"
)

type GenericReconciler[T client.Object] struct {
	client.Client
}

func (r *GenericReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	object := r.makeEmptyObject()
	if err := r.Get(ctx, req.NamespacedName, object); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if IsSecretType(object, config.StrSlice("ignore_secret_types")) {
		return ctrl.Result{}, nil
	}

	if _, ok := object.GetAnnotations()[ReplicateScheduleRequeue]; ok {
		delete(object.GetAnnotations(), ReplicateScheduleRequeue)
		return ctrl.Result{}, fmt.Errorf("updating, remove annotation %s: %w ", ReplicateScheduleRequeue, r.Update(ctx, object))
	}

	if !object.GetDeletionTimestamp().IsZero() {
		return r.finalizeAndDelete(ctx, object)
	}

	if controllerutil.AddFinalizer(object, ReplicatorFinalizer) {
		if err := r.Update(ctx, object); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return r.reconcile(ctx, object)
}

func (r *GenericReconciler[T]) makeEmptyObject() client.Object {
	var e T
	var obj client.Object = e

	switch obj.(type) {
	case *corev1.Secret:
		obj = new(corev1.Secret)
	case *corev1.ConfigMap:
		obj = new(corev1.ConfigMap)
	}
	return obj
}

func (r *GenericReconciler[T]) reconcile(ctx context.Context, object client.Object) (ctrl.Result, error) {
	source, isReplica, err := r.isReplica(ctx, object)
	if err != nil {
		return ctrl.Result{}, err
	}
	if isReplica {
		err = r.createOrUpdate(ctx, source, object.GetNamespace())
		return ctrl.Result{}, err
	}

	targetNamespaces, err := r.targetNamespaces(ctx, object)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, targetNamespace := range targetNamespaces {
		if err := r.createOrUpdate(ctx, object, targetNamespace.Name); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GenericReconciler[T]) isReplica(ctx context.Context, obj client.Object) (client.Object, bool, error) {
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
		if k8serrors.IsNotFound(err) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("failed to get source of replicated object: %w", err)
	}

	return &source, true, nil
}

func (r *GenericReconciler[T]) createOrUpdate(ctx context.Context, source client.Object, targetNamespace string) error {
	replica := r.makeEmptyObject()
	replica.SetName(source.GetName())
	replica.SetNamespace(targetNamespace)

	var found bool
	if err := r.Get(ctx, types.NamespacedName{Name: replica.GetName(), Namespace: replica.GetNamespace()}, replica); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		found = true
	}

	CopyFields(source, replica)

	var err error
	if found {
		err = r.Create(ctx, replica)
	} else {
		err = r.Update(ctx, replica)
	}

	if err != nil {
		return fmt.Errorf("found / update replica %s (source: %s) in namespace %s: %w", source.GetName(), replica.GetName(), replica.GetNamespace(), err)
	}
	return nil
}

func (r *GenericReconciler[T]) targetNamespaces(ctx context.Context, obj client.Object) ([]corev1.Namespace, error) {
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
		if ns.Name != obj.GetNamespace() && !slices.Contains(config.StrSlice("disallowed_namespaces"), ns.Name) {
			targetNS = append(targetNS, ns)
		}
	}

	return targetNS, nil
}

func (r *GenericReconciler[T]) allowedNamespaces(ctx context.Context, obj client.Object) ([]corev1.Namespace, error) {
	v, ok := obj.GetAnnotations()[ReplicatorAllowedNamespacesAnnotation]
	if !ok {
		return nil, nil
	}

	var namespaces []corev1.Namespace
	for _, ns := range strings.Split(v, ",") {
		if ns != obj.GetNamespace() && !slices.Contains(config.StrSlice("disallowed_namespaces"), ns) {
			var namespace corev1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: ns}, &namespace); err != nil {
				return nil, err
			}
			namespaces = append(namespaces, namespace)
		}
	}

	return namespaces, nil
}

func (r *GenericReconciler[T]) finalizeAndDelete(ctx context.Context, object client.Object) (ctrl.Result, error) {
	source, isReplica, err := r.isReplica(ctx, object)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !isReplica {
		targetNamespaces, err := r.targetNamespaces(ctx, object)
		if err != nil {
			return ctrl.Result{}, err
		}

		for _, targetNamespace := range targetNamespaces {
			if err := r.deleteObject(ctx, object, targetNamespace.Name); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if controllerutil.RemoveFinalizer(object, ReplicatorFinalizer) {
		if err := r.Update(ctx, object); err != nil {
			return ctrl.Result{}, err
		}
	}

	if isReplica && source != nil {
		AddAnnotation(source, ReplicateScheduleRequeue, "true")
		if err := r.Update(ctx, source); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GenericReconciler[T]) deleteObject(ctx context.Context, object client.Object, namespace string) error {
	var replica client.Object
	switch object.(type) {
	case *corev1.Secret:
		replica = &corev1.Secret{}
	case *corev1.ConfigMap:
		replica = &corev1.ConfigMap{}
	}

	if err := r.Get(ctx, types.NamespacedName{Name: object.GetName(), Namespace: namespace}, replica); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.Delete(ctx, replica)
}

func (r *GenericReconciler[T]) SetupWithManager(mgr ctrl.Manager, name string) error {
	logger := log.FromContext(context.Background()).WithName(name)

	watchFn := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.GetAnnotations()[ReplicatorAllowedAnnotation]
			if ok {
				logger.Info("CREATED", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			}
			return ok
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.GetAnnotations()[ReplicatorAllowedAnnotation]
			if ok {
				logger.Info("DELETED", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
				return true
			}
			_, ok = e.Object.GetAnnotations()[ReplicatorSourceAnnotation]
			if ok {
				logger.Info("DELETED", "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			}
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.ObjectNew.GetAnnotations()[ReplicatorAllowedAnnotation]
			if ok {
				logger.Info("UPDATED", "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
				return true
			}
			_, ok = e.ObjectNew.GetAnnotations()[ReplicatorSourceAnnotation]
			if ok {
				logger.Info("UPDATED", "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			}
			return ok
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(r.makeEmptyObject(), builder.WithPredicates(watchFn)).
		Complete(r)
}
