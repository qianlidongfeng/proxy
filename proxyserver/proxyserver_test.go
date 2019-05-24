package proxyserver

import (
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"testing"
)

func TestServer_Run(t *testing.T) {
	server:= NewProxyServer()
	err:=server.Run()
	if err !=nil{
		t.Error(err)
	}
	server.Realese()
}

func BenchmarkHttpProxies(b *testing.B) {
	client:=httpclient.NewHttpClient()
	for i:=0;i<b.N;i++{
		resp,err:=client.Get(`http://127.0.0.1:8080/httpproxies?count=10`)
		if err != nil{
			fmt.Println(err)
		}
		_=resp
	}
}
