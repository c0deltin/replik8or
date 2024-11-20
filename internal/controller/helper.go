package controller

func SetAnnotations(annotations map[string]string, annotation, value string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[annotation] = value
	return annotations
}
