package cache

import (
	"sync"
	"test/cache/singleflight"
)
/**
 * @Description: 通过全局变量groups,进行group的创建,管理等操作,直接面向用户
 * @return unc
 */

var(
	rwm sync.RWMutex //读写锁
	groups = make(map[string]*Group)
)

/**
 * @Description: 新建一个group，并添加到groups中，加了排它锁
 * @param name
 * @param maxBytes
 * @param getter
 * @return *Group
 */
func NewGroup(name string,maxBytes int64,getter Getter)  *Group {
	if getter == nil{
		panic("Group Getter cannot be nil")
	}
	//排他锁
	rwm.Lock()
	defer rwm.Unlock()
	g:=&Group{
		name:   name,
		getter: getter,
		cache:  cache{maxBytes: maxBytes},
		remoteCache: cache{maxBytes: maxBytes},
		loader: &singleflight.Group{},
	}
	groups[name]=g
	return g
}

/**
 * @Description: 通过groupname 从groups中获取group，加了共享锁
 * @param name
 * @return *Group
 */
func GetGroup(name string) *Group {
	//共享锁，防止读取到脏的数据
	rwm.RLock()
	defer rwm.RUnlock()
	g:= groups[name]
	return g
}