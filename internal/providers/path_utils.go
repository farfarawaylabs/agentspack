package providers

func mergeUniquePaths(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, path := range base {
		seen[path] = struct{}{}
	}

	for _, path := range extra {
		if _, exists := seen[path]; exists {
			continue
		}
		base = append(base, path)
		seen[path] = struct{}{}
	}

	return base
}
