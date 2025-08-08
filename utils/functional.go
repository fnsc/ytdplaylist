package utils

func Map[T any, U any](slice []T, fn func(T) U) []U {
	result := make([]U, 0, len(slice))
	for _, item := range slice {
		result = append(result, fn(item))
	}
	return result
}

func FilterMap[T any, U any](slice []T, fn func(T) (U, bool)) []U {
	result := []U{}
	for _, item := range slice {
		if v, ok := fn(item); ok {
			result = append(result, v)
		}
	}
	return result
}
