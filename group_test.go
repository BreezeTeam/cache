package cache

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"testing"
)

func TestGetter(t *testing.T) {
	//将一个匿名函数转换为GetterFunc类型
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		//key==value
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Fatal("callback failed")
	}
}

/**
 * @Description: 模拟一个数据库
 */
var db = map[string]int{
	"1":	630,
	"2":	589,
	"3": 	567,
}


func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))

	groupCache := NewGroup("db", 2<<10,
		GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[db] search key", key)

			if v, ok := db[key]; ok {
				//从db中加载数据,会被记录次数
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(strconv.Itoa(v)), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := groupCache.Get(k); err != nil || view.String() != strconv.Itoa(v) {
			t.Fatal("failed to get value of ",k)
		}
		//测试是否缓存失效
		if _, err := groupCache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := groupCache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}


func TestGetGroup(t *testing.T) {
	groupName := "db"
	NewGroup(groupName, 2<<10, GetterFunc(
		func(key string) (bytes []byte, err error) { return }))

	if group := GetGroup(groupName); group == nil || group.name != groupName {
		t.Fatalf("group %s not exist", groupName)
	}

	if group := GetGroup(groupName + " "); group != nil {
		t.Fatalf("expect nil, but %s got", group.name)
	}
}
