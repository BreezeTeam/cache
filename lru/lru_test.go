package lru

import (
	"reflect"
	"testing"
)

/**
 * @Description: 定义一个新类型
 */
type String string
func (s String)Len()int{
	return len(s)
}

func TestGet(t *testing.T) {
	lru:=New(int64(0),nil)
	lru.Add("key1",String("1234"))
	if v,ok := lru.Get("key1");!ok || string(v.(String)) !="1234"{
		t.Fatalf("cache hit key1=1234 faild")
	}
	if _,ok := lru.Get("key2");ok{
		t.Fatalf("cache miss key2 faild")
	}
}

func TestRemove(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

func TestOnDelete(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		println(key,value.Len())
		keys = append(keys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}

func TestAdd(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key", String("1"))
	lru.Add("key", String("111"))

	if lru.usedbytes != int64(len("key")+len("111")) {
		t.Fatal("expected 6 but got", lru.usedbytes)
	}
}
