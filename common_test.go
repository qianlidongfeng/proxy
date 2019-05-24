package proxy

import (
	"fmt"
	"github.com/qianlidongfeng/httpclient"
	"github.com/qianlidongfeng/toolbox"
	"testing"
	"time"
)



func TestLambda(b *testing.T) {
	total:=1000000000
	t:=time.Now()
	for i:=0;i<total;i++{
		func(index int) int{
			return i+1
		}(i)
	}
	fmt.Println("耗时:",time.Since(t))
	t=time.Now()
	for i:=0;i<total;i++{
		get(i)
	}
	fmt.Println("耗时:",time.Since(t))
}

func get(index int) int{
	return index+1
}

func TestGetIpFrom138(t *testing.T) {
	c:=httpclient.NewHttpClient()
	r,_:=c.Get("http://2019.ip138.com/ic.asp")
	h,_:=toolbox.GbkToUtf8(r.Html)
	GetIpFrom138(h)
}
