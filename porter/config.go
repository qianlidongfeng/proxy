package porter

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
	Limit int
	BadSuccessrate int
	BadExist int
	RpcClientServer []string
	RpcTimeout time.Duration
	SrcDB toolbox.MySqlConfig
	DestDB toolbox.MySqlConfig
	Loger LogConfig
}