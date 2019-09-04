package master

import (
	"net/http"
	"net"
	"time"
	"strconv"
	"github.com/sanjiOP/goodjob/common"
	"encoding/json"
	"fmt"
)



//
type ApiServer struct {
	httpServer *http.Server
}

// route /job/save
// param job = {"name":"echo","command":"echo hello","CronExpr":"* * * * * *"}
func handleJobSave(w http.ResponseWriter,r *http.Request){

	var (
		err error
		postJob string
		job common.Job
		oldJob *common.Job
		responseContent []byte
	)

	// 1:解析post表单
	if err = r.ParseForm();err != nil{
		goto ERR
	}

	// 2:将表单数据转成job结构体
	postJob = r.PostForm.Get("job")
	if err = json.Unmarshal([]byte(postJob),&job);err != nil{
		fmt.Println("错误1：",err)
		goto ERR
	}


	// 3:保存到etcd
	if oldJob,err = G_jobManage.SaveJob(&job);err != nil{
		fmt.Println("错误2：",err)
		goto ERR
	}


	// 4:正常响应
	if responseContent,err = common.SuccessResponse(oldJob);err == nil {
		w.Write(responseContent)
	}

	return
ERR:
	// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}
}



// route /job/delete
// param name = job1
func handleJobDelete(w http.ResponseWriter,r *http.Request){
	var (
		err error
		jobName string
		oldJob *common.Job
		responseContent []byte
	)

	// 1 解析表单
	if err = r.ParseForm(); err != nil{
		goto ERR
	}
	// 2 根据任务名称删除任务
	jobName = r.PostForm.Get("name")
	if oldJob,err = G_jobManage.DeleteJob(jobName);err != nil{
		goto ERR
	}


	// 3 正常响应
	if responseContent,err = common.SuccessResponse(oldJob);err == nil {
		w.Write(responseContent)
	}

	return
ERR:

	// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}

}


// route /job/list
// param
func handleJobList(w http.ResponseWriter,r *http.Request){

	var (
		err error
		jobList []*common.Job
		responseContent []byte
	)

	// 获取列表
	if jobList,err = G_jobManage.ListJob();err != nil{
		goto ERR
	}

	// 3 正常响应
	if responseContent,err = common.SuccessResponse(jobList);err == nil {
		w.Write(responseContent)
	}

	return
ERR:

	// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}
}



// route /job/kill
// param name=job1
func handleJobKill(w http.ResponseWriter,r *http.Request){

	var(
		err error
		jobName string
		responseContent []byte
	)

	// 1 解析表单
	if err = r.ParseForm(); err != nil{
		goto ERR
	}
	// 2 根据任务名称删除任务
	jobName = r.PostForm.Get("name")
	if err = G_jobManage.KillJob(jobName);err != nil{
		goto ERR
	}

	// 3 正常响应
	if responseContent,err = common.SuccessResponse(nil);err == nil {
		w.Write(responseContent)
	}

	return
ERR:

// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}

}


// route /job/log
// param name=job1
func handleJobLog(w http.ResponseWriter,r *http.Request){

	var(
		err error
		jobName string
		page int
		pageSize int
		responseContent []byte
		list []*common.JobLogRecord
	)

	// 1 解析表单
	if err = r.ParseForm(); err != nil{
		goto ERR
	}
	jobName		= r.Form.Get("jobName")
	if page,err	= strconv.Atoi(r.Form.Get("page"));err != nil{
		page = 0
	}
	if pageSize,err	= strconv.Atoi(r.Form.Get("page_size"));err != nil{
		pageSize = 20
	}

	// 2 获取日志内容
	if list,err = G_logManage.lists(jobName,int64(page),int64(pageSize));err != nil{
		goto ERR
	}

	// 3 正常响应
	if responseContent,err = common.SuccessResponse(list);err == nil {
		w.Write(responseContent)
	}

	return
ERR:

// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}
}



// route /work/list
func handleWorkList(w http.ResponseWriter,_ *http.Request){

	var(
		err error
		responseContent []byte
		list []string
	)

	// 1 获取节点列表
	if list,err = G_workManage.list();err != nil{
		goto ERR
	}

	// 2 正常响应
	if responseContent,err = common.SuccessResponse(list);err == nil {
		w.Write(responseContent)
	}

	return
ERR:

// 错误响应
	if responseContent,err = common.ErrorResponse(-1,err.Error());err == nil {
		w.Write(responseContent)
	}

}




// 单例
var(
	G_ApiServer * ApiServer
)

// 初始化
func InitApiServer() (err error){

	var(
		mux *http.ServeMux
		lister net.Listener
		httpServer *http.Server
		staticDir http.Dir
		staticHandler http.Handler
	)

	// 动态接口路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save",handleJobSave)
	mux.HandleFunc("/job/delete",handleJobDelete)
	mux.HandleFunc("/job/list",handleJobList)
	mux.HandleFunc("/job/kill",handleJobKill)
	mux.HandleFunc("/job/log",handleJobLog)
	mux.HandleFunc("/work/list",handleWorkList)

	// 静态页面
	staticDir		= http.Dir(G_config.WebRoot)
	staticHandler	= http.FileServer(staticDir)
	mux.Handle("/",http.StripPrefix("/",staticHandler))

	// tcp监听
	if lister,err = net.Listen("tcp",":"+strconv.Itoa(G_config.ApiPort));err != nil{
		return err
	}

	// 创建http服务
	httpServer = &http.Server{
		ReadTimeout		: time.Duration(G_config.ReadTimeOut) * time.Millisecond,
		WriteTimeout	: time.Duration(G_config.WriteTimeOut) * time.Millisecond,
		Handler			: mux,
	}

	// 赋值单例
	G_ApiServer = &ApiServer{
		httpServer:httpServer,
	}

	// 启动服务
	go httpServer.Serve(lister)

	return
}
