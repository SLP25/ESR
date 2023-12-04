package utils


func ContainsKey[K comparable, V any](dict map[K]V, field K) bool {
	_, ok := dict[field]
	return ok
}

func GetAnyKey[K comparable, V any](dict map[K]V, defaultKey K) K {
	for k := range dict {
		return k
	}

	return defaultKey
}

func GetKeys[K comparable, V any](dict map[K]V) []K {
	keys := make([]K, len(dict))

	i := 0
	for k := range dict {
		keys[i] = k
		i++
	}

	return keys
}