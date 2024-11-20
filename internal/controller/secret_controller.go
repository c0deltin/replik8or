package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type SecretReconciler struct {
	*DefaultController
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//logger := log.FromContext(ctx).WithValues("configmap", req.NamespacedName)

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if _, ok := secret.GetAnnotations()[ReplicateScheduleRequeue]; ok {
		delete(secret.GetAnnotations(), ReplicateScheduleRequeue)
		return ctrl.Result{}, r.Update(ctx, &secret)
	}

	if !secret.GetDeletionTimestamp().IsZero() {
		return r.deleteAndFinalize(ctx, &secret)
	}

	if controllerutil.AddFinalizer(&secret, ReplicatorFinalizer) {
		if err := r.Update(ctx, &secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	source, isReplica, err := r.isReplica(ctx, &secret)
	if err != nil {
		return ctrl.Result{}, err
	}
	if isReplica {
		err = r.replicate(ctx, source, secret.Namespace)
		return ctrl.Result{}, err
	}

	targetNamespaces, err := r.targetNamespaces(ctx, &secret)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, targetNamespace := range targetNamespaces {
		if err := r.replicate(ctx, &secret, targetNamespace.Name); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) replicate(ctx context.Context, source client.Object, targetNamespace string) error {
	replica := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:       source.GetName(),
			Namespace:  targetNamespace,
			Finalizers: []string{ReplicatorFinalizer},
		},
	}

	err := r.Get(ctx, types.NamespacedName{Name: replica.Name, Namespace: replica.Namespace}, &replica)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	replica.Annotations = SetAnnotations(replica.GetAnnotations(), ReplicatorSourceAnnotation, fmt.Sprintf("%s/%s", source.GetNamespace(), source.GetName()))
	replica.Immutable = source.(*corev1.Secret).Immutable
	replica.Data = source.(*corev1.Secret).Data

	if errors.IsNotFound(err) {
		err = r.Create(ctx, &replica)
	} else {
		err = r.Update(ctx, &replica)
	}

	return err
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := log.FromContext(context.Background())

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
		For(&corev1.Secret{}, builder.WithPredicates(
			watchFn,
		)).
		//Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(
		//	func(ctx context.Context, object client.Object) []reconcile.Request {
		//		_, ok := object.GetAnnotations()[ReplicatorSourceAnnotation]
		//		if ok {
		//			logger.Info("sourced config map found", "name", object.GetName(), "namespace", object.GetNamespace())
		//			return []reconcile.Request{
		//				{types.NamespacedName{Name: object.GetName(), Namespace: object.GetNamespace()}},
		//			}
		//		}
		//		return nil
		//	},
		//), builder.WithPredicates(watchFn)).
		Complete(r)
}
