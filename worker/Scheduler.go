package worker

import (
	"github.com/sanjiOP/goodjob/common"
	"time"
)

// 调度器
type Scheduler struct {

	// ETCD任务事件的队列
	jobEventChan chan *common.JobEvent

	// executor 执行完任务的队列
	jobResultChan chan *common.JobExecuteResult

	// 任务计划表（etcd中的任务推到调度器的内存中维护）
	jobPlanTable map[string]*common.SchedulerPlan

	// 任务执行表
	jobExecutingTable map[string]*common.JobExecuteInfo
}


// 任务变化后触发，维护调度器内存中的【任务计划表】
func (scheduler *Scheduler)handJobEvent(jobEvent *common.JobEvent){

	var(
		schedulerPlan *common.SchedulerPlan
		jobExecuteInfo *common.JobExecuteInfo
		jobExist bool
		err error
		jobExisted bool
	)

	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if schedulerPlan,err = common.SchedulerPlanFactory(jobEvent.Job);err != nil{
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = schedulerPlan
	case common.JOB_EVENT_DELETE:
		if schedulerPlan,jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name];jobExisted{
			delete(scheduler.jobPlanTable,jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		// 取消 shell command 的执行

		// 判断任务是否执行
		if jobExecuteInfo,jobExist = scheduler.jobExecutingTable[jobEvent.Job.Name];jobExist{
			jobExecuteInfo.CancelFunc()
		}

	}
}

// 任务执行结束后触发，维护调度器内存中的【任务执行表】
func (scheduler *Scheduler)handJobResult(result *common.JobExecuteResult){

	var(
		jobLogRecord *common.JobLogRecord
	)

	//fmt.Println("[执行结束]"," -[任务名称]",result.JobExecuteInfo.Job.Name," -[任务输出]",string(result.Output)," -[任务错误]",result.Err)
	delete(scheduler.jobExecutingTable,result.JobExecuteInfo.Job.Name)

	//
	//fmt.Println("[handJobResult] ",result.JobExecuteInfo.Job.Name)

	// 任务执行结束，写入mongodb(过滤掉抢锁的情况)
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED{
		jobLogRecord = &common.JobLogRecord{
			JobName:result.JobExecuteInfo.Job.Name,
			Command:result.JobExecuteInfo.Job.Command,
			Output:string(result.Output),
			PlanTime:result.JobExecuteInfo.PlanTime.UnixNano() /1000 /1000,
			SchedulerTime:result.JobExecuteInfo.RealTime.UnixNano() /1000 /1000,
			StartTime:result.StartTime.UnixNano() /1000 /1000,
			EndTime:result.EndTime.UnixNano() /1000 /1000,
		}
		if result.Err != nil{
			jobLogRecord.Err = result.Err.Error()
		}else{
			jobLogRecord.Err = ""
		}

		// 存储到mongodb中
		G_logSink.Append(jobLogRecord)


	}


}



// 执行任务
func (scheduler *Scheduler)TryStartJob(schedulerPlan *common.SchedulerPlan){
	//【notice】
	// 任务频率 1秒，任务运行时间 10秒
	// 任务在10秒内(执行中) 只能执行1次

	var(
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuteExist bool
	)
	// 如果任务正在执行，则跳过该任务
	if jobExecuteInfo,jobExecuteExist = scheduler.jobExecutingTable[schedulerPlan.Job.Name];jobExecuteExist{
		//fmt.Println("[正在执行]"," -[任务名称]",schedulerPlan.Job.Name)
		return
	}

	jobExecuteInfo = common.JobExecuteInfoFactory(schedulerPlan)
	scheduler.jobExecutingTable[schedulerPlan.Job.Name] = jobExecuteInfo


	// 执行任务
	//fmt.Println("[执行任务]"," -[任务名称]",schedulerPlan.Job.Name," -[plan time]",jobExecuteInfo.PlanTime," -[real time]",jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)

}

// 计算任务调度状态（扫描所有任务）
func (scheduler *Scheduler)TryScheduler()(schedulerAfter time.Duration){

	var(
		schedulerPlan *common.SchedulerPlan
		now time.Time
		nearTime *time.Time		// 指针类型，指针类型才可能为nil,值类型不允许nil
	)
	now = time.Now()

	// 如果没有调度任务，则sleep一秒钟
	if len(scheduler.jobPlanTable) == 0{
		schedulerAfter = 1 * time.Second
		return
	}


	// 1 遍历所有任务
	for _,schedulerPlan = range scheduler.jobPlanTable{

		if schedulerPlan.NextTime.Before(now) || schedulerPlan.NextTime.Equal(now){
			// 尝试执行任务
			scheduler.TryStartJob(schedulerPlan)

			// 更新下次执行时间
			schedulerPlan.NextTime = schedulerPlan.Expr.Next(now)
		}

		// 当前时间最近的任务
		if nearTime == nil || schedulerPlan.NextTime.Before(*nearTime) {
			nearTime = &schedulerPlan.NextTime
		}
	}

	// 下次调度的时间周期，【最近的调度时间 - 当前时间】
	schedulerAfter = (*nearTime).Sub(now)
	return

}


// 调度器主循环
func (scheduler *Scheduler)schedulerLoop(){

	var(
		jobEvent *common.JobEvent
		jobResult *common.JobExecuteResult
		schedulerAfter time.Duration
		schedulerTimer *time.Timer
	)

	// 一开始没有任务，默认为1秒
	schedulerAfter = scheduler.TryScheduler()
	schedulerTimer = time.NewTimer(schedulerAfter)

	for{
		select {
		case jobEvent = <-scheduler.jobEventChan:
			// job变化(put|delete|kill) 触发
			scheduler.handJobEvent(jobEvent)
		case jobResult = <-scheduler.jobResultChan:
			// job executor结束 触发
			scheduler.handJobResult(jobResult)
		case <- schedulerTimer.C:
			// 最近的任务到期 触发
		}

		// 调度一次任务(遍历一遍任务列表)
		schedulerAfter = scheduler.TryScheduler()
		schedulerTimer.Reset(schedulerAfter)
	}
}


// worker.watcher从etcd中获取任务变化
// 推送job事件到 scheduler
func (scheduler *Scheduler)PushJobEvent(jobEvent *common.JobEvent){
	scheduler.jobEventChan <- jobEvent
}

// executor 执行完job
// 推送任务结果到 scheduler
func (scheduler *Scheduler)PushJobResult(result *common.JobExecuteResult){
	scheduler.jobResultChan <- result
}



var (
	G_scheduler *Scheduler
)

// 调度器初始化
func InitScheduler()(err error){

	G_scheduler = &Scheduler{
		jobEventChan:make(chan *common.JobEvent,1000),
		jobResultChan:make(chan *common.JobExecuteResult,1000),
		jobPlanTable:make(map[string]*common.SchedulerPlan),
		jobExecutingTable:make(map[string]*common.JobExecuteInfo),
	}

	// 启动循环检测
	go G_scheduler.schedulerLoop()
	return
}
