package clientserver

import (
	"errors"
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/loger"
	"github.com/qianlidongfeng/loger/netloger"
	"github.com/qianlidongfeng/toolbox"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"time"
)
var(
	cfg Config
	log loger.Loger
	limit chan struct{}
)

type LogConfig struct{
	LogType string
	DB toolbox.MySqlConfig
}

type Config struct{
	TimeOut time.Duration
	Port string
	CheckUrl string
	MaxHttpClient int
	Log LogConfig
	Debug bool
}

type HttpGeter struct{
}

func (this *HttpGeter) Get(ctx context.Context, in *ProxyInfo)(*Respone, error){
	limit<-struct{}{}
	defer func(){
		<-limit
	}()
	resp:= Respone{}
	client:=httpclient.NewClient()
	client.SetTimeOut(cfg.TimeOut*time.Millisecond)
	client.DisableRedirect()
	err:=client.SetProxy(in.Proxy)
	if err != nil{
		return nil,err
	}
	var url string
	if in.Url != ""{
		url = in.Url
	}else{
		url = cfg.CheckUrl
	}
	r,err:=client.Get(url)
	if err != nil{
		return &resp,err
	}
	resp.Content=[]byte(r.Html)
	resp.Status=int32(r.StatusCode)
	return &resp,err
}

func Init(){
	appPath,err:=toolbox.AppPath()
	if err != nil{
		panic(err)
	}
	configFile:=appPath+".yaml"
	cfg = Config{}
	err=toolbox.LoadConfig(configFile,&cfg)
	if err != nil{
		panic(err)
	}
	if cfg.Debug == false{
		_,err:=toolbox.RediRectOutPutToLog()
		if err != nil{
			panic(err)
		}
	}
	if cfg.Log.LogType == "netlog"{
		lg:=netloger.NewSqloger()
		err=lg.Init(cfg.Log.DB)
		if err != nil{
			panic(err)
		}
		log=lg
	}else{
		log=loger.NewLocalLoger()
	}
	limit = make(chan struct{},cfg.MaxHttpClient)
}


func Start(){
	Init()
	defer func() {
		if e := recover(); e != nil {
			err := errors.New(e.(string))
			log.Fatal(err)
			return
		}
	}()
	server := grpc.NewServer()
	RegisterHttpServer(server, &HttpGeter{})
	fmt.Println("grpc 服务启动... ...")
	list, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		panic(err)
	}
	server.Serve(list)
}