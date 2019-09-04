package master

import (
	"time"
	"context"
	"encoding/json"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"github.com/sanjiOP/goodjob/common"
)

// 任务管理
type JobManage struct {
	// etcd 客户端
	ETCDClient *clientv3.Client
	// etcd kv操作资源
	ETCDkv clientv3.KV
	// etcd 租约
	ETCDLease clientv3.Lease
}





func (jobManage *JobManage) SaveJob(job *common.Job)(oldJob *common.Job,err error){

	var (
		jobKey string
		jobValue []byte
		putResponse *clientv3.PutResponse
	)

	//
	jobKey = common.PREFIX_JOB_DIR + job.Name
	if jobValue,err = json.Marshal(job);err != nil{
		return
	}

	// 写入 etcd
	if putResponse,err = jobManage.ETCDkv.Put(context.TODO(),jobKey,string(jobValue),clientv3.WithPrevKV());err != nil{
		return
	}

	// 如果更新，返回旧值
	if putResponse.PrevKv != nil{
		if err = json.Unmarshal(putResponse.PrevKv.Value,&oldJob);err != nil{
			err = nil
			return
		}
	}

	return
}


// 删除任务
func (jobManage *JobManage) DeleteJob(jobName string)(oldJob *common.Job,err error){

	var (
		jobKey string
		deleteResponse *clientv3.DeleteResponse
	)

	// 删除
	jobKey = common.PREFIX_JOB_DIR + jobName
	if deleteResponse,err = jobManage.ETCDkv.Delete(context.TODO(),jobKey,clientv3.WithPrevKV());err != nil{
		return
	}

	// 如果删除成功，返回旧值
	if len(deleteResponse.PrevKvs) != 0{
		if err = json.Unmarshal(deleteResponse.PrevKvs[0].Value,&oldJob);err != nil{
			err = nil
			return
		}
	}

	return
}


// 任务列表
func (jobManage *JobManage) ListJob()(jobList []*common.Job,err error){

	var (
		getResponse *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		job *common.Job
	)
	// 获取所有任务列表
	if getResponse,err = jobManage.ETCDkv.Get(context.TODO(),common.PREFIX_JOB_DIR,clientv3.WithPrefix());err != nil{
		return
	}

	// 解析 etcd中的任务
	jobList = make([]*common.Job,0)
	for _,kvPair = range getResponse.Kvs{
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value,job);err != nil{
			err = nil
			continue
		}
		jobList = append(jobList,job)
	}

	return
}


// 杀死任务
func (jobManage *JobManage) KillJob(jobName string)(err error){

	var(
		etcdKey string
		leaseGrantResponse *clientv3.LeaseGrantResponse
	)
	etcdKey = common.PREFIX_JOB_KILL_DIR + jobName

	// 创建租约过期时间，让 work监听到变化即可，无需保留该值 1000毫秒
	if leaseGrantResponse,err = jobManage.ETCDLease.Grant(context.TODO(),1000);err != nil{
		return
	}

	// 更新任务状态
	if _,err = jobManage.ETCDkv.Put(context.TODO(),etcdKey,"1",clientv3.WithLease(leaseGrantResponse.ID)); err != nil{
		return
	}

	return
}

// 单例
var (
	G_jobManage *JobManage
)

// 初始化
func InitJobManage() (err error){

	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
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
	kv		= clientv3.NewKV(client)
	lease	= clientv3.NewLease(client)

	// 赋值单例
	G_jobManage = &JobManage{client, kv, lease,}
	return

}


