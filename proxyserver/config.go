package proxyserver

import (
	"github.com/qianlidongfeng/httpserver"
	"github.com/qianlidongfeng/toolbox"
)

type LogConfig struct{
	LogType string
	DB toolbox.MySqlConfig
}



type Config struct{
	Debug bool
	Log LogConfig
	HttpServer httpserver.Config
	DB toolbox.MySqlConfig
}
