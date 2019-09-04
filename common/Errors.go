package common

import "errors"

var(
	ERR_LOCK_ALREADY_REQUIRED = errors.New("锁被占用")


	ERR_NO_LOCAL_IP_FOUND = errors.New("work节点没有物理网卡")
)
