package collection

type Duplicate[T any] struct {
	First          T
	Duplicate      T
	DuplicateIndex int
}

func FindDuplicatesBy[T any, K comparable](items []T, key func(T) K) []Duplicate[T] {
	seen := make(map[K]T, len(items))
	var duplicates []Duplicate[T]

	for index, item := range items {
		itemKey := key(item)
		first, exists := seen[itemKey]
		if !exists {
			seen[itemKey] = item
			continue
		}

		duplicates = append(duplicates, Duplicate[T]{
			First:          first,
			Duplicate:      item,
			DuplicateIndex: index,
		})
	}

	return duplicates
}
