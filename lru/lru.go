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