package cache

/**
 * @Description: 实现了View接口的缓存结构体
 */

/**
 * @Description: 只读数据结构
 */
type ByteView struct{
	value []byte
}

/**
 * @Description: 将ByteView 实现为 View
 * @receiver v ByteView
 * @return int
 */
func (v ByteView) Len() int{
	return len(v.value)
}

/**
 * @Description: 返回数据的String形式
 * @receiver v ByteView
 * @return string
 */
func (v ByteView)String() string{
	return string(v.value)
}

/**
 * @Description: 返回对象的拷贝
 * @receiver v ByteView
 * @return []byte
 */
func (v ByteView) Copy() []byte{
	return cloneBytes(v.value)
}

func cloneBytes(b []byte) []byte {
	c:=make([]byte, len(b))
	copy(c,b)
	return c
}
