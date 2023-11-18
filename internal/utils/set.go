package utils

type set[V comparable] map[V]bool
type Set[V comparable] set[V]
	
func EmptySet[V comparable]() Set[V] {
	return make(map[V]bool)
}

func SetFrom[V comparable](elems ...V) Set[V] {
	ans := make(Set[V])
	for _, v := range elems {
		ans.Add(v)
	}
	return ans
}

func (this Set[V]) Add(val V) {
	this[val] = true
}

//returns true if the element was present in the set
func (this Set[V]) Remove(val V) bool {
	ans := this[val]
	delete(this, val)
	return ans
}

func (this Set[V]) Contains(val V) bool {
	return this[val]
}

func (this Set[V]) Length() int {
	return len(this)
}

func (this Set[V]) ToSlice() []V {
	ans := make([]V, len(this))
	i := 0
	for v := range this {
		ans[i] = v
		i++
	}
	return ans
}