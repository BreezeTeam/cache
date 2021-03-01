package cache

/**
 * @Description: 只读数据结构
 */
type ByteView struct{
	value []byte
}

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
