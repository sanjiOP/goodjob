package common


const (

	// etcd 任务前缀
	PREFIX_JOB_DIR = "/cron/jobs/"

	// 结束任务前缀
	PREFIX_JOB_KILL_DIR = "/cron/killer/"

	// 任务锁前缀
	PREFIX_JOB_LOCK_DIR = "/cron/lock/"

	// 服务注册目录
	PREFIX_WORK_NODE_DIR = "/cron/works/"


	//任务事件类型定义
	JOB_EVENT_SAVE = 1
	JOB_EVENT_DELETE = 2
	JOB_EVENT_KILL = 3
)
