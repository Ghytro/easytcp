package algo

func Find[T any](collection []T, iteratee func(item T) bool) (found T, ok bool) {
	if idx := IndexOf(collection, iteratee); idx != -1 {
		return collection[idx], true
	}
	return
}

func IndexOf[T any](collection []T, iteratee func(item T) bool) int {
	for i, el := range collection {
		if iteratee(el) {
			return i
		}
	}
	return -1
}
