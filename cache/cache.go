package cache

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"test/cache/lru"
	"test/cache/singleflight"
	"time"
)

/**
 * @Description: 提供了将lru包装为group对象的能力
 */

//Getter interface
type Getter interface {
	Get(key string) ([]byte, error)
}

/**
 * @Description: a Getter impl with a func type,主要目的是为了将函数转为Getter 接口,方便调用
 * @param key
 * @return []byte
 * @return error
 */
type GetterFunc func(key string) ([]byte, error)
//Getter interface impl with GetterFunc
func (f GetterFunc) Get(key string) ([]byte, error){
	return f(key)
}
/**
 * @Description: 原子加
 * @param l
 * @param r
 */
func atomic_plus(l *uint32, r uint32) {
	atomic.AddUint32(l, r)
}

/**
 * @Description: 远程请求节点状态
 */
type keyStatus struct {
	firstGetTime time.Time //第一次请求的时间
	remoteCnt  uint32 //请求次数
}


/**
 * @Description: GroupCache,控制缓存的存储和 单机|分布式情况下的缓存存取服务
 * 主要是提供和外部进行交付的方法：Get
 */
type Group struct {
	name   string
	/**
     * @Description: getter是一个Getter接口,必须实现Get方法
     */
	getter Getter
	cache  cache //主要的缓存

	/**
     * @Description: 一个Group,具有一个NodePicker,能够根据传的key,以及节点客户端得到响应的节点
     */
	nodePicker NodePicker

	/**
     * @Description: 使用singleflight来防止缓存击穿
     */
	loader *singleflight.Group

	/**
     * @Description: 热点互备功能
     */
	remoteCache cache //随机缓存远程调用的结果
	//hotCache cache //热点缓存
	keyStatusMap map[string]*keyStatus
}



/**
 * @Description: 注册节点选择器
 * @receiver g
 * @param nodePicker
 */
func (g *Group) Register(nodePicker NodePicker)  {
	/**
	 * @Description: 只能注册一次
	 */
	if g.nodePicker !=nil{
		panic("RegisterNodePicker called more than once")
	}
	g.nodePicker = nodePicker
}

/**
 * @Description: 会利用getter,调用这个接口的Get函数来获取缓存,如果不存在,缓存失效,需要加载缓存
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
	//从remoteCache中查找数据,存在则返回缓存值
	if v,ok:=g.remoteCache.get(key);ok{
		return v,nil
	}
	//从hotCache中查找数据,存在则返回缓存值
	//if v,ok:=g.hotCache.get(key);ok{
	//	return v,nil
	//}
	//不存在就load缓存值
	return g.load(key)
}


/**
 * @Description: 缓存失效时调用,单机场景下调用getLocally，远程场景调用getFromPeer从其他节点获取数据
 * @receiver g
 * @param key
 * @return value
 * @return error
 */
func (g *Group) load(key string) (value ByteView,err error) {
	view,err :=g.loader.Do(key, func() (interface{}, error) {
		//remote调用
		if g.nodePicker !=nil {
			if nodeClient,ok := g.nodePicker.PickNode(key);ok{
				if value,err=g.getRemote(nodeClient,key);err == nil{
					return value,nil
				}
				log.Println("[Cache] Faild to get remote from nodeClient",err)
			}
		}
		//单机场景
		return g.getLocally(key)
	})
	if err != nil {
		return view.(ByteView),nil
	}
	return
}

/**
 * @Description: 通过nodeClient,能够根据group的名字和具体的key,查询到具体的缓存数据
 * @receiver g
 * @param nodeClient
 * @param key
 * @return ByteView
 * @return error
 */
func (g *Group) getRemote(nodeClient NodeClient,key string)(ByteView,error)  {
	bytes,err :=nodeClient.Get(g.name,key)
	if err != nil {
		return ByteView{},err
	}
	//将远程获取到的数据添加在remoteCache中
	value := ByteView{value: cloneBytes(bytes)}
	if rand.Intn(10) == 0{
		g.remoteCache.add(key,value)
	}
	return value,nil
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

