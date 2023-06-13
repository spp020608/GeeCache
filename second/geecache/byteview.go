package geecache

// 只读数据结构 ByteView 用来表示缓存值
// 是 GeeCache 主要的数据结构之一
type ByteView struct {
	b []byte
}

// Len返回ByteView的长度
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice返回一个复制切片字节.
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String 返回字节数组转成字符串
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
