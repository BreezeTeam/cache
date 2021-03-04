package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

/**
 * @Description: 函数类型Hash,可以实现自定义的Hash算法
 * @param data
 * @return uint32
 */
type Hash func(data []byte) uint32

/**
 * @Description: 一致性hash
 */
type ConsistentHash struct{
	hash Hash //hash func
	nodeVirReplicas int
	virNodeMap map[int]string // map(hash(vir node),node)
	keys[] int //hash rings
}

/**
 * @Description: New ConsistentHash,default hash is crc32.ChecksumIEEE
 * @param NodeVirReplicas
 * @param fn
 * @return *ConsistentHash
 */
func New(nodeVirReplicas int, fn Hash)*ConsistentHash{
	c := &ConsistentHash{
		nodeVirReplicas:nodeVirReplicas,
		hash:fn,
		virNodeMap:make(map[int]string),
	}
	if c.hash == nil{
		c.hash = crc32.ChecksumIEEE
	}
	return c
}

func (c *ConsistentHash ) Add(nodeNames ...string)  {
	for _,nodeName:=range nodeNames{
		for i := 0; i < c.nodeVirReplicas; i++ {
			virHashCode := int( c.hash( []byte( strconv.Itoa(i)+nodeName )) )
			c.keys = append( c.keys,virHashCode)
			c.virNodeMap[virHashCode] = nodeName
		}
	}
	//sort keys on rins
	sort.Ints(c.keys)
}

/**
 * @Description: 根据key,从一致性hash算法中的环上得到虚拟节点,又映射到真实节点
 * @receiver c
 * @param key
 * @return string
 */
func (c *ConsistentHash)Get(key string) string{
	if len(c.keys)==0{
		return ""
	}
	keyHashCode := int(c.hash( []byte(key)))
	//binary search vir node
	indexOfKeys :=sort.Search(len(c.keys),func(i int)bool{
		return c.keys[i]>=keyHashCode
	})
	//if indexOfKeys==len(keys) ,then binary search not found ,bug our indexOfKeys is 0
	return c.virNodeMap[c.keys[indexOfKeys%len(c.keys)]]
}

