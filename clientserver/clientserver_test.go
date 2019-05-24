package clientserver

import (
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/toolbox"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestMyServer_SayHello(t *testing.T) {
	conn, err := grpc.Dial("127.0.0.1:9527", grpc.WithInsecure())
	if err != nil{
		fmt.Println(err)
	}
	// 当请求完毕后记得关闭连接,否则大量连接会占用资源
	defer conn.Close()

	// 创建grpc客户端
	c := NewGreeterClient(conn)
	h:= qianlidongfeng.NewHttpClient(conn)
	resp,err:=h.Get(context.Background(),&qianlidongfeng.ProxyInfo{Proxy: "http://118.190.94.254:9001",Type:"http"})
	html,err:= toolbox.GbkToUtf8(string(resp.Content))
	fmt.Println(html)
	name := "我是客户端,正在请求服务端!!!"
	// 客户端向grpc服务端发起请求
	result, err := c.SayHello(context.Background(), &HelloRequest{Name:name})
	fmt.Println(name)
	if err != nil{
		fmt.Println("请求失败!!!")
		return
	}
	// 获取服务端返回的结果
	fmt.Println(result.Message)
}

func TestStart(t *testing.T) {
	client,_:=httpclient.NewHttpClient()
	client.SetTimeOut(2000*time.Millisecond)
	client.SetSocksProxy("110.52.211.28:9999")
	h,e:=client.Get("http://2019.ip138.com/ic.asp")
	_,_=h,e
}
