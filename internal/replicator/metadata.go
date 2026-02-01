package replicator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SourceNameLabel      = "replicator.c0deltin.dev/source-name"
	SourceNamespaceLabel = "replicator.c0deltin.dev/source-namespace"

	ReplicationAllowedAnnotation = "replik8or.c0deltin.dev/replication-allowed"
	DesiredNamespacesAnnotation  = "replik8or.c0deltin.dev/desired-namespaces"

	LastReplicationAnnotation = "replik8or.c0deltin.dev/last-replication"
	SourceVersionAnnotation   = "replik8or.c0deltin.dev/source-version"
)

func HasAnnotations(object client.Object, annotations ...string) bool {
	return matchingItems(object.GetAnnotations(), annotations...)
}

func HasLabels(object client.Object, labels ...string) bool {
	return matchingItems(object.GetLabels(), labels...)
}

func matchingItems(m map[string]string, items ...string) bool {
	var matching int
	for _, label := range items {
		if _, ok := m[label]; ok {
			matching++
		}
	}
	return matching == len(items)
}
