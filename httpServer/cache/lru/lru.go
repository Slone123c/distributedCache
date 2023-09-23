package lru

import "container/list"

// 创建一个包含字典和双向链表的结构体类型 Cache

type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nbytes   int64                    // 当前已使用的内存
	ll       *list.List               // 双向链表
	cache    map[string]*list.Element // 存储缓存的键值对，键是字符串，值是双向链表中对应节点的指针
	// 记录被移除时的回调函数
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int // 用于返回值所占用的内存大小
}

// New 用于实例化 Cache
func New(maxBytes int64, onEnvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEnvicted,
	}
}

// Get 查找功能，主要步骤：1. 从字典找到对应双向链表的节点 2. 将节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 从哈希表c.cache中尝试获取与key相关联的列表节点element
	if element, ok := c.cache[key]; ok {
		// 如果找到了，将该节点移动到双向链表的头部（即最近使用的位置）
		c.ll.MoveToFront(element)
		// 从列表节点中提取出实际的数据
		kv := element.Value.(*entry)
		// 返回找到的值和true
		return kv.value, true
	}
	// 如果在缓存中没有找到相关的键，则返回默认值和false
	return
}

// RemoveOldest 缓存淘汰，移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	// 获取双向链表的最后一个元素，即最久未使用的元素
	element := c.ll.Back()
	if element != nil {
		// 从双向链表中删除该元素
		c.ll.Remove(element)
		// 将该元素的值转型为*entry类型，并赋值给kv
		kv := element.Value.(*entry)
		// 从哈希表中删除该键值对
		delete(c.cache, kv.key)
		// 更新当前缓存所使用的字节数
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 如果设置了淘汰回调，则执行
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 新增/修改节点
func (c *Cache) Add(key string, value Value) {
	// 判断节点（key-value对）是否已经存在于缓存中
	if element, ok := c.cache[key]; ok {
		// 如果存在，则移动该节点到双向链表的头部，表示该节点最近被访问或修改
		c.ll.MoveToFront(element)
		// 获取节点实际存储的数据，并转型为*entry
		kv := element.Value.(*entry)
		// 更新缓存的大小，新值的大小减去旧值的大小
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新节点的值
		kv.value = value
	} else {
		// 如果节点不存在，创建新的entry并添加到双向链表的头部
		element := c.ll.PushFront(&entry{
			key:   key,
			value: value,
		})
		// 将新节点添加到缓存的哈希表中
		c.cache[key] = element
		// 更新缓存的大小，包括key和value的大小
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 检查当前缓存的大小是否超过了设定的最大值，如果是则调用RemoveOldest()淘汰节点
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len 获取添加了多少条数据，方便测试
func (c *Cache) Len() int {
	return c.ll.Len()
}
