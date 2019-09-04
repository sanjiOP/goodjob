package worker

import (
	"io/ioutil"
	"encoding/json"
)

type Config struct {
	// ==etcd配置
	// 集群
	EtcdEndPoints []string `json:"etcdEndPoints"`
	// etcd连接超时
	EtcdDialTimeout int `json:"etcdDialTimeout"`
}


var (
	G_config *Config
)


// 初始化配置信息
func InitConfig(filePath string) (err error) {

	var (
		content []byte
		conf Config
	)

	if content,err = ioutil.ReadFile(filePath); err != nil {
		return err
	}


	// json反序列化
	if err = json.Unmarshal(content,&conf);err != nil{
		return err
	}

	// 单例赋值
	G_config = &conf
	return
}

