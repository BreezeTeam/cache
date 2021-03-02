package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultPrefix ="/_cache/"

type GroupHTTP struct {
	addr string
	prefix string
}

func NewGroupHTTP(addr string)*GroupHTTP  {
	return &GroupHTTP{
		addr: addr,
		prefix: defaultPrefix,
	}
}

/**
 * @Description: 将GroupHTTP 实现为Http Handler接口
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
 * @Description: 日志
 * @receiver g
 * @param format
 * @param args
 */
func (g *GroupHTTP) Log(format string, args ...interface{}){
	log.Printf("[GroupHTTP %s] %s", g.addr, fmt.Sprintf(format, args...))
}