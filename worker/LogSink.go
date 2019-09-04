package worker

import (
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/sanjiOP/goodjob/common"
	"go.mongodb.org/mongo-driver/mongo/options"
	"context"
	"time"
)


// mongodb 日志记录
type LogSink struct {

	// mongodb 客户端
	client *mongo.Client

	// 日志表
	collection *mongo.Collection

	// 日志channel
	logChan chan *common.JobLogRecord

	// 日志定时自动提交（未满足批次阈值的情况下自动提价）
	autoCommitChan chan *common.LogBatch
}


// 保存日志
// 不用处理错误情况
func (logSink *LogSink)saveLogs(logBatch *common.LogBatch){

	var (
		_ *mongo.InsertManyResult
		err error
	)

	// 写入mongodb数据库
	if _,err = logSink.collection.InsertMany(context.TODO(),logBatch.Logs);err != nil{
		//fmt.Println("mongo inster error : ",err.Error())
	}
	//fmt.Println("mongo inster success",result.InsertedIDs)
}


// 发送日志
func (logSink *LogSink)Append(logRecord *common.JobLogRecord){

	select {
	// channel 满了则进入default分支（默认丢弃）
	case logSink.logChan <- logRecord:
	default:
		// 队列满了则丢弃
	}
}


// 日志存储协程
func (logSink *LogSink)WriteLoop(){

	var(
		logRecord *common.JobLogRecord
		logBatch *common.LogBatch
		commitTimer *time.Timer
		timeOutBatch *common.LogBatch
	)


	for{
		select {

		// 调度器发送日志记录 channel 的通知
		case logRecord = <- logSink.logChan:

			if logBatch == nil{
				logBatch = &common.LogBatch{}
				// 在满足超时后，日志自动提交
				commitTimer = time.AfterFunc(1000 * time.Millisecond, func(logBatch *common.LogBatch)(func()) {
					// 此回调函数在新的协程中执行
					// 不建议直接提交日志，会产生数据冲突
					// 发出超时通知，让WriteLoop执行保存日志
					// logBatch的指针会变，
					return func() {
						logSink.autoCommitChan <- logBatch
					}

				}(logBatch))
			}

			// 缓存日志，批量写入
			logBatch.Logs = append(logBatch.Logs,logRecord)
			if len(logBatch.Logs) >= 5{
				logSink.saveLogs(logBatch)

				// 清空
				logBatch = nil
				commitTimer.Stop()
			}


		//
		case timeOutBatch = <- logSink.autoCommitChan:
			// 判断超时的日志批次是否为当前的日志批次
			if timeOutBatch != logBatch{
				// 跳过已提交的批次
				continue
			}
			logSink.saveLogs(timeOutBatch)
			logBatch = nil
		}
	}

}



var (
	G_logSink *LogSink
)

func InitLogSink()(err error){

	var (
		client *mongo.Client
		ctx context.Context
	)

	// 创建mongodb链接客户端
	ctx, _ = context.WithTimeout(context.Background(), 10 * time.Second)
	if client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://127.0.0.1:27017"));err != nil{
		return
	}
	// 全局变量赋值
	G_logSink = &LogSink{
		client : client,
		collection : client.Database("cron").Collection("job"),
		logChan : make(chan *common.JobLogRecord,1000),
		autoCommitChan:make(chan *common.LogBatch,1000),
	}

	// 启动写入协程
	go G_logSink.WriteLoop()
	return

}
