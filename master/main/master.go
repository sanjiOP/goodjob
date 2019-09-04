package main

import (
	"runtime"
	"fmt"
	"github.com/sanjiOP/goodjob/master"
	"flag"
	"time"
)


var (
	confFile string
)


// 初始化命令行参数
func initArgs(){

	// ./master -config ./master.json
	flag.StringVar(&confFile,"config","./master.json","指定master.json")
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
	if err = master.InitConfig(confFile); err != nil{
		goto ERR
	}

	// 4 初始化work节点发现
	if err = master.InitWorkManage();err != nil{
		goto ERR
	}

	// 5 日志管理器
	if err = master.InitLogManage();err != nil{
		goto ERR
	}

	// 6 任务管理器
	if err = master.InitJobManage();err != nil{
		goto ERR
	}

	// 7 启动http服务
	if err = master.InitApiServer();err != nil{
		goto ERR
	}

	for{
		time.Sleep(1 * time.Second)
	}

	return
ERR:
	fmt.Println(err)
}
