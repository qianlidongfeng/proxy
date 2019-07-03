package proxy

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/qianlidongfeng/toolbox"
	"strings"
	"time"
)

type Info struct{
	IP string
	Area string
}

func GetIpFrom138(html string)(info Info,err error){
	minfo,err:= toolbox.GetMapByReg(`(?U).+\[(?P<ip>.+)\].+：(?P<area>.+)( |\<)`,html)
	if err != nil{
		return
	}
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

func GetIpFromBaidu(html string)(info Info,err error){
	doc,err:=goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil{
		return
	}
	s := doc.Find(`#\31 > div.c-border > div:nth-child(2) > div.c-span21.c-span-last.op-ip-detail > table > tbody > tr > td`).Text()
	if s==""{
		err = errors.New("getip failed")
		return
	}
	minfo ,err:= toolbox.GetMapByReg(`.+: (?P<ip>(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})(\.(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})){3})(?U)(?P<area>.+)( |\t).*`,s)
	if err != nil{
		return
	}
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

func GetIpFromCz(html string) (info Info,err error){
	begin:=strings.Index(html,"(")
	end:=strings.LastIndex(html,")")
	if begin==-1||end==-1{
		err=errors.New("get selfip failed,error flag begin end")
		return
	}
	content:=strings.Split(html[begin+1:end],",")
	if len(content)!=3{
		err=errors.New("get selfip failed,error info")
		return
	}
	info.IP=content[0]
	areaInfo,_:=toolbox.GbkToUtf8(content[1])
	areaInfo=strings.TrimLeft(areaInfo,"'")
	areaInfo=strings.TrimRight(areaInfo,"'")
	area:= strings.Split(areaInfo," ")
	info.Area=area[0]
	return
}
