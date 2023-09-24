package cache

type NodeSelector interface {
	// SelectNode 根据一个键（key）来选择一个节点（Node）
	SelectNode(key string) (nodeGetter NodeGetter, ok bool)
}

type NodeGetter interface {
	// Get 从某个分组（group）和一个键（key）中获取数据
	Get(group string, key string) ([]byte, error)
}
