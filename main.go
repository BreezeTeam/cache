package main

import (
	"cache"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

var db = map[string]int{
	"1":	630,
	"2":	589,
	"3": 	567,
}

/**
 * @Description: 创建一个group
 * @return *cache.Group
 */
func createGroup() *cache.Group {
	return cache.NewGroup("test",2<<10,cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("db search key: " + key)
			if v,ok := db[key];ok {
				return []byte(strconv.Itoa(v)),nil
			}
			return nil,fmt.Errorf("%s not exist",key)
		},
	))
}

/**
 * @Description: 开始一个节点服务
 * @param addr
 * @param addrs
 * @param group
 */
func startCacheServer(addr string,addrs []string,group *cache.Group){

	//创建一个节点服务
	nodeServer:=cache.NewGroupHTTP(addr)

	//为该服务添加其他节点的信息到一致性hash上
	nodeServer.Set(addrs...)

	//为一个group注册一个节点服务,该节点服务能够支持分布式节点寻找的能力
	group.Register(nodeServer)

	log.Println("nodeServer for cache is running at ",addr)
	log.Fatal(http.ListenAndServe(addr[7:],nodeServer))
}

func  startAPIServer(apiAddr string,group *cache.Group)  {
	http.Handle("/api",http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			key:=request.URL.Query().Get("key")
			view,err :=group.Get(key)
			if err != nil {
				http.Error(writer,err.Error(),http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type","application/octet-stream")
			writer.Write(view.Copy())
		}))
	log.Println("apiServer for cache is running at ",apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:],nil))
}


/**
 * @Description: Test
```
1. go build -o server.go
2. ./server -port=8001
3. ./server -port=8002
4. ./server -port=8003 -api=1
5. curl "http://localhost:9999/api?key=1"
5. curl "http://localhost:8002/cache/test/1"
```
 */
func main()  {

	var (
		port int
		api bool
	)
	flag.IntVar(&port,"port",8001,"Node Server for Cache with port")
	flag.BoolVar(&api, "api", false, "Start API Server?")
	flag.Parse()

	apiAddr:="http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs,v)
	}

	group :=createGroup()
	if api{
		go startAPIServer(apiAddr,group)
	}
	startCacheServer(addrMap[port],addrs,group)
}