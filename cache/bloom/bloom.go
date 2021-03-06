package bloom

import (
	"crypto/sha256"
	"hash/fnv"
)

func fnvHash(data []byte) uint {
	m := fnv.New32()
	m.Write(data)
	return uint(m.Sum32())
}
/**
 * @Description: 布隆过滤器
 */
type BloomFilter struct {
	k uint
	bs []int64
	bsl uint
}
/**
 * @Description: 新建一个BloomFilter
 * @param n
 * @param k
 * @return *BloomFilter
 */
func NewBloomFilter(n uint,k uint) *BloomFilter {
	return &BloomFilter{
		k: k,
		bs:make([]int64,n/64+1),
		bsl: n/64+1,
	}
}

func (filter *BloomFilter) set(index uint) {
	index, bit := index/64, index%64
	filter.bs[index] |= 1 << bit
}

func (filter *BloomFilter) unSet(index uint) {
	index, bit := index/64, index%64
	filter.bs[index] ^= 1 << bit
}

func (filter *BloomFilter) isSet(index uint) bool {
	index, bit := index/64, index%64
	word := filter.bs[index]
	return (word | (1 << bit)) == word
}

//Put 添加一条记录
func (filter *BloomFilter) Put(data []byte){
	sha_data := sha256.Sum256(data)
	for i := 0; i < int(filter.k); i++ {
		data = sha_data[i:8+i]
		data =append(data,sha_data[24-i:24-i+8]...)
		filter.set(fnvHash(data)%filter.bsl)
	}
}

// Put 添加一条string记录
func (filter *BloomFilter) PutString(data string) {
	filter.Put([]byte(data))
}

// Has 推测记录是否已存在
func (filter *BloomFilter) Has(data []byte) bool {
	sha_data := sha256.Sum256(data)
	for i := 0; i < int(filter.k); i++ {
		data = sha_data[i:8+i]
		data =append(data,sha_data[24-i:24-i+8]...)
		if !filter.isSet(fnvHash(data) % filter.bsl) {
			return false
		}
	}
	return true
}

// Has 推测记录是否已存在
func (filter *BloomFilter) HasString(data string) bool {
	return filter.Has([]byte(data))
}