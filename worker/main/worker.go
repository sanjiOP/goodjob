package main

import (
	"runtime"
	"fmt"
	"flag"
	"time"
	"github.com/sanjiOP/goodjob/worker"
)


var (
	confFile string
)


// 初始化命令行参数
func initArgs(){

	// ./worker -config ./worker.json
	flag.StringVar(&confFile,"config","./worker.json","指定worker.json")
	flag.Parse()

}

// 初始化环境信息
func initEnv(){
	// 定义线程数量跟cpu数量一致
	runtime.GOMAXPROCS(runtime.NumCPU())
}



//
func main(){

	var(
		err error
	)

	// 1 初始化命令行参数
	initArgs()

	// 2 初始化环境
	initEnv()

	// 3 初始化配置
	if err = worker.InitConfig(confFile); err != nil{
		goto ERR
	}

	// 4 启动日志协程
	if err = worker.InitLogSink();err != nil{
		goto ERR
	}

	// 5 服务注册
	if err = worker.InitRegister();err != nil{
		goto ERR
	}

	// 6 启动执行器
	if err = worker.InitExecutor();err != nil{
		goto ERR
	}

	// 7 启动调度器
	if err = worker.InitScheduler();err != nil{
		goto ERR
	}

	// 8 任务管理器
	if err = worker.InitJobManage();err != nil{
		goto ERR
	}


	for{
		time.Sleep(1 * time.Second)
	}

	return
ERR:
	fmt.Println(err)
}
