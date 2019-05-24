package checker

import (
	"github.com/qianlidongfeng/toolbox"
	"time"
)

type LogConfig struct{
	LogType string
	DB toolbox.MySqlConfig
}

type Config struct{
	Thread int
	Debug bool
	BadSuccessrate int
	BadExist int
	RpcClientServer []string
	RpcTimeout time.Duration
	DB toolbox.MySqlConfig
	Loger LogConfig
}