package common

import (
	"encoding/json"
	"strings"
	"github.com/gorhill/cronexpr"
	"time"
	"context"
)

// 任务
type Job struct {

	// 任务名称
	Name string `json:"name"`
	// 任务脚本命令
	Command string `json:"command"`
	// 周期表达式
	CronExpr string `json:"cronExpr"`

}

// 任务事件
type JobEvent struct {

	// save | delete
	EventType int
	Job *Job
}

// 任务调度计划
type SchedulerPlan struct {
	Job *Job					// 任务信息
	Expr *cronexpr.Expression	// 解析好的表达式
	NextTime time.Time			// 任务下次实行时间

}

// 任务执行状态
type JobExecuteInfo struct {
	Job *Job
	PlanTime time.Time // 理论的调度时间
	RealTime time.Time // 实际的执行时间
	CancelCtx context.Context		// command的上下文
	CancelFunc context.CancelFunc	// 用于取消 command 执行的通知函数
}

// 任务执行结果
type JobExecuteResult struct{
	JobExecuteInfo *JobExecuteInfo	// 执行状态
	Output []byte					// shell命名输出
	Err error						// shell命名错误原因
	StartTime time.Time				// shell启动时间
	EndTime time.Time				// shell 结束时间
}


// 任务执行日志
type JobLogRecord struct {
	JobName string `json:"jobName" bson:"jobName"`
	Command string `json:"command" bson:"command"`
	Err string `json:"err" bson:"err"`
	Output string `json:"output" bson:"output"`
	PlanTime int64 `json:"planTime" bson:"planTime"`				// 任务计划开始时间（cron表达式计算出来的时间）
	SchedulerTime int64 `json:"schedulerTime" bson:"schedulerTime"`		// 任务调度时间（如果任务执行时间长于调度时间，中间的某些调度会被忽略，时间不一致）
	StartTime int64 `json:"startTime" bson:"startTime"`				// 任务开始时间
	EndTime int64 `json:"endTime" bson:"endTime"`					// 任务结束时间
}


// 日志批次，用于多条日志的插入
type LogBatch struct {
	// 多条日志记录
	Logs []interface{}
}


// 日志列表查询条件
type LogListFilter struct {
	JobName string `bson:"jobName"`
}


// 日志列表排序规则
type LogListSort struct {
	SortBy int `bson:"startTime"`
}


// http 接口响应结构
type Response struct {
	Code int `json:"code"`
	Msg string `json:"msg"`
	Info interface{} `json:"info"`
}

// 成功响应
func SuccessResponse(info interface{})(resp []byte, err error){
	var(
		response Response
	)
	response = Response{
		Code:0,
		Msg:"请求成功",
		Info:info,
	}
	resp,err = json.Marshal(response)
	return
}

// 错误响应
func ErrorResponse(code int,msg string)(resp []byte, err error){
	var(
		response Response
	)
	response = Response{
		Code:code,
		Msg:msg,
	}
	resp,err = json.Marshal(response)
	return
}


// 返序列化 job任务
func UnpackJob(value []byte)(job *Job,err error){
	var(
		varJob *Job
	)
	varJob = &Job{}
	if err = json.Unmarshal(value,varJob);err != nil{
		return
	}
	job = varJob
	return
}


// 从etcd的key中提取任务名
// /cron/jobs/job10  -> job10
func ExtractJobName(jobKey string)(jobName string){
	return strings.TrimPrefix(jobKey,PREFIX_JOB_DIR)
}


// 从etcd的key中提取任务名
// /cron/kill/job10  -> job10
func ExtractKillerName(killerKey string)(jobName string){
	return strings.TrimPrefix(killerKey,PREFIX_JOB_KILL_DIR)
}

// 提取work的ip地址
// /cron/works/192.168.0.1
func ExtractWorkIp(workKey string)(addr string){
	return strings.TrimPrefix(workKey,PREFIX_WORK_NODE_DIR)
}


// 构建jobEvent
func JobEventFactory(eventType int,job *Job)(jobEvent *JobEvent)  {
	return &JobEvent{eventType,job}
}


// 构建任务执行计划
func SchedulerPlanFactory(job *Job)(schedulerPlan *SchedulerPlan,err error){

	var(
		expr *cronexpr.Expression
	)
	// 解析cron表达式
	if expr,err = cronexpr.Parse(job.CronExpr);err != nil{
		return
	}

	schedulerPlan = &SchedulerPlan{
		Job:job,
		Expr:expr,
		NextTime:expr.Next(time.Now()),
	}

	return
}


// 构建任务执行状态
func JobExecuteInfoFactory(schedulerPlan *SchedulerPlan)(jobExecuteInfo *JobExecuteInfo){
	jobExecuteInfo = &JobExecuteInfo{
		Job:schedulerPlan.Job,
		PlanTime:schedulerPlan.NextTime,
		RealTime:time.Now(),
	}
	jobExecuteInfo.CancelCtx,jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}






