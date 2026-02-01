package source

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/c0deltin/replik8or/internal/config"
	"github.com/c0deltin/replik8or/internal/replicator"
)

type Reconciler[T client.Object] struct {
	client client.Client
	config *config.Config

	emptyObjectFn     func() T
	emptyObjectListFn func() client.ObjectList

	replicator *replicator.Replicator[T]
}

func NewReconciler[T client.Object](
	client client.Client,
	config *config.Config,
	emptyObjectFn func() T,
	emptyObjectListFn func() client.ObjectList,
) *Reconciler[T] {
	return &Reconciler[T]{
		client:            client,
		config:            config,
		emptyObjectFn:     emptyObjectFn,
		emptyObjectListFn: emptyObjectListFn,
		replicator:        replicator.New[T](client, config),
	}
}

func (r *Reconciler[T]) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	var source = r.emptyObjectFn()
	if err := r.client.Get(ctx, req.NamespacedName, source); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !source.GetDeletionTimestamp().IsZero() {
		return r.finalizeAndDelete(ctx, source)
	}

	if controllerutil.AddFinalizer(source, sourceFinalizer) {
		if err := r.client.Update(ctx, source); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{RequeueAfter: time.Second}, nil
	}

	targetNamespaces, err := r.replicator.ListTargetNamespaces(ctx, source)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, targetNamespace := range targetNamespaces {
		var replica = r.emptyObjectFn()
		replica.SetName(source.GetName())
		replica.SetNamespace(targetNamespace)

		if err := r.replicator.CreateOrUpdate(ctx, source, replica); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler[T]) finalizeAndDelete(ctx context.Context, source client.Object) (reconcile.Result, error) {
	var replicaList = r.emptyObjectListFn()
	if err := r.client.List(ctx, replicaList, client.MatchingLabels{
		replicator.SourceNameLabel:      source.GetName(),
		replicator.SourceNamespaceLabel: source.GetNamespace(),
	}); err != nil {
		return reconcile.Result{}, err
	}

	replicas, err := meta.ExtractList(replicaList)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, replica := range replicas {
		if err := r.client.Delete(ctx, replica.(client.Object)); err != nil {
			return reconcile.Result{}, err
		}
	}

	if controllerutil.RemoveFinalizer(source, sourceFinalizer) {
		if err := r.client.Update(ctx, source); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
