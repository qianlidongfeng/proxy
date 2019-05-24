package proxyserver

import (
	"github.com/qianlidongfeng/httpserver"
	"github.com/qianlidongfeng/loger"
	"github.com/qianlidongfeng/loger/netloger"
	"github.com/qianlidongfeng/toolbox"
	syslog "log"
	"os"
)

type ProxyServer struct{
	HttpServer httpserver.HttpServer
	cfg Config
	stdout *os.File
}

func NewProxyServer() ProxyServer{
	return ProxyServer{}
}

func (this *ProxyServer) Run() error{
	configFile,err := toolbox.GetConfigFile()
	if err != nil{
		return err
	}
	err=toolbox.LoadConfig(configFile,&this.cfg)
	if err != nil{
		return err
	}
	if this.cfg.Debug == false{
		this.stdout,err=toolbox.RediRectOutPutToLog()
		if err != nil{
			syslog.Fatal(err)
		}
	}
	if this.cfg.Log.LogType == "netlog" && this.cfg.Debug==false{
		lg:=netloger.NewSqloger()
		err=lg.Init(this.cfg.Log.DB)
		if err != nil{
			return err
		}
		log=lg
	}else{
		log=loger.NewLocalLoger()
	}
	db,err= toolbox.InitMysql(this.cfg.DB)
	if err != nil{
		log.Fatal(err)
	}
	proxieTable = this.cfg.DB.Table
	this.HttpServer=httpserver.NewHttpServer()
	this.HttpServer.Bind("/httpproxies",HttpProxies)
	return this.HttpServer.Run(this.cfg.HttpServer)
}

func (this *ProxyServer) Realese(){
	this.stdout.Close()
	log.Close()
	db.Close()
}