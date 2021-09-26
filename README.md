# cache
cache by go

## 简介
学习[groupcache](https://github.com/golang/groupcache)实现的分布式KV缓存   

### groupcache
groupcache是memecached的作者实现的新的缓存系统  
从业务角度,有两点改变  
1. 引入了group概念
>在同一个缓存集群中,可能会存储F(x),G(x).如果没有group,我们就需要在key上面加上函数标识  
>同时由于两个函数位于同一个内存缓存集训,两个函数的缓存会相互淘汰.很难控制,结果可能不符合预期  
>引入Group后,可以针对不同的函数设置不同的group,每个group独立设置内存占用上限,分别进行group组内淘汰  

2. 值不可修改
>一旦某个x对应值为y,那么就一直为y  
>因此也就没有memcache中的缓存失效时间这种概念  

总结:groupcache适合于存储函数调用,后面也会涉及到这方面的内容
### ps.
>本项目为学习项目,谨慎用于生产有类似需求,请直接访问[groupcache](https://github.com/golang/groupcache)

## 缓存淘汰策略
groupcache是内存缓存,内存资源有限,需要在内存不够时,淘汰点一部分数据
### LRU算法
>最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，那么将来被访问的概率也会更高。LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

```go
package lru

import "container/list"

/**
 * @Description: LRU链表
 */
type LRU struct {
	maxBytes         int64                         //允许使用的最大内存
	usedbytes        int64                         //当前已经使用的内存
	doublyLinkedList *list.List                    //双向链表
	searchMap        map[string]*list.Element      //查询map
	onDelete         func(key string, value Value) //当一个值被删除时的回调函数
}

/**
 * @Description: Value接口
 */
type Value interface {
	Len() int //该接口必须包含Len(),用于返回值所占用的内存大小
}

/**
 * @Description: 键值对
 */
type entry struct {
	key   string
	value Value
}

func New(maxBytes int64, onDelete func(string, Value)) *LRU {
	return &LRU{
		maxBytes:         maxBytes,
		doublyLinkedList: list.New(),
		searchMap:        make(map[string]*list.Element),
		onDelete:         onDelete,
	}
}

func (lru *LRU) Get(key string) (value Value, ok bool) {
	if element, ok := lru.searchMap[key]; ok {
		//移动到队尾
		lru.doublyLinkedList.MoveToBack(element)
		kValue := element.Value.(*entry)
		//返回找到的值
		return kValue.value, true
	}
	return
}
func (lru *LRU) Add(key string, value Value) {
	//键存在，更新节点的值，并且移动到队尾。更新usedbytes
	if element, ok := lru.searchMap[key]; ok {
		lru.doublyLinkedList.MoveToBack(element)
		kValue := element.Value.(*entry)
		lru.usedbytes += int64(value.Len()) - int64(kValue.value.Len())
		kValue.value = value
	} else {
		//不存在就新增加，向队列添加新节点，并且向map添加映射关系，最后更新usedbytes
		element := lru.doublyLinkedList.PushBack(&entry{key, value})
		lru.searchMap[key] = element
		lru.usedbytes += int64(len(key)) + int64(value.Len())
	}
	//如果超过了maxBytes，进行缓存淘汰
	for lru.maxBytes != 0 && lru.maxBytes < lru.usedbytes {
		lru.Remove()
	}
}

func (lru *LRU) Remove() {
	//删除队首元素
	element := lru.doublyLinkedList.Front()
	if element != nil {
		lru.doublyLinkedList.Remove(element)
		kValue := element.Value.(*entry)
		//删除映射关系
		delete(lru.searchMap, kValue.key)
		//更新已用内存数
		lru.usedbytes -= int64(len(kValue.key)) + int64(kValue.value.Len())
		//调用回调函数
		if lru.onDelete != nil {
			lru.onDelete(kValue.key, kValue.value)
		}
	}
}

/**
 * @Description: 双链表的长度
 * @receiver lru
 * @return int
 */
func (lru *LRU) Len() int{
	return lru.doublyLinkedList.Len()
}
```

## LRU缓存并发控制
LRU缓存,没有任何的并发控制,在多个请求同时到来时,读写操作会出现冲突
### 互斥锁
>使用`sync.Mytex`互斥锁来封装LRU缓存中我们使用到的接口
```go
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
```

## 分布式多节点
支持分布式缓存,需要解决以下问题
1. 节点选择问题
>在一个分布式缓存中,节点的选择十分钟重要,几十个个节点中只有一个节点中真正缓存了数据  
每次请求只有几十分之一,如果节点选择错误,那么需要重新加载缓存数据,十分耗时  
也会导致缓存命中率降低 

hash算法通常能解决这个问题

2. 节点数量变化
>当节点数量变化时,普通的hash算法,就不能适应了  
>因为几乎所有的缓存值对应的节点都会变化,意味着大量的缓存失效,会造成缓存雪崩

>**缓存雪崩**：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。

解决办法是使用一致性hash算法

### 一致性hash
一致性hash算法将key映射到2^32的空间中,并且将这个空间连成环  
1. 将节点数据对应的hash值,映射到环上某处  
2. 计算key的hash值,映射到环上,并且顺时针找到的第一个节点,就是一致性hash算法选择的节点  

![一致性哈希添加节点 consistent hashing add peer](https://gitee.com/Euraxluo/images/raw/master/picgo/add_peer.jpg)

优点:
- 在增加/删除节点时,只需要重新定位该节点附近的一部分数据,不用重新定位所有的节点

缺点:
- 当节点数量比较少时,容易发生数据倾斜
    > 解决办法:为节点增加虚拟节点,扩充节点的数量,只需要维护虚拟节点和真实节点的映射关系

```go
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
 * @Description: 根据key,从一致性hash算法中的环上根据二分查找得到虚拟节点,又映射到真实节点
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
```

## 节点通信
分布式节点之前需要进行通信,才能进行数据交流

### HTTPServer
```go
/**
 * @Description: 将GroupHTTP 实现为Http Handler接口,能够提供HTTP服务
 * @receiver g
 * @param w
 * @param r
 */
func (g *GroupHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	g.Log("%s %s",r.Method,r.URL.Path)
	if !strings.HasPrefix(r.URL.Path, g.prefix){
		http.Error(w,"Bad Request",http.StatusBadRequest)
		return
	}

	//解析parts=>/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(g.prefix):],"/",2)
	if len(parts) != 2{
		http.Error(w,"Bad Request",http.StatusBadRequest)
		return
	}
	groupName:=parts[0]
	key:=parts[1]

	//find group by groupName
	group := GetGroup(groupName)
	if group ==nil{
		http.Error(w,"no such group: "+groupName,http.StatusNotFound)
		return
	}

	//get view by key from group
	view,err:=group.Get(key)
	if err!=nil{
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	//write view copy to http.request
	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(view.Copy())
}
```

### HTTPClient
```go
/**
 * @Description: 通过HTTP协议访问节点的HTTPServer的节点客户端实现
 * @receiver h
 * @param group
 * @param key
 * @return []byte
 * @return error
 */
func (h *httpClient)Get(group string,key string) ([]byte, error){
	//包装Get请求
	u:=fmt.Sprintf("%v%v/%v",h.baseURL,url.QueryEscape(group),url.QueryEscape(key))
	res,err := http.Get(u)
	if err != nil {
		return nil,err
	}
	defer res.Body.Close()

	if res.StatusCode!=http.StatusOK{
		return nil, fmt.Errorf("server returned:%v",res.Status)
	}

	bytes,err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v",err)
	}

	return bytes,nil
}
```
### 使用protobuf 来提高数据传输效率

```protobuf
//protoc --go_out=. *.proto
syntax = "proto3";
option go_package="./;cachepb";

package cachepb;

message Request {
  string group = 1;
  string key = 2;
}

message Response {
  bytes value = 1;
}

service GroupCache {
  rpc Get(Request) returns (Response);
}
```
```go
/**
 * @Description: 通过nodeClient,能够根据group的名字和具体的key,查询到具体的缓存数据
 * @receiver g
 * @param nodeClient
 * @param key
 * @return ByteView
 * @return error
 */
func (g *Group) getRemote(nodeClient NodeClient,key string)(ByteView,error)  {
	req:=&pb.Request{
		Group: g.name,
		Key: key,
	}
	res:=&pb.Response{}


	//bytes,err :=nodeClient.Get(g.name,key) //http 方式
	if err:=nodeClient.Get(req,res);err != nil {
		return ByteView{},err
	}

	//将远程获取到的数据添加在remoteCache中
	value := ByteView{value: cloneBytes(res.Value)}
	if rand.Intn(10) == 0{
		g.remoteCache.add(key,value)
	}
	return value,nil
}
```

GroupHttpServer and GroupHttpClient
```go
/**
 * @Description: 将GroupHTTP 实现为Http Handler接口,能够提供HTTP服务
 * @receiver g
 * @param w
 * @param r
 */
func (g *GroupHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	g.Log("%s %s",r.Method,r.URL.Path)
	if !strings.HasPrefix(r.URL.Path, g.prefix){
		http.Error(w,"Bad Request",http.StatusBadRequest)
		return
	}

	//解析parts=>/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(g.prefix):],"/",2)
	if len(parts) != 2{
		http.Error(w,"Bad Request",http.StatusBadRequest)
		return
	}
	groupName:=parts[0]
	key:=parts[1]

	//find group by groupName
	group := GetGroup(groupName)
	if group ==nil{
		http.Error(w,"no such group: "+groupName,http.StatusNotFound)
		return
	}

	//get view by key from group
	view,err:=group.Get(key)
	if err!=nil{
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	body, err := proto.Marshal(&pb.Response{Value: view.Copy()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//write view copy to http.request
	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(body)
}

/**
 * @Description: 通过HTTP协议访问节点的HTTPServer的节点客户端实现
 * @receiver h
 * @param group
 * @param key
 * @return []byte
 * @return error
 */
func (h *httpClient)Get(in *pb.Request,out *pb.Response)error{
	//包装Get请求
	u:=fmt.Sprintf("%v%v/%v",h.baseURL,url.QueryEscape(in.GetGroup()),url.QueryEscape(in.GetKey()))
	res,err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode!=http.StatusOK{
		return fmt.Errorf("server returned:%v",res.Status)
	}

	bytes,err := ioutil.ReadAll(res.Body)//这里是空的

	if err != nil {
		return  fmt.Errorf("reading response body:%v",err)
	}
	if err = proto.Unmarshal(bytes,out);err != nil{
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
```

## 缓存击穿防护
>**缓存击穿**：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增 

主要解决方案有两种:
1. 缓存永不过期
>缓存中缓存值不设置ttl,而是由业务模块设置逻辑过期,当过期时间到达时,由业务模块异步去更新缓存  
>该方法在其他缓存系统中会出现,我们的缓存本来就永不过期,已经做了这个工作

2.加互斥锁,阻塞排队
>在其他请求在加载数据时,设置mutex锁,让其他请求阻塞,当加载完数据后再释放mutex锁,其他请求,直接从缓存中获取数据  
>该方法能够提高请求效率,同时减少数据加载,对数据源进行防护  
>其他类似的请求排队,限流算法还有: 1.计数 2.滑动窗口 3.  令牌桶Token Bucket 4.漏桶leaky bucket

3. 二级缓存
>C1为原始缓存,过期时间很短,C2为拷贝缓存,过期事件很长  
>该方法在其他系统中会出现,但是我们本来一级缓存就永不过期


总结:我们需要实现互斥锁的方法来进行防护,这种结构叫做[singleflight](golang.org/x/sync/singleflight)

```go
package singleflight

import "sync"

//call 代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入
type call struct {
	wg sync.WaitGroup
	val interface{}
	err error
}
/**
 * @Description: Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)。
 */
type Group struct {
	mu sync.Mutex
	callMap map[string]*call
}

/**
 * @Description: 针对相同的key,不论DO被调用多少次,函数fn都只会被调用一次
 * @receiver g
 * @param key
 * @param fn
 * @param error)
 * @return v
 * @return err
 * @return shared
 */
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error){
	g.mu.Lock()//要准备操作callMap,先加一个互斥锁
	if g.callMap == nil {
		g.callMap = make(map[string]*call)
	}

	// 当前key的函数正在被执行
	if c, ok := g.callMap[key]; ok {
		g.mu.Unlock()
		//这里别的协程在调用,那我我们就阻塞等待别的协成Done掉标记,然后进行把调用结果返回
		c.wg.Wait()//阻塞等待执行中的，等待其执行完毕后获取它的执行结果
		return c.val, c.err //c.val 是函数的运行结果
	}

	// 初始化一个call，往map中写后就解
	c := new(call)
	c.wg.Add(1)//调用fu前添加一个锁标记
	g.callMap[key] = c //将初始化的call添加到映射中,表示,我在
	g.mu.Unlock()

	// 执行获取key的函数，并将结果赋值给这个Call
	c.val, c.err = fn()
	c.wg.Done()//调用结束,去除说锁标记

	// 重新上锁操作callMap
	g.mu.Lock()
	delete(g.callMap, key)
	g.mu.Unlock()

	return c.val, c.err
}
```
## 缓存穿透防护
>**缓存穿透**：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

主要解决方案有以下两种:
1. 缓存空对象
>缓存穿透的原因是因为缓存中没有缓存这些空数据的key,导致这些请求跑到了数据源层,所以我们可以将这些空数据也进行缓存,这样就能拦截这些空查询  
>我们的缓存永不过期,缓存空对象会浪费大量内存,同时,这种请求是缓存不完的,该方法不能起到效果

2. 布隆过滤器
>当业务层有查询请求的时候，首先去BloomFilter中查询该key是否存在。若不存在，则说明数据库中也不存在该数据，因此缓存都不要查了，直接返回null。若存在，则继续执行后续的流程，先前往缓存中查询，缓存中没有的话再前往数据库中的查询。  
>分布式多节点缓存,意味着你需要在,每个节点都用布隆过滤器,而不容过滤器

~~总结:我们可以在apiServer那里设置BloomFilter,就能比较好的避免缓存穿透~~
实践:不能和缓存耦合,需要放到业务层
BloomFilter
```go
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
func newBloomFilter(n uint,k uint) *BloomFilter {
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
		data = append(data,sha_data[24-i:24-i+8]...)
        //这里用的是fnv32()算法,这种算法散列均匀,也比较快速,并且通过每次改变不同的位,实现多个hash函数的效果
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
```

 

## 热点互备
>当出现热点key时,很容易增加网络消耗,造成缓存效率降低的问题,groupcache也实现了热点互备

解决方案:
1. 随机缓存
>增加一个hotCache,同样是使用的lru缓存过期机制,在进行远程调用时,有一定的几率存储下这些hotCache  
>hotCache中存储的是节点远程调用过的key,可以避免频繁发生远程获取(但是这些key不是热点key)

```go
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
```

2. 判断为MinuteQPS较高的Key才进行缓存
```go
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
	//计算qps
	value := ByteView{value: cloneBytes(bytes)}

	if status,ok :=g.keyStatusMap[key];ok{
		atomic_plus(&status.remoteCnt,1)
		interval:=float64(time.Now().Unix()-status.firstGetTime.Unix())/60
		qps:=status.remoteCnt.Get() /int64(math.Max(1,math.Round(interval)))
		if qps >= maxMinuteRemoteQPS {
			g.hotCache.add(key,value)
			delete(g.keyStatusMap,key)
		}
	}else{
		g.keyStatusMap[key] = &keyStatus{
			firstGetTime: time.Now(),
			remoteCnt: 1,
		}
	}
	return value,nil
}
```