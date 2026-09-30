package repository

func deduplicate[T any](items []T, keyFunc func(T) string) []T {
	if len(items) == 0 {
		return items
	}

	lastIndex := make(map[string]int, len(items))
	for i, item := range items {
		lastIndex[keyFunc(item)] = i
	}

	result := make([]T, 0, len(lastIndex))
	for i, item := range items {
		if i == lastIndex[keyFunc(item)] {
			result = append(result, item)
		}
	}

	return result
}
