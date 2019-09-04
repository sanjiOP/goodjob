package master

import (
	"go.etcd.io/etcd/clientv3"
	"time"
	"github.com/sanjiOP/goodjob/common"
	"context"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

// work节点 服务发现
type WorkManage struct{
	// etcd 客户端
	ETCDClient *clientv3.Client
	// etcd kv操作资源
	ETCDkv clientv3.KV
}


// work节点列表
func ( workManage *WorkManage)list()(addrList []string,err error){

	var (
		getResponse *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		addr string
	)

	// 获取所有work节点
	if getResponse,err = workManage.ETCDkv.Get(context.TODO(),common.PREFIX_WORK_NODE_DIR,clientv3.WithPrefix());err != nil{
		return
	}
	addrList = make([]string,0)
	for _,kvPair = range getResponse.Kvs{
		addr = common.ExtractWorkIp(string(kvPair.Key))
		addrList = append(addrList,addr)
	}

	return
}



var (
	G_workManage *WorkManage
)

// 初始化
func InitWorkManage()(err error){


	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
	)

	// 定义配置项
	config = clientv3.Config{
		// 集群地址
		Endpoints:G_config.EtcdEndPoints,
		// 超时
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}


	// 创建etcd连接
	if client,err = clientv3.New(config);err != nil {
		return err
	}

	// 获取kv和lease对象
	kv = clientv3.NewKV(client)
	G_workManage = &WorkManage{ETCDClient:client,ETCDkv:kv}
	return
}
