package worker

import (
	"go.etcd.io/etcd/clientv3"
	"context"
	"github.com/sanjiOP/goodjob/common"
)


// 分布式 任务锁（TXN事务）
type JobLock struct {
	// 任务名称
	JobName string
	// etcd kv操作资源
	ETCDkv clientv3.KV
	// etcd 租约
	ETCDLease clientv3.Lease
	// 用于终止自动续租
	CancelFunc context.CancelFunc
	// 租约id
	LeaseId clientv3.LeaseID
	// 是否上锁成功
	IsLocked bool
}


// 对任务上锁
func (jobLock *JobLock)TryLock()(err error)  {

	var(
		leaseGrantReponse *clientv3.LeaseGrantResponse
		cancelCtx context.Context
		cancelFunc context.CancelFunc
		leaseId clientv3.LeaseID
		keepChan <-chan *clientv3.LeaseKeepAliveResponse
		txn clientv3.Txn
		lockKey string
		txnResponse *clientv3.TxnResponse
	)

	// 1: 创建租约
	if leaseGrantReponse,err = jobLock.ETCDLease.Grant(context.TODO(),5);err != nil{
		return
	}
	// 用于取消租约
	cancelCtx,cancelFunc = context.WithCancel(context.TODO())

	// 2: 自动续租（保持这个锁一直存在）
	leaseId = leaseGrantReponse.ID
	if keepChan,err = jobLock.ETCDLease.KeepAlive(cancelCtx,leaseId);err != nil{
		goto FAIL
	}

	// 处理续租应答的协程
	go func() {
		var(
			keepResponse *clientv3.LeaseKeepAliveResponse
		)
		for{
			select {
			case keepResponse = <- keepChan:
				// 自动续租的应答:被cancel掉，跳出循环
				if keepResponse == nil{
					goto BREAK
				}
			}
		}
	BREAK:
	}()

	// 4: 创建事务txn
	txn = jobLock.ETCDkv.Txn(context.TODO())
	lockKey = common.PREFIX_JOB_LOCK_DIR + jobLock.JobName

	// 5：事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey),"=",0)).
		Then(clientv3.OpPut(lockKey,"1",clientv3.WithLease(leaseId))).
			Else(clientv3.OpGet(lockKey))
	// 提交事务
	if txnResponse,err = txn.Commit();err != nil{
		goto FAIL
	}

	// 6：成功返回|失败释放租约
	if !txnResponse.Succeeded{
		// 锁被占用  Else()
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	// 抢锁成功
	jobLock.LeaseId		= leaseId
	jobLock.CancelFunc	= cancelFunc
	jobLock.IsLocked	= true
	return
FAIL:
	// 取消续租
	cancelFunc()
	// 释放租约
	jobLock.ETCDLease.Revoke(context.TODO(),leaseId)
	return
}


// 任务释放锁
func (jobLock *JobLock)UnLock()(err error)  {
	if jobLock.IsLocked{
		// 取消续租
		jobLock.CancelFunc()
		// 释放租约
		jobLock.ETCDLease.Revoke(context.TODO(),jobLock.LeaseId)
	}

	return
}



// 初始化任务锁
func InitJobLock(jobName string,kv clientv3.KV,lease clientv3.Lease)(jobLock *JobLock)  {
	jobLock = &JobLock{
		JobName:jobName,
		ETCDkv:kv,
		ETCDLease:lease,
	}
	return
}







