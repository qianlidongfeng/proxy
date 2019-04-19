package checker

import (
	"github.com/qianlidongfeng/loger/netloger"
	"github.com/qianlidongfeng/toolbox"
	"time"
)

type LogConfig struct{
	LogType string
	DB netloger.SqlConfig
}

type Config struct{
	Thread int
	TimeOut time.Duration
	Debug bool
	Limit int
	DB toolbox.MySqlConfig
	Loger LogConfig
}