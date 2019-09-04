package master

import (
	"io/ioutil"
	"encoding/json"
)

type Config struct {

	// ==http 配置
	// 端口
	ApiPort int `json:"apiPort"`
	// 读超时
	ReadTimeOut int `json:"readTimeOut"`
	// 写超时
	WriteTimeOut int `json:"writeTimeOut"`
	// http服务器，静态页面跟目录
	WebRoot string `json:"webRoot"`

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

