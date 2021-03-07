package cache

import pb "cache/cachepb"

//根据传的key选择响应的节点
type NodePicker interface {
	PickNode(key string)(node NodeClient,ok bool)
}

//GroupHTTP就是一个这个接口
type NodeClient interface {
	//从对应的group查找缓存
	Get(in *pb.Request,out *pb.Response)error
}
