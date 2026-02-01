package replicator

import (
	"context"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListTargetNamespaces returns a list of namespace names in which replicas should exist.
// It respects the annotation of the source object, the namespace of the source object itself which will be ignored
// and also the namespaces that are disallowed to have replicas by configuration.
func (r *Replicator[T]) ListTargetNamespaces(ctx context.Context, source T) ([]string, error) {
	var (
		targetNamespaces []string
		err              error
	)
	if HasAnnotations(source, DesiredNamespacesAnnotation) {
		targetNamespaces, err = r.desiredNamespaces(ctx, source)
	} else {
		targetNamespaces, err = r.clusterNamespaces(ctx)
	}
	if err != nil {
		return nil, err
	}

	for i, namespace := range targetNamespaces {
		if source.GetNamespace() == namespace || slices.Contains(r.config.DisallowedNamespaces, namespace) {
			targetNamespaces = slices.Delete(targetNamespaces, i, i+1)
		}
	}

	return targetNamespaces, nil
}

// desiredNamespaces extracts the namespaces set on resource by DesiredNamespacesAnnotation and checks for existing.
func (r *Replicator[T]) desiredNamespaces(ctx context.Context, source T) ([]string, error) {
	namespacesAnnotation, ok := source.GetAnnotations()[DesiredNamespacesAnnotation]
	if !ok {
		return nil, nil
	}

	var desiredNamespaces []string
	for _, desiredNamespace := range strings.Split(namespacesAnnotation, ",") {
		var namespace corev1.Namespace
		if err := r.client.Get(ctx, client.ObjectKey{Name: desiredNamespace}, &namespace); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		desiredNamespaces = append(desiredNamespaces, namespace.Name)
	}
	return desiredNamespaces, nil
}

// clusterNamespaces lists all namespaces within the cluster and returns a slice containing their names.
func (r *Replicator[T]) clusterNamespaces(ctx context.Context) ([]string, error) {
	var namespaceList corev1.NamespaceList
	if err := r.client.List(ctx, &namespaceList); err != nil {
		return nil, err
	}

	var namespaces = make([]string, len(namespaceList.Items))
	for i := range namespaceList.Items {
		namespaces[i] = namespaceList.Items[i].Name
	}

	return namespaces, nil
}
