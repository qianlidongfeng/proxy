package clientserver

import (
	"errors"
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/toolbox"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"time"
)
var(
	cfg Config
	limit chan struct{}
)

type Config struct{
	TimeOut time.Duration
	Port string
	CheckUrl string
	MaxHttpClient int
}

type HttpGeter struct{
}

func (this *HttpGeter) Get(ctx context.Context, in *ProxyInfo)(*Respone, error){
	limit<-struct{}{}
	defer func(){<-limit}()
	resp:= Respone{}
	client:=httpclient.NewHttpClient()
	client.SetTimeOut(cfg.TimeOut*time.Millisecond)
	if in.Type=="http"{
		client.SetHttpProxy(in.Proxy)
	}else if in.Type=="sock5"{
		client.SetSock5Proxy(in.Proxy)
	}else{
		return &resp,errors.New(fmt.Sprintf("invalid proxy type:%s",in.Type))
	}
	r,err:=client.Get(cfg.CheckUrl)
	if err != nil{
		return &resp,err
	}
	resp.Content=[]byte(r.Html)
	resp.Status=int32(r.StatusCode)
	return &resp,err
}


func Start(){
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
	limit=make(chan struct{},cfg.MaxHttpClient)
	list, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()
	RegisterHttpServer(server, &HttpGeter{})
	fmt.Println("grpc 服务启动... ...")
	server.Serve(list)
}