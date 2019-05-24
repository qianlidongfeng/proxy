package proxy

import (
	"errors"
	"github.com/qianlidongfeng/toolbox"
	"time"
)

type Info struct{
	IP string
	Area string
}

func GetIpFrom138(html string)(info Info,err error){
	defer func() {
		if e := recover(); e != nil {
			err = errors.New("getip failed")
			return
		}
	}()
	minfo := toolbox.GetMapByReg(`(?U).+\[(?P<ip>.+)\].+：(?P<area>.+)( |\<)`,html)
	if _,ok:=minfo["ip"];!ok{
		time.Sleep(10*time.Second)
		err = errors.New("get ip failed")
		return
	}
	if _,ok:=minfo["area"];!ok{
		time.Sleep(10*time.Second)
		err = errors.New("get area failed")
		return
	}
	info.IP=minfo["ip"]
	info.Area=minfo["area"]
	return
}