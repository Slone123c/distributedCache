package lru

// ByteView 私有字段 b，外部代码不能直接访问或者修改
type ByteView struct {
	b []byte
}

func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回数据的副本而不是引用
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

// 函数确保创建一个新的切片并将所有数据从原始切片复制过去。
// 这样外部对这个切片的修改不会影响原始数据
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
