package source

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/c0deltin/replik8or/internal/replicator"
)

const (
	reflectionIndexField = "reflection.enabled"
	sourceFinalizer      = "replik8or.c0deltin.dev/source"
)

func (r *Reconciler[T]) SetupWithManager(name string, mgr manager.Manager) error {
	if err := r.setReflectionIndexer(mgr); err != nil {
		return err
	}

	return builder.ControllerManagedBy(mgr).
		Named(name).
		For(r.emptyObjectFn(), builder.WithPredicates(r.sourcePredicates())).
		Watches(
			r.emptyObjectFn(),
			handler.EnqueueRequestsFromMapFunc(r.enqueueReplicas),
			builder.WithPredicates(r.replicaPredicates()),
		).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.mapNamespacesToSources),
			builder.WithPredicates(r.namespacePredicates()),
		).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger {
			return ctrl.Log.WithName("replik8or")
		}).
		Complete(r)
}

func (r *Reconciler[T]) setReflectionIndexer(mgr manager.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		context.Background(),
		r.emptyObjectFn(),
		reflectionIndexField,
		func(object client.Object) []string {
			if v, ok := object.GetAnnotations()[replicator.ReplicationAllowedAnnotation]; ok && v == "true" {
				return []string{"true"}
			}
			return nil
		},
	)
}

func (r *Reconciler[T]) namespacePredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			namespace, ok := e.Object.(*corev1.Namespace)
			if !ok {
				return false
			}
			return namespace.Status.Phase == corev1.NamespaceActive
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			namespace, ok := e.ObjectNew.(*corev1.Namespace)
			if !ok {
				return false
			}
			return namespace.Status.Phase == corev1.NamespaceActive
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *Reconciler[T]) mapNamespacesToSources(ctx context.Context, _ client.Object) []reconcile.Request {
	var sourceList = r.emptyObjectListFn()
	if err := r.client.List(ctx, sourceList, client.MatchingFields{
		reflectionIndexField: "true",
	}); err != nil {
		return nil
	}

	sources, err := meta.ExtractList(sourceList)
	if err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, source := range sources {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(source.(client.Object))})
	}

	return requests
}

func (r *Reconciler[T]) replicaPredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return replicator.HasLabels(e.ObjectOld, replicator.SourceNamespaceLabel, replicator.SourceNameLabel)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return replicator.HasLabels(e.Object, replicator.SourceNamespaceLabel, replicator.SourceNameLabel)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *Reconciler[T]) enqueueReplicas(_ context.Context, object client.Object) []reconcile.Request {
	var result []reconcile.Request
	if replicator.HasLabels(object, replicator.SourceNamespaceLabel, replicator.SourceNameLabel) {
		labels := object.GetLabels()

		result = append(result, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: labels[replicator.SourceNamespaceLabel],
				Name:      labels[replicator.SourceNameLabel],
			},
		})
	}
	return result
}

func (r *Reconciler[T]) sourcePredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return replicator.HasAnnotations(e.Object, replicator.ReplicationAllowedAnnotation)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return replicator.HasAnnotations(e.ObjectOld, replicator.ReplicationAllowedAnnotation)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return replicator.HasAnnotations(e.Object, replicator.ReplicationAllowedAnnotation)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
