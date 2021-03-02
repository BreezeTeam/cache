package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"test/cache"
)

var db = map[string]int{
	"1":	630,
	"2":	589,
	"3": 	567,
}

func main()  {
	cache.NewGroup("test",2<<10,cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("db search key: " + key)
			if v,ok := db[key];ok {
				return []byte(strconv.Itoa(v)),nil
			}
			return nil,fmt.Errorf("%s not exist",key)
		},
	))

	addr:="localhost:7777"
	handler:=cache.NewGroupHTTP(addr)
	log.Println("GroupHTTP for cache is running at ",addr)
	log.Fatal(http.ListenAndServe(addr,handler))
}