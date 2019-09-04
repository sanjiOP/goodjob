package worker

import (
	clientv3 "go.etcd.io/etcd/clientv3"
	"time"
	"net"
	"github.com/sanjiOP/goodjob/common"
	"context"
)

// 服务注册
// /cron/works/{localIp}
type Register struct{

	//etcd
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease

	// 本机ip
	localIp string
}


// 注册服务
// 注册到/cron/works/{localIp},并自动续租
func (register *Register)KeepAlive(){

	var (
		err error
		registerKey string
		leaseResponse *clientv3.LeaseGrantResponse
		keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResponse *clientv3.LeaseKeepAliveResponse
		//putResponse *clientv3.PutResponse
		//
		cancelCtx context.Context
		cancelFunc context.CancelFunc
	)
	//
	registerKey = common.PREFIX_WORK_NODE_DIR + register.localIp


	for {

		cancelFunc = nil

		// 租约
		if leaseResponse,err = register.lease.Grant(context.TODO(),10);err != nil{
			goto RETRY
		}

		// 自动续租
		if keepAliveChan ,err = register.lease.KeepAlive(context.TODO(),leaseResponse.ID);err != nil{
			goto RETRY
		}

		// 注册服务
		cancelCtx,cancelFunc = context.WithCancel(context.TODO())
		if _,err = register.kv.Put(cancelCtx,registerKey,"",clientv3.WithLease(leaseResponse.ID));err != nil{
			goto RETRY
		}


		// 处理续租的应答
		for{
			select {
			case keepAliveResponse = <- keepAliveChan:
				if keepAliveResponse == nil{
					// 续租失败
					goto RETRY
				}
			}
		}
		
		RETRY:
			time.Sleep(1)
			if cancelFunc != nil{
				cancelFunc()
			}

	}



}



var (
	G_register *Register
)

// 初始化
func InitRegister()(err error){

	var(
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
		localIp string
	)

	// 定义配置项
	config = clientv3.Config{
		// 集群地址
		Endpoints:G_config.EtcdEndPoints,
		// 超时
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	// 创建etcd连接
	if client,err = clientv3.New(config);err != nil {
		return
	}

	// 赋值
	kv		= clientv3.NewKV(client)
	lease	= clientv3.NewLease(client)
	if localIp,err = getLocalIp();err != nil{
		return
	}
	G_register = &Register{client:client,kv:kv,lease:lease,localIp:localIp}

	// 注册服务
	go G_register.KeepAlive()

	return
}


// 获取本机ip
func getLocalIp()(ipv4 string,err error){

	// 遍历所有网卡
	var (
		addrList []net.Addr
		addr net.Addr
		ipNet *net.IPNet
		isIpNet bool
	)
	if addrList,err = net.InterfaceAddrs();err != nil{
		return
	}

	// 取第一个非localhost（非环回地址）网卡的ip
	for _,addr = range addrList{
		// unix socket | ipv4 | ipv6
		// 需要反解
		if ipNet,isIpNet = addr.(*net.IPNet);isIpNet && !ipNet.IP.IsLoopback(){

			// 跳过ipv6
			if ipNet.IP.To4() != nil{
				ipv4 = ipNet.IP.String()// 192.168.0.1
				return
			}

		}
	}
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}
