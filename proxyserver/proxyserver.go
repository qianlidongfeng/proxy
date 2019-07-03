package proxyserver

import (
	"github.com/qianlidongfeng/httpserver"
	"github.com/qianlidongfeng/loger"
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
	log,err:=loger.NewLoger(this.cfg.Log)
	if err != nil{
		return err
	}
	db,err= toolbox.InitMysql(this.cfg.DB)
	if err != nil{
		log.Fatal(err)
	}
	proxieTable = this.cfg.DB.Table
	this.HttpServer=httpserver.NewHttpServer()
	this.HttpServer.Bind("/mycollectionhttpproxies",MyCollectionHttpProxies)
	this.HttpServer.Bind("/mycollectionsock5proxies",MyCollectionSock5Proxies)
	this.HttpServer.Bind("/myscanhttpproxies",MyScanHttpProxies)
	this.HttpServer.Bind("/myscansock5proxies",MyScanSock5Proxies)
	this.HttpServer.Bind("/myproxies",MyProxies)
	return this.HttpServer.Run(this.cfg.HttpServer)
}

func (this *ProxyServer) Realese(){
	this.stdout.Close()
	log.Close()
	db.Close()
}