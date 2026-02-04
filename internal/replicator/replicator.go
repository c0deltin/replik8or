package replicator

import (
	"context"
	"fmt"

	"github.com/c0deltin/replik8or/internal/config"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Replicator[T client.Object] struct {
	client client.Client
	config *config.Config
}

func New[T client.Object](client client.Client, config *config.Config) *Replicator[T] {
	return &Replicator[T]{
		client: client,
		config: config,
	}
}

func (r *Replicator[T]) CreateOrUpdate(ctx context.Context, source, replica T) error {
	res, err := controllerutil.CreateOrUpdate(ctx, r.client, replica, func() error {
		return CopyFields(source, replica)
	})
	if err != nil {
		return fmt.Errorf("create or updating replica: %w", err)
	}

	lgr := log.FromContext(ctx).
		WithValues("source", NamespacedName(source), "replica", NamespacedName(replica))

	switch res {
	case controllerutil.OperationResultCreated:
		lgr.Info("created replica")
	case controllerutil.OperationResultUpdated:
		lgr.Info("updated replica")
	default:
		lgr.Info("replica already in place", "operation", res)
	}
	return nil
}

// CopyFields copy fields of source to replica object.
func CopyFields(source, replica client.Object) error {
	switch v := replica.(type) {
	case *corev1.Secret:
		v.Data = source.(*corev1.Secret).Data
		v.Type = source.(*corev1.Secret).Type
		v.Immutable = source.(*corev1.Secret).Immutable
	case *corev1.ConfigMap:
		v.Data = source.(*corev1.ConfigMap).Data
		v.BinaryData = source.(*corev1.ConfigMap).BinaryData
		v.Immutable = source.(*corev1.ConfigMap).Immutable
	default:
		return fmt.Errorf("type %T not implemented", v)
	}

	copyLabels(source, replica)
	copyAnnotations(source, replica)

	return nil
}

// copyLabels copies the source labels to the replica and sets a reference to the source object.
func copyLabels(source, replica client.Object) {
	labels := source.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[SourceNamespaceLabel] = source.GetNamespace()
	labels[SourceNameLabel] = source.GetName()
	replica.SetLabels(labels)
}

// copyAnnotations copies the source annotations to the replica, removes the replication annotations and set the
// replicated resourceVersion of the source object.
func copyAnnotations(source, replica client.Object) {
	annotations := source.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	delete(annotations, ReplicationAllowedAnnotation)
	delete(annotations, DesiredNamespacesAnnotation)
	// annotations[LastReplicationAnnotation] = time.Now().Format(time.RFC3339)
	annotations[SourceVersionAnnotation] = source.GetResourceVersion()
	replica.SetAnnotations(annotations)
}

func NamespacedName(object client.Object) client.ObjectKey {
	return client.ObjectKey{Namespace: object.GetNamespace(), Name: object.GetName()}
}
