package consistentHash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type HashFunc func(data []byte) uint32

type Map struct {
	hashFunc            HashFunc       // 哈希函数
	virtualNodeReplicas int            // 虚拟节点倍数
	hashRing            []int          // 哈希环
	nodeMap             map[int]string // 虚拟节点与真实节点的映射表
}

func NewMap(replicas int, fn HashFunc) *Map {
	m := &Map{
		hashFunc:            fn,
		virtualNodeReplicas: replicas,
		nodeMap:             make(map[int]string),
	}
	// 通过依赖注入的方式，实现替换哈希函数
	if m.hashFunc == nil {
		// 返回的是一个uint32类型的值，范围从0到 2^32 - 1
		m.hashFunc = crc32.ChecksumIEEE
	}
	return m
}

// Add 用于添加真实的节点，允许传入 0 或多个真实节点的名称
func (m *Map) Add(keys ...string) {
	// 遍历所有真实节点名称
	for _, key := range keys {
		// 为每个节点设置虚拟节点
		for i := 0; i < m.virtualNodeReplicas; i++ {
			// 计算虚拟节点的哈希值，虚拟节点的名称构成：strconv.Itoa(i) + key （由虚拟节点索引和真实节点名称组成）
			hashValue := int(m.hashFunc([]byte(strconv.Itoa(i) + key)))
			// 将计算出的哈希值添加到哈希环中
			m.hashRing = append(m.hashRing, hashValue)
			// 在虚拟节点与真实节点的映射表中，添加该虚拟节点和真实节点的映射关系
			m.nodeMap[hashValue] = key
		}
	}
	sort.Ints(m.hashRing)
}

// Get 用于从哈希环中选择节点
func (m *Map) Get(key string) string {
	if len(m.hashRing) == 0 {
		return ""
	}
	// 使用定义的哈希函数计算输入 key 的哈希值
	hashValue := int(m.hashFunc([]byte(key)))
	// 进行二分查找，寻找第一个大于或等于给定哈希值（hashValue）的元素，并返回元素在哈希环中的索引（idx）
	// sort.Search 函数返回满足闭包函数条件的第一个最小值的索引
	idx := sort.Search(len(m.hashRing), func(i int) bool {
		// i 是哈希环（m.hashRing）中的一个索引。
		// 判断当前索引 i 所指向的哈希值是否大于或等于目标 hashValue
		return m.hashRing[i] >= hashValue
	})
	// idx%len(m.hashRing): 确保索引在哈希环的有效范围内
	// m.hashRing[idx%len(m.hashRing)]: 根据索引取出哈希环中的哈希值
	// m.nodeMap[m.hashRing[idx%len(m.hashRing)]]: 使用哈希值作为 key，在节点映射表中查找真实的节点
	return m.nodeMap[m.hashRing[idx%len(m.hashRing)]]
}
