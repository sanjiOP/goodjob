package master

import (
	"go.mongodb.org/mongo-driver/mongo"
	"context"
	"time"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/sanjiOP/goodjob/common"
)

// 日志管理对象
type LogManage struct{
	// mongodb 客户端
	client *mongo.Client

	// 日志表
	collection *mongo.Collection
}


// 日志列表
func (logManage *LogManage)lists(jobName string,skip int64,limit int64)(list []*common.JobLogRecord,err error)  {

	var(
		logListFilter *common.LogListFilter
		logListSort *common.LogListSort
		findOptions *options.FindOptions
		cursor *mongo.Cursor
		logRecord *common.JobLogRecord
	)

	// 分配地址
	list = make([]*common.JobLogRecord,0)

	// 过滤条件
	logListFilter = &common.LogListFilter{
		JobName:jobName,
	}

	// 排序(-1 按照 startTime 倒排)
	logListSort = &common.LogListSort{
		SortBy:-1,
	}

	// 查询可选项
	findOptions = &options.FindOptions{
		Skip : &skip,
		Limit : &limit,
		Sort : logListSort,
	}


	// 查询
	if cursor,err = logManage.collection.Find(context.TODO(),logListFilter,findOptions);err != nil{
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		logRecord = &common.JobLogRecord{}
		if err = cursor.Decode(logRecord);err != nil{
			// json反解失败，跳过
			continue
		}
		list = append(list,logRecord)
	}

	return
}


var (
	G_logManage *LogManage
)

// 初始化
func InitLogManage()(err error){


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
	G_logManage = &LogManage{
		client : client,
		collection:client.Database("cron").Collection("job"),
	}

	return
}
