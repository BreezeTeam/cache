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