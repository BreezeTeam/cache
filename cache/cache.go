package cache

import (
	"test/cache/lru"
	"errors"
	"sync"
)


/**
 * @Description: 用于实现并发控制的对lru的包装的结构体
 */
type cache struct {
	mutex sync.Mutex //互斥锁
	lru *lru.LRU
	maxBytes int64
}
/**
 * @Description: 包装了lru的Add()
 * @receiver c
 * @param key
 * @param value
 */
func (c *cache) add(key string, value ByteView){
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//延迟初始化
	if c.lru == nil{
		c.lru = lru.New(c.maxBytes,nil)
	}
	c.lru.Add(key,value)
}

/**
 * @Description: 包装了lru的Get()
 * @receiver c
 * @param key
 * @return value
 * @return ok
 */
func (c *cache)get(key string)(value ByteView,ok bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.lru == nil{
		return
	}
	if v,ok := c.lru.Get(key);ok{
		return v.(ByteView),ok
	}
	return
}


//Getter interface
type Getter interface {
	Get(key string) ([]byte, error)
}

//a Getter impl with a func type
type GetterFunc func(key string) ([]byte, error)
//Getter interface impl with GetterFunc
func (f GetterFunc) Get(key string) ([]byte, error){
	return f(key)
}

/**
 * @Description: GroupCache,控制缓存的存储和单机，分布式情况下的缓存存取服务
 * 主要是提供和外部进行交付的方法：Get
 */
type Group struct {
	name   string
	getter Getter
	cache  cache
}

/**
 * @param key
 * @return ByteView
 * @return error
 */
func (g *Group) Get(key string) (ByteView, error)  {
	if key == "" {
		return ByteView{},errors.New("key is required")
	}
	//从cache中查找缓存，存在则返回缓存值
	if v,ok :=g.cache.get(key);ok{
		return v,nil
	}
	//不存在就load缓存值
	return g.load(key)
}

/**
 * @Description: 单机场景下调用getLocally，很不是场景调用getFromPeer从其他节点获取数据
 * @receiver g
 * @param key
 * @return value
 * @return error
 */
func (g *Group) load(key string) (ByteView, error) {
	//单机场景
	return g.getLocally(key)
}

/**
 * @Description: 单机场景下的获取源数据的方法
 * @receiver g
 * @param key
 * @return ByteView
 * @return error
 */
func (g *Group) getLocally(key string) (ByteView, error) {
	//调用用户回调函数 g.getter.Get(key)，获取源数据
	bytes,err := g.getter.Get(key)
	if err != nil{
		return ByteView{},err
	}
	//将源数据包装为ByteView类型，然后保存
	value := ByteView{value: cloneBytes(bytes)}
	g.cache.add(key,value)
	return value,nil
}


