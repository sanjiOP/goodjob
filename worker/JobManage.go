package worker

import (
	"time"
	"context"
	"go.etcd.io/etcd/clientv3"
	"github.com/sanjiOP/goodjob/common"
	"go.etcd.io/etcd/mvcc/mvccpb"

)

// 任务管理
type JobManage struct {
	// etcd 客户端
	ETCDClient *clientv3.Client
	// etcd kv操作资源
	ETCDkv clientv3.KV
	// etcd 租约
	ETCDLease clientv3.Lease
	// etcd watcher
	ETCDWatcher clientv3.Watcher
}

// 监听任务变化
func(jobManage *JobManage) watchJob()(err error){

	var (
		getResponse *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		job *common.Job
		watchStartVersion int64
		watchChan clientv3.WatchChan
		watchResponse clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName string
		jobEvent *common.JobEvent
	)

	// 1:get 所有任务，并且获知集群reversion
	if getResponse,err = jobManage.ETCDkv.Get(context.TODO(),common.PREFIX_JOB_DIR,clientv3.WithPrefix());err != nil{
		return
	}
	for _,kvPair = range getResponse.Kvs{
		if job,err = common.UnpackJob(kvPair.Value);err !=nil{
			continue
		}
		jobEvent = common.JobEventFactory(common.JOB_EVENT_SAVE,job)
		//worker启动，先从etcd中同步所有任务，推送到scheduler协程
		G_scheduler.PushJobEvent(jobEvent)
	}


	// 2:从该reversion开始监听job变化，监听协程
	go func(){
		watchStartVersion = getResponse.Header.Revision + 1
		watchChan = jobManage.ETCDWatcher.Watch(context.TODO(),common.PREFIX_JOB_DIR,clientv3.WithRev(watchStartVersion),clientv3.WithPrefix())

		// 处理监听事件
		for watchResponse = range watchChan{

			for _,watchEvent = range watchResponse.Events{
				switch watchEvent.Type {
				case mvccpb.PUT:
					// 推送更新事件
					if job,err = common.UnpackJob(watchEvent.Kv.Value);err != nil{
						continue
					}
					jobEvent = common.JobEventFactory(common.JOB_EVENT_SAVE,job)
				case mvccpb.DELETE:
					// 推送删除事件
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					jobEvent = common.JobEventFactory(common.JOB_EVENT_DELETE,&common.Job{Name:jobName})
				}
				// 推送事件 给scheduler协程
				G_scheduler.PushJobEvent(jobEvent)

			}

		}

	}()
	return
}


// 监听是否强制杀死任务
func (jobManage *JobManage)watchKill()(err error)  {

	var (
		watchChan clientv3.WatchChan
		watchResponse clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName string
		jobEvent *common.JobEvent
	)


	go func(){
		watchChan = jobManage.ETCDWatcher.Watch(context.TODO(),common.PREFIX_JOB_KILL_DIR,clientv3.WithPrefix())

		// 处理监听事件
		for watchResponse = range watchChan{

			for _,watchEvent = range watchResponse.Events{
				switch watchEvent.Type {
				case mvccpb.PUT:
					// 杀死任务事件
					jobName		= common.ExtractKillerName(string(watchEvent.Kv.Key))
					jobEvent	= common.JobEventFactory(common.JOB_EVENT_KILL,&common.Job{Name:jobName})
					// 推送事件 给scheduler协程
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE:
					// killer标记过期，被自动删除
				}

			}

		}

	}()


	return
}



// 创建任务锁
func(jobManage *JobManage) CreateJobLock(jobName string)(jobLock *JobLock){
	jobLock = InitJobLock(jobName,jobManage.ETCDkv,jobManage.ETCDLease)
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
		watcher clientv3.Watcher
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
	watcher = clientv3.NewWatcher(client)

	// 赋值单例
	G_jobManage = &JobManage{client, kv, lease,watcher,}


	// 启动任务变化监听
	G_jobManage.watchJob()

	// 启动任务强杀监听
	G_jobManage.watchKill()

	return

}


