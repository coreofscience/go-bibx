package union

// UnionFind represents an union find data structure.
type UnionFind[K comparable] struct {
	parent map[K]K
	rank   map[K]int
}

// New creates a new Union find data structure.
func New[K comparable](items []K) *UnionFind[K] {
	u := &UnionFind[K]{
		parent: make(map[K]K),
		rank:   make(map[K]int),
	}
	for _, item := range items {
		u.parent[item] = item
		u.rank[item] = 0
	}
	return u
}

// Find returns the root of the set containing item.
func (u *UnionFind[K]) Find(item K) K {
	parent, ok := u.parent[item]
	if !ok {
		return item
	}
	if parent != item {
		u.parent[item] = u.Find(parent)
	}
	return u.parent[item]
}

// Union merges the sets containing items a and b.
func (u *UnionFind[K]) Union(a, b K) {
	rootA := u.Find(a)
	rootB := u.Find(b)
	if rootA == rootB {
		return
	}
	if u.rank[rootA] < u.rank[rootB] {
		u.parent[rootA] = rootB
		return
	}
	if u.rank[rootA] > u.rank[rootB] {
		u.parent[rootB] = rootA
		return
	}
	u.parent[rootA] = rootB
	u.rank[rootA]++
}

// Connected returns true if items a and b are connected.
func (u *UnionFind[K]) Connected(a, b K) bool {
	rootA := u.Find(a)
	rootB := u.Find(b)
	return rootA == rootB
}

// Components returns the connected components.
func (u *UnionFind[K]) Components() [][]K {
	connected := make(map[K][]K)
	for item := range u.parent {
		root := u.Find(item)
		connected[root] = append(connected[root], item)
	}
	components := make([][]K, 0, len(connected))
	for _, items := range connected {
		components = append(components, items)
	}
	return components
}
