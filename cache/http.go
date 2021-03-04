package cache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"test/cache/consistenthash"
)
/**
 * @Description: 基于 http 的缓存服务器
 */

const defaultPrefix ="/cache/"
const defaultNodeVirReplicas =50


type GroupHTTP struct {
	//GroupHttp属性
	addr string
	prefix string

	//和分布式有关的
	mu sync.Mutex
	nodes *consistenthash.ConsistentHash
	NodeClientMap map[string]*httpClient
}
/**
 * @Description: 构造函数
 * @param addr
 * @return *GroupHTTP
 */
func NewGroupHTTP(addr string)*GroupHTTP  {
	return &GroupHTTP{
		addr: addr,
		prefix: defaultPrefix,
	}
}


/**
 * @Description: 设置节点和节点客户端的映射,并且会把节点添加到一致性hash上
 * @receiver g
 * @param nodeNames
 */
func (g *GroupHTTP) Set(nodeNames ...string)  {
	g.mu.Lock()
	defer g.mu.Unlock()

	//使用默认的hash函数
	g.nodes = consistenthash.New(defaultNodeVirReplicas,nil)
	g.nodes.Add(nodeNames...)

	//构造出节点客户端映射
	g.NodeClientMap = make(map[string]*httpClient,len(nodeNames))
	for _, nodeName := range nodeNames {
		g.NodeClientMap[nodeName] = &httpClient{baseURL: nodeName+g.prefix}
	}
}


/**
 * @Description: 将GroupHTTP 实现为 NodePicker,GroupHTTP 能够通过一致性hash根据key得到节点客户端
 * @receiver g
 * @param key
 * @return node
 * @return ok
 */
func (g *GroupHTTP ) PickNode(key string)(node NodeClient,ok bool){
	g.mu.Lock()
	defer g.mu.Unlock()

	if nodeName:=g.nodes.Get(key);nodeName !="" && nodeName != g.addr{
		g.Log("Pick node %s",nodeName)
		return g.NodeClientMap[nodeName],true
	}
	return nil,false
}



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


/**
 * @Description: 日志辅助函数
 * @receiver g
 * @param format
 * @param args
 */
func (g *GroupHTTP) Log(format string, args ...interface{}){
	log.Printf("[GroupHTTP %s] %s", g.addr, fmt.Sprintf(format, args...))
}




/**
 * @Description: Group HTTP Client,实现了NodeClient这个接口
 */
type httpClient struct {
	baseURL string
}

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