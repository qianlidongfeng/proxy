package proxyserver

import (
	"github.com/qianlidongfeng/httpserver"
	"github.com/qianlidongfeng/loger"
	"github.com/qianlidongfeng/toolbox"
)


type Config struct{
	Debug bool
	Log loger.Config
	HttpServer httpserver.Config
	DB toolbox.MySqlConfig
}
