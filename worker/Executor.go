package worker

import (
	"github.com/sanjiOP/goodjob/common"
	"os/exec"
	"time"
	"math/rand"
)

// 任务执行器
type Executor struct {}


// 执行任务
func (executor *Executor)ExecuteJob(info *common.JobExecuteInfo){

	// 执行 shell command
	go func() {

		var (
			command *exec.Cmd
			err error
			output []byte
			result *common.JobExecuteResult
			jobLock *JobLock
		)

		result = &common.JobExecuteResult{
			JobExecuteInfo:info,
			Output:make([]byte,0),
		}
		result.StartTime = time.Now()


		// 初始化 锁
		jobLock = G_jobManage.CreateJobLock(info.Job.Name)
		//随机睡眠（0-1s）
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.UnLock()

		if !jobLock.IsLocked {
			result.Err = err
			result.EndTime = time.Now()
		}else{

			// 忽略掉上锁的网络时间
			result.StartTime = time.Now()

			// 执行shell命令
			command = exec.CommandContext(info.CancelCtx,"/bin/bash","-c",info.Job.Command)
			output,err = command.CombinedOutput()

			// 任务执行完，需要把结果返回给 scheduler,scheduler需要从jobExecutingTable中删除该任务
			result.Output		= output
			result.Err			= err
			result.EndTime		= time.Now()
		}

		G_scheduler.PushJobResult(result)


	}()
}



var (
	G_executor *Executor
)
// 初始化
func InitExecutor()(err error){
	G_executor = &Executor{

	}
	return
}
