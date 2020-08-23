package v1alpha1

// FilterApplicationResource returns an array of resources in src, but not in target
func FilterApplicationResource(src, target []ApplicationResource) []ApplicationResource {
	res := make([]ApplicationResource, 0, len(src))
	m := map[string]int{}

	calcKey := func(r *ApplicationResource) string {
		return r.GroupVersionKind().String() + "/" + r.Namespace + "/" + r.Name
	}

	for i := range target {
		m[calcKey(&target[i])] = i
	}

	for i := range src {
		_, ok := m[calcKey(&src[i])]

		if !ok {
			res = append(res, src[i])
		}
	}

	return res
}
